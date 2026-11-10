package totp

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParsedOTP represents the data extracted from an otpauth:// URI
type ParsedOTP struct {
	Type      OTPType
	Issuer    string
	Account   string
	Secret    string
	Algorithm Algorithm
	Digits    int
	Period    int
	Counter   uint64
}

// ParseURI parses standard otpauth://totp or otpauth://hotp URIs
func ParseURI(uriStr string) (*ParsedOTP, error) {
	u, err := url.Parse(strings.TrimSpace(uriStr))
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "otpauth" {
		return nil, errors.New("scheme must be 'otpauth'")
	}

	var otpType OTPType
	switch strings.ToLower(u.Host) {
	case "totp":
		otpType = TypeTOTP
	case "hotp":
		otpType = TypeHOTP
	default:
		return nil, fmt.Errorf("unsupported OTP type: %s", u.Host)
	}

	q := u.Query()
	secret := q.Get("secret")
	if secret == "" {
		return nil, errors.New("missing secret in URI")
	}

	cleanedSecret, err := CleanSecret(secret)
	if err != nil {
		return nil, fmt.Errorf("invalid secret: %w", err)
	}

	// Path parsing: /Issuer:Account or /Account
	label := strings.TrimPrefix(u.Path, "/")
	label, _ = url.PathUnescape(label)

	var issuer, account string
	if idx := strings.Index(label, ":"); idx != -1 {
		issuer = strings.TrimSpace(label[:idx])
		account = strings.TrimSpace(label[idx+1:])
	} else {
		account = strings.TrimSpace(label)
	}

	// Query issuer overrides or provides fallback
	if qIssuer := q.Get("issuer"); qIssuer != "" {
		issuer = qIssuer
	}

	// Algorithm
	algo := AlgorithmSHA1
	if a := q.Get("algorithm"); a != "" {
		switch strings.ToUpper(a) {
		case "SHA256":
			algo = AlgorithmSHA256
		case "SHA512":
			algo = AlgorithmSHA512
		default:
			algo = AlgorithmSHA1
		}
	}

	// Digits
	digits := 6
	if d := q.Get("digits"); d != "" {
		if val, err := strconv.Atoi(d); err == nil && val > 0 {
			digits = val
		}
	}

	// Period (for TOTP)
	period := 30
	if p := q.Get("period"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			period = val
		}
	}

	// Counter (for HOTP)
	var counter uint64 = 0
	if c := q.Get("counter"); c != "" {
		if val, err := strconv.ParseUint(c, 10, 64); err == nil {
			counter = val
		}
	}

	return &ParsedOTP{
		Type:      otpType,
		Issuer:    issuer,
		Account:   account,
		Secret:    cleanedSecret,
		Algorithm: algo,
		Digits:    digits,
		Period:    period,
		Counter:   counter,
	}, nil
}

// BuildURI constructs a standard otpauth:// URI
func BuildURI(p ParsedOTP) string {
	typeStr := string(p.Type)
	if typeStr == "" || typeStr == string(TypeSteam) {
		typeStr = string(TypeTOTP)
	}

	label := p.Account
	if p.Issuer != "" {
		label = fmt.Sprintf("%s:%s", p.Issuer, p.Account)
	}

	u := url.URL{
		Scheme: "otpauth",
		Host:   typeStr,
		Path:   "/" + label,
	}

	q := u.Query()
	q.Set("secret", p.Secret)
	if p.Issuer != "" {
		q.Set("issuer", p.Issuer)
	}
	if p.Algorithm != "" && p.Algorithm != AlgorithmSHA1 {
		q.Set("algorithm", string(p.Algorithm))
	}
	if p.Digits > 0 && p.Digits != 6 {
		q.Set("digits", strconv.Itoa(p.Digits))
	}
	if p.Type == TypeTOTP && p.Period > 0 && p.Period != 30 {
		q.Set("period", strconv.Itoa(p.Period))
	}
	if p.Type == TypeHOTP {
		q.Set("counter", strconv.FormatUint(p.Counter, 10))
	}

	u.RawQuery = q.Encode()
	return u.String()
}
