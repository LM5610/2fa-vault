package crypto

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	passphrase := "CorrectHorseBatteryStaple!2026"
	plaintext := []byte("secret_2fa_vault_data_contents_1234567890")

	encrypted, err := Encrypt(plaintext, passphrase)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, passphrase)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("plaintext mismatch: expected %s, got %s", plaintext, decrypted)
	}
}

func TestDecrypt_WrongPassword(t *testing.T) {
	passphrase := "SecretMasterPassword"
	plaintext := []byte("top_secret_data")

	encrypted, err := Encrypt(plaintext, passphrase)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(encrypted, "WrongPassword123")
	if err == nil {
		t.Fatal("expected error on wrong password, got nil")
	}

	if err != ErrDecryptionFailed {
		t.Errorf("expected ErrDecryptionFailed, got %v", err)
	}
}

func TestDecrypt_TamperedData(t *testing.T) {
	passphrase := "SecretMasterPassword"
	plaintext := []byte("top_secret_data")

	encrypted, err := Encrypt(plaintext, passphrase)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	var env EncryptedEnvelope
	if err := json.Unmarshal(encrypted, &env); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Corrupt ciphertext
	if len(env.Ciphertext) > 4 {
		env.Ciphertext = "AAAA" + env.Ciphertext[4:]
	} else {
		env.Ciphertext = "AAAA"
	}

	tampered, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	_, err = Decrypt(tampered, passphrase)
	if err == nil {
		t.Fatal("expected decryption error on tampered data, got nil")
	}
}
