package totp

import (
	"testing"
	"time"
)

func TestHOTP_RFC4226Vectors(t *testing.T) {
	// RFC 4226 Appendix D test vectors
	// Secret in ASCII: "12345678901234567890" -> Base32: "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

	expected := []string{
		"755224",
		"287082",
		"359152",
		"969429",
		"338314",
		"254676",
		"287922",
		"162583",
		"399871",
		"520489",
	}

	for i, exp := range expected {
		opts := Options{
			Secret:    secret,
			Algorithm: AlgorithmSHA1,
			Digits:    6,
			Counter:   uint64(i),
			Type:      TypeHOTP,
		}

		code, err := GenerateCode(opts, time.Time{})
		if err != nil {
			t.Fatalf("step %d failed: %v", i, err)
		}
		if code != exp {
			t.Errorf("step %d: expected %s, got %s", i, exp, code)
		}
	}
}

func TestTOTP_RFC6238Vectors(t *testing.T) {
	// RFC 6238 Appendix B test vectors
	secretSHA1 := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

	testCases := []struct {
		timestamp int64
		expected  string
	}{
		{59, "94287082"},
		{1111111109, "07081804"},
		{1111111111, "14050471"},
		{1234567890, "89005924"},
		{2000000000, "69279037"},
	}

	for _, tc := range testCases {
		opts := Options{
			Secret:    secretSHA1,
			Algorithm: AlgorithmSHA1,
			Digits:    8,
			Period:    30,
			Type:      TypeTOTP,
		}

		code, err := GenerateCode(opts, time.Unix(tc.timestamp, 0))
		if err != nil {
			t.Fatalf("time %d failed: %v", tc.timestamp, err)
		}
		if code != tc.expected {
			t.Errorf("time %d: expected %s, got %s", tc.timestamp, tc.expected, code)
		}
	}
}

func TestCleanSecret(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"JBSWY3DPEHPK3PXP", "JBSWY3DPEHPK3PXP"},
		{"jbsw y3dp ehpk 3pxp", "JBSWY3DPEHPK3PXP"},
		{"JBSW-Y3DP-EHPK-3PXP", "JBSWY3DPEHPK3PXP"},
		{"JBSWY3DP", "JBSWY3DP"},
		{"JBSWY", "JBSWY==="},
	}

	for _, c := range cases {
		out, err := CleanSecret(c.input)
		if err != nil {
			t.Errorf("CleanSecret(%q) error: %v", c.input, err)
		}
		if out != c.expected {
			t.Errorf("CleanSecret(%q) = %q; want %q", c.input, out, c.expected)
		}
	}
}

func TestURIParseAndBuild(t *testing.T) {
	uri := "otpauth://totp/GitHub:alice@example.com?algorithm=SHA256&digits=8&issuer=GitHub&period=60&secret=JBSWY3DPEHPK3PXP"
	parsed, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}

	if parsed.Issuer != "GitHub" {
		t.Errorf("expected Issuer GitHub, got %s", parsed.Issuer)
	}
	if parsed.Account != "alice@example.com" {
		t.Errorf("expected Account alice@example.com, got %s", parsed.Account)
	}
	if parsed.Algorithm != AlgorithmSHA256 {
		t.Errorf("expected Algorithm SHA256, got %s", parsed.Algorithm)
	}
	if parsed.Digits != 8 {
		t.Errorf("expected Digits 8, got %d", parsed.Digits)
	}
	if parsed.Period != 60 {
		t.Errorf("expected Period 60, got %d", parsed.Period)
	}

	rebuilt := BuildURI(*parsed)
	reparsed, err := ParseURI(rebuilt)
	if err != nil {
		t.Fatalf("re-parse failed: %v", err)
	}
	if reparsed.Account != parsed.Account || reparsed.Secret != parsed.Secret {
		t.Errorf("reparsed mismatch: %+v vs %+v", reparsed, parsed)
	}
}

func TestSteamOTP(t *testing.T) {
	opts := Options{
		Secret: "JBSWY3DPEHPK3PXP",
		Type:   TypeSteam,
	}
	code, err := GenerateCode(opts, time.Unix(1600000000, 0))
	if err != nil {
		t.Fatalf("Steam OTP failed: %v", err)
	}
	if len(code) != 5 {
		t.Errorf("expected 5-character Steam code, got %s", code)
	}
}
