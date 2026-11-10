package vault

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"twofa-vault/pkg/crypto"
	"twofa-vault/pkg/totp"
)

var (
	ErrVaultNotInitialized = errors.New("vault is not initialized")
	ErrVaultAlreadyInit    = errors.New("vault already initialized")
	ErrVaultLocked         = errors.New("vault is locked")
	ErrAccountNotFound     = errors.New("account not found")
)

// Vault manages the secure encrypted storage of 2FA accounts
type Vault struct {
	storagePath string
	mu          sync.RWMutex
	data        *VaultData
	passphrase  string
	locked      bool
	lockTimer   *time.Timer
}

// New creates a new Vault manager pointing to storagePath
func New(storagePath string) *Vault {
	v := &Vault{
		storagePath: storagePath,
		locked:      true,
	}
	return v
}

// StoragePath returns the path to the vault storage file
func (v *Vault) StoragePath() string {
	return v.storagePath
}

// IsInitialized returns true if the vault file exists
func (v *Vault) IsInitialized() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	_, err := os.Stat(v.storagePath)
	return err == nil
}

// IsLocked returns true if the vault is currently locked
func (v *Vault) IsLocked() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.locked || v.data == nil
}

// Init initializes a fresh vault with the master password
func (v *Vault) Init(masterPassword string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if _, err := os.Stat(v.storagePath); err == nil {
		return ErrVaultAlreadyInit
	}

	if len(strings.TrimSpace(masterPassword)) < 6 {
		return errors.New("master password must be at least 6 characters")
	}

	initialData := &VaultData{
		Version:         1,
		Accounts:        []Account{},
		AutoLockMinutes: 15,
		UpdatedAt:       time.Now(),
	}

	v.data = initialData
	v.passphrase = masterPassword
	v.locked = false

	if err := v.saveUnlocked(); err != nil {
		v.data = nil
		v.passphrase = ""
		v.locked = true
		return err
	}

	v.resetAutoLockTimer()
	return nil
}

// Unlock decrypts the vault using the master password
func (v *Vault) Unlock(masterPassword string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	raw, err := os.ReadFile(v.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrVaultNotInitialized
		}
		return fmt.Errorf("read vault file failed: %w", err)
	}

	decrypted, err := crypto.Decrypt(raw, masterPassword)
	if err != nil {
		return err
	}

	var data VaultData
	if err := json.Unmarshal(decrypted, &data); err != nil {
		return fmt.Errorf("corrupted vault json: %w", err)
	}

	v.data = &data
	v.passphrase = masterPassword
	v.locked = false
	v.resetAutoLockTimer()

	return nil
}

// Lock locks the vault and wipes sensitive memory
func (v *Vault) Lock() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.lockInternal()
}

func (v *Vault) lockInternal() {
	v.locked = true
	v.passphrase = ""
	v.data = nil
	if v.lockTimer != nil {
		v.lockTimer.Stop()
		v.lockTimer = nil
	}
}

// ChangePassword re-encrypts the vault with a new master password
func (v *Vault) ChangePassword(oldPass, newPass string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.locked || v.data == nil {
		return ErrVaultLocked
	}
	if v.passphrase != oldPass {
		return errors.New("current master password is incorrect")
	}
	if len(strings.TrimSpace(newPass)) < 6 {
		return errors.New("new master password must be at least 6 characters")
	}

	v.passphrase = newPass
	v.data.UpdatedAt = time.Now()
	return v.saveUnlocked()
}

// SetAutoLockMinutes updates the auto-lock duration
func (v *Vault) SetAutoLockMinutes(minutes int) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.locked || v.data == nil {
		return ErrVaultLocked
	}

	v.data.AutoLockMinutes = minutes
	v.resetAutoLockTimer()
	return v.saveUnlocked()
}

// AutoLockMinutes returns the current auto-lock duration
func (v *Vault) AutoLockMinutes() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.data == nil {
		return 15
	}
	return v.data.AutoLockMinutes
}

func (v *Vault) resetAutoLockTimer() {
	if v.lockTimer != nil {
		v.lockTimer.Stop()
	}
	if v.data == nil || v.data.AutoLockMinutes <= 0 {
		return
	}

	duration := time.Duration(v.data.AutoLockMinutes) * time.Minute
	v.lockTimer = time.AfterFunc(duration, func() {
		v.Lock()
	})
}

