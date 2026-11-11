package migration

import (
	"encoding/base64"
	"net/url"
	"strings"
	"testing"

	"twofa-vault/pkg/totp"
	"twofa-vault/pkg/vault"
)

func TestParseGoogleMigrationURI_StandardVector(t *testing.T) {
	// Standard open-source test vector for Google Authenticator migration
	uri := "otpauth-migration://offline?data=CjEKCkhlbGxvId6tvu8SGEV4YW1wbGU6YWxpY2VAZ29vZ2xlLmNvbRoHRXhhbXBsZTAC"

	accounts, err := ParseGoogleMigrationURI(uri)
	if err != nil {
		t.Fatalf("ParseGoogleMigrationURI failed: %v", err)
	}

	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}

	acc := accounts[0]
	if acc.Issuer != "Example" {
		t.Errorf("expected Issuer 'Example', got '%s'", acc.Issuer)
	}
	if acc.AccountName != "alice@google.com" {
		t.Errorf("expected AccountName 'alice@google.com', got '%s'", acc.AccountName)
	}
	if acc.Secret != "JBSWY3DPEHPK3PXP" {
		t.Errorf("expected Secret 'JBSWY3DPEHPK3PXP', got '%s'", acc.Secret)
	}
	if acc.Algorithm != totp.AlgorithmSHA1 {
		t.Errorf("expected Algorithm SHA1, got %v", acc.Algorithm)
	}
	if acc.Digits != 6 {
		t.Errorf("expected Digits 6, got %d", acc.Digits)
	}
	if acc.Type != totp.TypeTOTP {
		t.Errorf("expected Type TOTP, got %v", acc.Type)
	}
}

func TestParseGoogleMigrationURI_RoundTrip(t *testing.T) {
	inputAccounts := []vault.Account{
		{
			Issuer:      "GitHub",
			AccountName: "alice@github.com",
			Secret:      "JBSWY3DPEHPK3PXP",
			Algorithm:   totp.AlgorithmSHA1,
			Digits:      6,
			Period:      30,
			Type:        totp.TypeTOTP,
		},
		{
			Issuer:      "AWS",
			AccountName: "admin@corp",
			Secret:      "MZXW6YTBOI======",
			Algorithm:   totp.AlgorithmSHA256,
			Digits:      8,
			Period:      30,
			Type:        totp.TypeTOTP,
		},
		{
			Issuer:      "Bank",
			AccountName: "user@bank.com",
			Secret:      "JBSWY3DPEHPK3PXP",
			Algorithm:   totp.AlgorithmSHA1,
			Digits:      6,
			Period:      30,
			Type:        totp.TypeHOTP,
			Counter:     42,
		},
	}

	uri, err := BuildGoogleMigrationURI(inputAccounts)
	if err != nil {
		t.Fatalf("BuildGoogleMigrationURI failed: %v", err)
	}

	if !strings.HasPrefix(uri, "otpauth-migration://offline?data=") {
		t.Fatalf("unexpected URI format: %s", uri)
	}

	parsed, err := ParseGoogleMigrationURI(uri)
	if err != nil {
		t.Fatalf("ParseGoogleMigrationURI failed: %v", err)
	}

	if len(parsed) != len(inputAccounts) {
		t.Fatalf("expected %d accounts, got %d", len(inputAccounts), len(parsed))
	}

	// Verify GitHub account
	if parsed[0].Issuer != "GitHub" || parsed[0].AccountName != "alice@github.com" {
		t.Errorf("account 0 mismatch: %+v", parsed[0])
	}
	if parsed[0].Secret != "JBSWY3DPEHPK3PXP" {
		t.Errorf("account 0 secret mismatch: %s", parsed[0].Secret)
	}
	if parsed[0].Algorithm != totp.AlgorithmSHA1 || parsed[0].Digits != 6 || parsed[0].Type != totp.TypeTOTP {
		t.Errorf("account 0 parameters mismatch: %+v", parsed[0])
	}

	// Verify AWS account
	if parsed[1].Issuer != "AWS" || parsed[1].AccountName != "admin@corp" {
		t.Errorf("account 1 mismatch: %+v", parsed[1])
	}
	if parsed[1].Algorithm != totp.AlgorithmSHA256 || parsed[1].Digits != 8 || parsed[1].Type != totp.TypeTOTP {
		t.Errorf("account 1 parameters mismatch: %+v", parsed[1])
	}

	// Verify Bank HOTP account
	if parsed[2].Issuer != "Bank" || parsed[2].AccountName != "user@bank.com" {
		t.Errorf("account 2 mismatch: %+v", parsed[2])
	}
	if parsed[2].Type != totp.TypeHOTP || parsed[2].Counter != 42 {
		t.Errorf("account 2 HOTP mismatch: type=%v counter=%d", parsed[2].Type, parsed[2].Counter)
	}
}

