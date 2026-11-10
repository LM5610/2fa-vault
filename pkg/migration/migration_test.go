package migration

import (
	"testing"

	"twofa-vault/pkg/vault"
)

func TestParseImportText_URIs(t *testing.T) {
	raw := `
# My backup 2FA list
otpauth://totp/GitHub:alice@github.com?secret=JBSWY3DPEHPK3PXP&issuer=GitHub
otpauth://totp/Google:alice@gmail.com?secret=JBSWY3DPEHPK3PXP&issuer=Google&digits=6
otpauth://hotp/Bank:alice@bank.com?secret=JBSWY3DPEHPK3PXP&issuer=Bank&counter=5
`
	accounts, err := ParseImportText([]byte(raw))
	if err != nil {
		t.Fatalf("ParseImportText failed: %v", err)
	}

	if len(accounts) != 3 {
		t.Fatalf("expected 3 accounts, got %d", len(accounts))
	}

	if accounts[0].Issuer != "GitHub" || accounts[0].AccountName != "alice@github.com" {
		t.Errorf("account 0 mismatch: %+v", accounts[0])
	}
	if accounts[2].Counter != 5 {
		t.Errorf("account 2 counter mismatch: %d", accounts[2].Counter)
	}
}

func TestExportToURIs(t *testing.T) {
	accs := []vault.Account{
		{
			Issuer:      "GitHub",
			AccountName: "dev@github.com",
			Secret:      "JBSWY3DPEHPK3PXP",
		},
	}
	uris := ExportToURIs(accs)
	if uris == "" {
		t.Fatal("expected non-empty URI string")
	}

	reimported, err := ParseImportText([]byte(uris))
	if err != nil || len(reimported) != 1 {
		t.Fatalf("reimport failed: %v, count=%d", err, len(reimported))
	}
	if reimported[0].Issuer != "GitHub" {
		t.Errorf("expected GitHub, got %s", reimported[0].Issuer)
	}
}
