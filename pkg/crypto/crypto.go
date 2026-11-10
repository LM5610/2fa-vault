package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

var (
	ErrDecryptionFailed = errors.New("decryption failed: incorrect password or corrupted data")
	ErrInvalidPayload   = errors.New("invalid encrypted payload format")
)

// KDFParams specifies parameters used for Argon2id
type KDFParams struct {
	Time    uint32 `json:"time"`
	Memory  uint32 `json:"memory"`
	Threads uint8  `json:"threads"`
	KeyLen  uint32 `json:"key_len"`
}

// DefaultKDFParams returns balanced Argon2id parameters (safe & fast for client devices)
func DefaultKDFParams() KDFParams {
	return KDFParams{
		Time:    1,
		Memory:  64 * 1024, // 64 MB
		Threads: 4,
		KeyLen:  32, // AES-256
	}
}

// EncryptedEnvelope is the serialized structure holding encrypted vault data
type EncryptedEnvelope struct {
	Version    int       `json:"version"`
	KDF        string    `json:"kdf"`
	Params     KDFParams `json:"params"`
	Salt       string    `json:"salt"`
	Nonce      string    `json:"nonce"`
	Ciphertext string    `json:"ciphertext"`
}

// DeriveKey generates a 32-byte key from passphrase and salt using Argon2id
func DeriveKey(passphrase string, salt []byte, params KDFParams) []byte {
	return argon2.IDKey([]byte(passphrase), salt, params.Time, params.Memory, params.Threads, params.KeyLen)
}

// Encrypt encrypts plaintext with the given passphrase using Argon2id and AES-256-GCM
func Encrypt(plaintext []byte, passphrase string) ([]byte, error) {
	if len(passphrase) == 0 {
		return nil, errors.New("empty passphrase")
	}

	params := DefaultKDFParams()

	// 16-byte random salt
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt failed: %w", err)
	}

	// 12-byte standard GCM nonce
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce failed: %w", err)
	}

	key := DeriveKey(passphrase, salt, params)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES block failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM failed: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	envelope := EncryptedEnvelope{
		Version:    1,
		KDF:        "argon2id",
		Params:     params,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}

	return json.MarshalIndent(envelope, "", "  ")
}

// Decrypt unpacks and decrypts the encrypted payload with the given passphrase
func Decrypt(envelopeJSON []byte, passphrase string) ([]byte, error) {
	var env EncryptedEnvelope
	if err := json.Unmarshal(envelopeJSON, &env); err != nil {
		return nil, ErrInvalidPayload
	}

	if env.KDF != "argon2id" && env.KDF != "" {
		return nil, fmt.Errorf("unsupported KDF algorithm: %s", env.KDF)
	}

	salt, err := base64.StdEncoding.DecodeString(env.Salt)
	if err != nil || len(salt) == 0 {
		return nil, ErrInvalidPayload
	}

	nonce, err := base64.StdEncoding.DecodeString(env.Nonce)
	if err != nil || len(nonce) == 0 {
		return nil, ErrInvalidPayload
	}

	ciphertext, err := base64.StdEncoding.DecodeString(env.Ciphertext)
	if err != nil || len(ciphertext) == 0 {
		return nil, ErrInvalidPayload
	}

	params := env.Params
	if params.KeyLen == 0 {
		params.KeyLen = 32
	}
	if params.Memory == 0 {
		params.Memory = 64 * 1024
	}
	if params.Time == 0 {
		params.Time = 1
	}
	if params.Threads == 0 {
		params.Threads = 4
	}

	key := DeriveKey(passphrase, salt, params)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM failed: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}