func TestParseGoogleMigrationURI_Base64Variants(t *testing.T) {
	// Standard payload
	accounts := []vault.Account{
		{
			Issuer:      "Test",
			AccountName: "user@test.com",
			Secret:      "JBSWY3DPEHPK3PXP",
		},
	}
	uri, err := BuildGoogleMigrationURI(accounts)
	if err != nil {
		t.Fatalf("build URI failed: %v", err)
	}

	u, _ := url.Parse(uri)
	data := u.Query().Get("data")

	// 1. URL-safe unpadded base64
	rawBytes, _ := base64.StdEncoding.DecodeString(data)
	urlSafeUnpadded := base64.RawURLEncoding.EncodeToString(rawBytes)
	testURI1 := "otpauth-migration://offline?data=" + urlSafeUnpadded
	res1, err := ParseGoogleMigrationURI(testURI1)
	if err != nil || len(res1) != 1 {
		t.Fatalf("urlSafeUnpadded failed: %v, count=%d", err, len(res1))
	}

	// 2. Query string with spaces instead of '+' (simulates net/url unescaping)
	withSpaces := strings.ReplaceAll(data, "+", " ")
	testURI2 := "otpauth-migration://offline?data=" + withSpaces
	res2, err := ParseGoogleMigrationURI(testURI2)
	if err != nil || len(res2) != 1 {
		t.Fatalf("withSpaces failed: %v, count=%d", err, len(res2))
	}

	// 3. Raw standard unpadded base64
	rawStdUnpadded := base64.RawStdEncoding.EncodeToString(rawBytes)
	testURI3 := "otpauth-migration://offline?data=" + rawStdUnpadded
	res3, err := ParseGoogleMigrationURI(testURI3)
	if err != nil || len(res3) != 1 {
		t.Fatalf("rawStdUnpadded failed: %v, count=%d", err, len(res3))
	}
}

func TestParseGoogleMigrationURI_Errors(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{"empty URI", ""},
		{"wrong scheme", "otpauth://totp/Test?secret=JBSWY3DPEHPK3PXP"},
		{"missing data query", "otpauth-migration://offline"},
		{"invalid base64", "otpauth-migration://offline?data=***INVALID***"},
		{"invalid protobuf varint", "otpauth-migration://offline?data=gICAgICAgICAgAA="},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseGoogleMigrationURI(tc.uri)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
		})
	}
}

func TestParseGoogleMigration_UnknownFields(t *testing.T) {
	// Construct a payload containing unknown wire types (fixed32, fixed64, unknown varint)
	// to ensure protobuf forward-compatibility
	payload := &MigrationPayload{
		Version: 1,
		OtpParameters: []*OtpParameters{
			{
				Secret: []byte("Hello!\xde\xad\xbe\xef"),
				Name:   "Extra:user",
				Issuer: "Extra",
			},
		},
	}
	bytes := payload.MarshalProtobuf()

	// Append unknown field: field 99, wire 0 (varint) = (99 << 3) | 0 = 792
	bytes = append(bytes, encodeVarint(uint64((99<<3)|0))...)
	bytes = append(bytes, encodeVarint(12345)...)

	// Append unknown field: field 100, wire 2 (length-delimited) = (100 << 3) | 2 = 802
	bytes = append(bytes, encodeVarint(uint64((100<<3)|2))...)
	bytes = append(bytes, encodeVarint(uint64(len("unknown string")))...)
	bytes = append(bytes, []byte("unknown string")...)

	accs, err := ParseGoogleMigrationPayload(bytes)
	if err != nil {
		t.Fatalf("expected success with unknown fields, got: %v", err)
	}
	if len(accs) != 1 || accs[0].Issuer != "Extra" {
		t.Fatalf("mismatch after skipping unknown fields: %+v", accs)
	}
}