// ListAccounts returns all accounts decorated with their live OTP code, remaining time and progress
func (v *Vault) ListAccounts(query, tag string) ([]AccountDTO, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.locked || v.data == nil {
		return nil, ErrVaultLocked
	}
	v.resetAutoLockTimer()

	now := time.Now()
	query = strings.ToLower(strings.TrimSpace(query))
	tag = strings.ToLower(strings.TrimSpace(tag))

	results := make([]AccountDTO, 0, len(v.data.Accounts))
	for _, acc := range v.data.Accounts {
		if query != "" {
			matchIssuer := strings.Contains(strings.ToLower(acc.Issuer), query)
			matchAccount := strings.Contains(strings.ToLower(acc.AccountName), query)
			matchNotes := strings.Contains(strings.ToLower(acc.Notes), query)
			if !matchIssuer && !matchAccount && !matchNotes {
				continue
			}
		}

		if tag != "" {
			hasTag := false
			for _, t := range acc.Tags {
				if strings.ToLower(t) == tag {
					hasTag = true
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		// Calculate code
		opts := totp.Options{
			Secret:    acc.Secret,
			Algorithm: acc.Algorithm,
			Digits:    acc.Digits,
			Period:    acc.Period,
			Counter:   acc.Counter,
			Type:      acc.Type,
		}

		code, err := totp.GenerateCode(opts, now)
		if err != nil {
			code = "ERR"
		}

		rem := totp.RemainingSeconds(acc.Period, now)
		prog := totp.Progress(acc.Period, now)
		if acc.Type == totp.TypeHOTP {
			rem = 0
			prog = 0
		}

		results = append(results, AccountDTO{
			Account:   acc,
			Code:      code,
			Remaining: rem,
			Progress:  prog,
		})
	}

	// Sort pinned first, then by Order, then alphabetical by Issuer / Account
	sort.Slice(results, func(i, j int) bool {
		if results[i].Pinned != results[j].Pinned {
			return results[i].Pinned
		}
		if results[i].Order != results[j].Order {
			return results[i].Order < results[j].Order
		}
		if results[i].Issuer != results[j].Issuer {
			return strings.ToLower(results[i].Issuer) < strings.ToLower(results[j].Issuer)
		}
		return strings.ToLower(results[i].AccountName) < strings.ToLower(results[j].AccountName)
	})

	return results, nil
}

// GetAccount returns a single account by ID
func (v *Vault) GetAccount(id string) (*Account, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.locked || v.data == nil {
		return nil, ErrVaultLocked
	}
	v.resetAutoLockTimer()

	for _, acc := range v.data.Accounts {
		if acc.ID == id {
			cp := acc
			return &cp, nil
		}
	}
	return nil, ErrAccountNotFound
}

// AddAccount adds a new 2FA account to the vault
func (v *Vault) AddAccount(acc Account) (*Account, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.locked || v.data == nil {
		return nil, ErrVaultLocked
	}

	cleanSecret, err := totp.CleanSecret(acc.Secret)
	if err != nil {
		return nil, fmt.Errorf("invalid secret: %w", err)
	}
	acc.Secret = cleanSecret

	if acc.ID == "" {
		acc.ID = GenerateID()
	}
	if acc.Digits <= 0 {
		acc.Digits = 6
	}
	if acc.Period <= 0 {
		acc.Period = 30
	}
	if acc.Algorithm == "" {
		acc.Algorithm = totp.AlgorithmSHA1
	}
	if acc.Type == "" {
		acc.Type = totp.TypeTOTP
	}

	now := time.Now()
	acc.CreatedAt = now
	acc.UpdatedAt = now

	v.data.Accounts = append(v.data.Accounts, acc)
	v.data.UpdatedAt = now

	if err := v.saveUnlocked(); err != nil {
		return nil, err
	}

	v.resetAutoLockTimer()
	return &acc, nil
}

// UpdateAccount updates an existing 2FA account
func (v *Vault) UpdateAccount(acc Account) (*Account, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.locked || v.data == nil {
		return nil, ErrVaultLocked
	}

	cleanSecret, err := totp.CleanSecret(acc.Secret)
	if err != nil {
		return nil, fmt.Errorf("invalid secret: %w", err)
	}
	acc.Secret = cleanSecret

	found := false
	for i, existing := range v.data.Accounts {
		if existing.ID == acc.ID {
			acc.CreatedAt = existing.CreatedAt
			acc.UpdatedAt = time.Now()
			v.data.Accounts[i] = acc
			found = true
			break
		}
	}

	if !found {
		return nil, ErrAccountNotFound
	}

	v.data.UpdatedAt = time.Now()
	if err := v.saveUnlocked(); err != nil {
		return nil, err
	}

	v.resetAutoLockTimer()
	return &acc, nil
}

// DeleteAccount removes an account by ID
func (v *Vault) DeleteAccount(id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.locked || v.data == nil {
		return ErrVaultLocked
	}

	idx := -1
	for i, existing := range v.data.Accounts {
		if existing.ID == id {
			idx = i
			break
		}
	}

	if idx == -1 {
		return ErrAccountNotFound
	}

	v.data.Accounts = append(v.data.Accounts[:idx], v.data.Accounts[idx+1:]...)
	v.data.UpdatedAt = time.Now()

	if err := v.saveUnlocked(); err != nil {
		return err
	}

	v.resetAutoLockTimer()
	return nil
}

// IncrementHOTP increments the counter for an HOTP account and returns the new code
func (v *Vault) IncrementHOTP(id string) (string, uint64, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.locked || v.data == nil {
		return "", 0, ErrVaultLocked
	}

	for i, existing := range v.data.Accounts {
		if existing.ID == id {
			if existing.Type != totp.TypeHOTP {
				return "", 0, errors.New("account is not an HOTP type")
			}

			v.data.Accounts[i].Counter++
			v.data.Accounts[i].UpdatedAt = time.Now()

			opts := totp.Options{
				Secret:    existing.Secret,
				Algorithm: existing.Algorithm,
				Digits:    existing.Digits,
				Counter:   v.data.Accounts[i].Counter,
				Type:      totp.TypeHOTP,
			}
			code, err := totp.GenerateCode(opts, time.Time{})
			if err != nil {
				return "", 0, err
			}

			_ = v.saveUnlocked()
			v.resetAutoLockTimer()
			return code, v.data.Accounts[i].Counter, nil
		}
	}

	return "", 0, ErrAccountNotFound
}

// TogglePin toggles the pinned status of an account
func (v *Vault) TogglePin(id string) (*Account, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.locked || v.data == nil {
		return nil, ErrVaultLocked
	}

	for i := range v.data.Accounts {
		if v.data.Accounts[i].ID == id {
			v.data.Accounts[i].Pinned = !v.data.Accounts[i].Pinned
			v.data.Accounts[i].UpdatedAt = time.Now()
			if err := v.saveUnlocked(); err != nil {
				return nil, err
			}
			v.resetAutoLockTimer()
			cp := v.data.Accounts[i]
			return &cp, nil
		}
	}

	return nil, ErrAccountNotFound
}

// ReorderAccounts updates the ordering of accounts given an array of account IDs
func (v *Vault) ReorderAccounts(ids []string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.locked || v.data == nil {
		return ErrVaultLocked
	}

	idMap := make(map[string]int)
	for i, id := range ids {
		idMap[id] = i
	}

	for i := range v.data.Accounts {
		if pos, ok := idMap[v.data.Accounts[i].ID]; ok {
			v.data.Accounts[i].Order = pos
		}
	}

	v.data.UpdatedAt = time.Now()
	if err := v.saveUnlocked(); err != nil {
		return err
	}
	v.resetAutoLockTimer()
	return nil
}

// ToggleRecoveryCode toggles the used status of a specific recovery code
func (v *Vault) ToggleRecoveryCode(accountID string, codeIndex int) (*Account, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.locked || v.data == nil {
		return nil, ErrVaultLocked
	}

	for i := range v.data.Accounts {
		if v.data.Accounts[i].ID == accountID {
			if codeIndex < 0 || codeIndex >= len(v.data.Accounts[i].RecoveryCodes) {
				return nil, errors.New("invalid recovery code index")
			}
			v.data.Accounts[i].RecoveryCodes[codeIndex].Used = !v.data.Accounts[i].RecoveryCodes[codeIndex].Used
			v.data.Accounts[i].UpdatedAt = time.Now()
			if err := v.saveUnlocked(); err != nil {
				return nil, err
			}
			v.resetAutoLockTimer()
			cp := v.data.Accounts[i]
			return &cp, nil
		}
	}

	return nil, ErrAccountNotFound
}

// GetRawEncryptedData reads the raw encrypted vault file from disk (for P2P LAN sync)
func (v *Vault) GetRawEncryptedData() ([]byte, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return os.ReadFile(v.storagePath)
}

// ImportRawEncryptedData overwrites the encrypted vault file on disk (for P2P LAN sync)
func (v *Vault) ImportRawEncryptedData(data []byte) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	tmp := v.storagePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, v.storagePath); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	// Re-lock vault so master password must be entered
	v.locked = true
	v.data = nil
	v.passphrase = ""
	return nil
}

// Tags returns all unique tags currently in use across accounts
func (v *Vault) Tags() ([]string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.locked || v.data == nil {
		return nil, ErrVaultLocked
	}

	tagMap := make(map[string]struct{})
	for _, acc := range v.data.Accounts {
		for _, t := range acc.Tags {
			cleaned := strings.TrimSpace(t)
			if cleaned != "" {
				tagMap[cleaned] = struct{}{}
			}
		}
	}

	tags := make([]string, 0, len(tagMap))
	for t := range tagMap {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags, nil
}

// ExportPlain returns unencrypted JSON backup of the vault
func (v *Vault) ExportPlain() ([]byte, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.locked || v.data == nil {
		return nil, ErrVaultLocked
	}
	return json.MarshalIndent(v.data, "", "  ")
}

// ExportEncrypted exports the vault re-encrypted with a specific backup passphrase
func (v *Vault) ExportEncrypted(passphrase string) ([]byte, error) {
	plain, err := v.ExportPlain()
	if err != nil {
		return nil, err
	}
	return crypto.Encrypt(plain, passphrase)
}

// ImportAccounts merges a slice of accounts into the vault
func (v *Vault) ImportAccounts(accounts []Account) (int, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.locked || v.data == nil {
		return 0, ErrVaultLocked
	}

	count := 0
	for _, acc := range accounts {
		cleanSec, err := totp.CleanSecret(acc.Secret)
		if err != nil {
			continue
		}
		acc.Secret = cleanSec
		if acc.ID == "" {
			acc.ID = GenerateID()
		}
		if acc.Digits <= 0 {
			acc.Digits = 6
		}
		if acc.Period <= 0 {
			acc.Period = 30
		}
		if acc.Algorithm == "" {
			acc.Algorithm = totp.AlgorithmSHA1
		}
		if acc.Type == "" {
			acc.Type = totp.TypeTOTP
		}

		now := time.Now()
		acc.CreatedAt = now
		acc.UpdatedAt = now

		v.data.Accounts = append(v.data.Accounts, acc)
		count++
	}

	if count > 0 {
		v.data.UpdatedAt = time.Now()
		if err := v.saveUnlocked(); err != nil {
			return 0, err
		}
	}

	v.resetAutoLockTimer()
	return count, nil
}

// saveUnlocked marshals and encrypts data, then saves it atomically
func (v *Vault) saveUnlocked() error {
	if v.data == nil || v.passphrase == "" {
		return errors.New("cannot save locked or empty vault")
	}

	plain, err := json.Marshal(v.data)
	if err != nil {
		return fmt.Errorf("marshal vault failed: %w", err)
	}

	encrypted, err := crypto.Encrypt(plain, v.passphrase)
	if err != nil {
		return fmt.Errorf("encrypt vault failed: %w", err)
	}

	dir := filepath.Dir(v.storagePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create vault directory failed: %w", err)
	}

	tmpFile := v.storagePath + ".tmp"
	if err := os.WriteFile(tmpFile, encrypted, 0600); err != nil {
		return fmt.Errorf("write temp vault failed: %w", err)
	}

	if err := os.Rename(tmpFile, v.storagePath); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("atomic rename failed: %w", err)
	}

	return nil
}
