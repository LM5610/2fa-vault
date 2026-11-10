package vault

import (
	"os"
	"path/filepath"
	"testing"

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

	// 5. Delete Account
	if err := v.DeleteAccount(acc.ID); err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
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
