package vault

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"twofa-vault/pkg/totp"
)

func TestVault_Lifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	vaultPath := filepath.Join(tmpDir, "test_vault.enc")

	v := New(vaultPath)
	if v.IsInitialized() {
		t.Fatal("vault should not be initialized initially")
	}

	masterPass := "SecretVaultPassword!123"

	// 1. Init
	if err := v.Init(masterPass); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if !v.IsInitialized() {
		t.Fatal("vault should be initialized after Init")
	}
	if v.IsLocked() {
		t.Fatal("vault should be unlocked after Init")
	}

	// 2. Add Account
	acc, err := v.AddAccount(Account{
		Issuer:      "GitHub",
		AccountName: "octocat@github.com",
		Secret:      "JBSWY3DPEHPK3PXP",
		Algorithm:   totp.AlgorithmSHA1,
		Digits:      6,
		Period:      30,
		Type:        totp.TypeTOTP,
		Tags:        []string{"Work", "Dev"},
	})
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}
	if acc.ID == "" {
		t.Fatal("expected generated ID")
	}

	// 3. List accounts
	list, err := v.ListAccounts("", "")
	if err != nil {
		t.Fatalf("ListAccounts failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 account, got %d", len(list))
	}
	if list[0].Code == "" || list[0].Code == "ERR" {
		t.Fatalf("expected valid OTP code, got %s", list[0].Code)
	}
	if list[0].Remaining <= 0 {
		t.Fatalf("expected remaining > 0, got %d", list[0].Remaining)
	}

	// 4. Test Lock & Unlock
	v.Lock()
	if !v.IsLocked() {
		t.Fatal("vault should be locked after Lock()")
	}

	_, err = v.ListAccounts("", "")
	if err != ErrVaultLocked {
		t.Fatalf("expected ErrVaultLocked, got %v", err)
	}

	// Unlock with wrong password
	err = v.Unlock("WrongPassword")
	if err == nil {
		t.Fatal("expected error with wrong password")
	}

	// Unlock with correct password
	if err := v.Unlock(masterPass); err != nil {
		t.Fatalf("Unlock with correct password failed: %v", err)
	}
	if v.IsLocked() {
		t.Fatal("vault should be unlocked")
	}

	// 5. Test RecoveryCodes & Pinning & Reorder
	acc2, err := v.AddAccount(Account{
		Issuer:      "Google",
		AccountName: "user@gmail.com",
		Secret:      "JBSWY3DPEHPK3PXP",
		RecoveryCodes: []RecoveryCode{
			{Code: "1111-2222", Used: false},
			{Code: "3333-4444", Used: false},
		},
	})
	if err != nil {
		t.Fatalf("AddAccount 2 failed: %v", err)
	}

	// Toggle recovery code
	toggled, err := v.ToggleRecoveryCode(acc2.ID, 0)
	if err != nil {
		t.Fatalf("ToggleRecoveryCode failed: %v", err)
	}
	if !toggled.RecoveryCodes[0].Used {
		t.Fatal("expected recovery code 0 to be used")
	}

	// Toggle pin
	pinnedAcc, err := v.TogglePin(acc2.ID)
	if err != nil {
		t.Fatalf("TogglePin failed: %v", err)
	}
	if !pinnedAcc.Pinned {
		t.Fatal("expected account to be pinned")
	}

	// 6. Delete Accounts
	if err := v.DeleteAccount(acc.ID); err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}
	if err := v.DeleteAccount(acc2.ID); err != nil {
		t.Fatalf("DeleteAccount 2 failed: %v", err)
	}

	listAfterDel, err := v.ListAccounts("", "")
	if err != nil {
		t.Fatalf("ListAccounts failed: %v", err)
	}
	if len(listAfterDel) != 0 {
		t.Fatalf("expected 0 accounts, got %d", len(listAfterDel))
	}

	_ = os.Remove(vaultPath)
}

func TestVault_SnapshotsAndRollback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "2fa-vault-snapshot-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	vaultPath := filepath.Join(tmpDir, "vault.enc")
	v := New(vaultPath)

	password := "SecretPass123"
	if err := v.Init(password); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	// Add an account
	acc, err := v.AddAccount(Account{
		Issuer:      "GitHub",
		AccountName: "dev-user",
		Secret:      "JBSWY3DPEHPK3PXP",
	})
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	// Verify snapshots were created
	snapshots, err := v.ListSnapshots()
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}
	if len(snapshots) == 0 {
		t.Fatal("expected at least 1 snapshot")
	}

	snapshotWithAccount := snapshots[0].Filename

	// Now delete account
	if err := v.DeleteAccount(acc.ID); err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	listAfterDel, _ := v.ListAccounts("", "")
	if len(listAfterDel) != 0 {
		t.Fatalf("expected 0 accounts, got %d", len(listAfterDel))
	}

	// Rollback to snapshotWithAccount
	if err := v.RollbackSnapshot(snapshotWithAccount); err != nil {
		t.Fatalf("RollbackSnapshot failed: %v", err)
	}

	// Verify account is restored!
	restoredList, err := v.ListAccounts("", "")
	if err != nil {
		t.Fatalf("ListAccounts after rollback failed: %v", err)
	}
	if len(restoredList) != 1 {
		t.Fatalf("expected 1 restored account, got %d", len(restoredList))
	}
	if restoredList[0].Issuer != "GitHub" {
		t.Fatalf("expected restored issuer GitHub, got %s", restoredList[0].Issuer)
	}
}
