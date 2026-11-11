package vault

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"twofa-vault/pkg/totp"
)

// RecoveryCode represents a single emergency backup code
type RecoveryCode struct {
	Code string `json:"code"`
	Used bool   `json:"used"`
}

// Account represents a single 2FA entry
type Account struct {
	ID            string         `json:"id"`
	Issuer        string         `json:"issuer"`
	AccountName   string         `json:"account_name"`
	Secret        string         `json:"secret"`
	Algorithm     totp.Algorithm `json:"algorithm"`
	Digits        int            `json:"digits"`
	Period        int            `json:"period"`
	Type          totp.OTPType   `json:"type"`
	Counter       uint64         `json:"counter"`
	Tags          []string       `json:"tags,omitempty"`
	Notes         string         `json:"notes,omitempty"`
	Pinned        bool           `json:"pinned,omitempty"`
	Order         int            `json:"order,omitempty"`
	RecoveryCodes []RecoveryCode `json:"recovery_codes,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// AccountDTO is the view returned to UI/clients with live OTP code
type AccountDTO struct {
	Account
	Code      string  `json:"code"`
	Remaining int     `json:"remaining"`
	Progress  float64 `json:"progress"`
	NextCode  string  `json:"next_code,omitempty"`
}

// VaultData represents the unencrypted contents of the vault
type VaultData struct {
	Version         int       `json:"version"`
	Accounts        []Account `json:"accounts"`
	AutoLockMinutes int       `json:"auto_lock_minutes"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// GenerateID produces a random 16-hex character ID
func GenerateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
