package mobile

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"twofa-vault/pkg/totp"
	"twofa-vault/pkg/vault"
)

// Global mobile vault instance for GoMobile iOS / Android bindings
var (
	mobileVault *vault.Vault
	mobileMu    sync.Mutex
)

// InitVault initializes a new vault file with master password on mobile (iOS/Android)
func InitVault(storagePath, password string) error {
	mobileMu.Lock()
	defer mobileMu.Unlock()

	mobileVault = vault.New(storagePath)
	return mobileVault.Init(password)
}

// UnlockVault unlocks an existing vault on mobile
func UnlockVault(storagePath, password string) error {
	mobileMu.Lock()
	defer mobileMu.Unlock()

	mobileVault = vault.New(storagePath)
	return mobileVault.Unlock(password)
}

// LockVault locks the mobile vault
func LockVault() {
	mobileMu.Lock()
	defer mobileMu.Unlock()

	if mobileVault != nil {
		mobileVault.Lock()
	}
}

// IsVaultLocked returns true if vault is locked or uninitialized
func IsVaultLocked() bool {
	mobileMu.Lock()
	defer mobileMu.Unlock()

	if mobileVault == nil {
		return true
	}
	return mobileVault.IsLocked()
}

// ListAccountsJSON returns all accounts as a JSON string for Swift/iOS UI consumption
func ListAccountsJSON(query, tag string) (string, error) {
	mobileMu.Lock()
	defer mobileMu.Unlock()

	if mobileVault == nil || mobileVault.IsLocked() {
		return "", errors.New("vault locked")
	}

	accounts, err := mobileVault.ListAccounts(query, tag)
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(accounts)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// AddAccountFromURI parses an otpauth URI and adds it to the mobile vault
func AddAccountFromURI(uri string) (string, error) {
	mobileMu.Lock()
	defer mobileMu.Unlock()

	if mobileVault == nil || mobileVault.IsLocked() {
		return "", errors.New("vault locked")
	}

	parsed, err := totp.ParseURI(uri)
	if err != nil {
		return "", err
	}

	acc, err := mobileVault.AddAccount(vault.Account{
		Issuer:      parsed.Issuer,
		AccountName: parsed.Account,
		Secret:      parsed.Secret,
		Algorithm:   parsed.Algorithm,
		Digits:      parsed.Digits,
		Period:      parsed.Period,
		Type:        parsed.Type,
		Counter:     parsed.Counter,
	})
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(acc)
	return string(bytes), err
}

// CalculateLiveOTP calculates the current OTP code directly for a given secret
func CalculateLiveOTP(secret, algo string, digits, period int) (string, int, error) {
	opts := totp.Options{
		Secret:    secret,
		Algorithm: totp.Algorithm(algo),
		Digits:    digits,
		Period:    period,
		Type:      totp.TypeTOTP,
	}

	now := time.Now()
	code, err := totp.GenerateCode(opts, now)
	if err != nil {
		return "", 0, err
	}
	rem := totp.RemainingSeconds(period, now)
	return code, rem, nil
}
