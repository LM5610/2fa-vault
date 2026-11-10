package totp

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"math"
	"strings"
	"time"
)

// Algorithm specifies the HMAC hashing algorithm
type Algorithm string

const (
	AlgorithmSHA1   Algorithm = "SHA1"
	AlgorithmSHA256 Algorithm = "SHA256"
	AlgorithmSHA512 Algorithm = "SHA512"
)

// OTPType specifies the one-time password type
type OTPType string

const (
	TypeTOTP  OTPType = "totp"
	TypeHOTP  OTPType = "hotp"
	TypeSteam OTPType = "steam"
)

const steamAlphabet = "23456789BCDFGHJKMNPQRTVWXY"

// Options holds parameters for OTP generation
type Options struct {
	Secret    string
	Algorithm Algorithm
	Digits    int
	Period    int
	Counter   uint64
	Type      OTPType
}

// DefaultOptions returns default options for TOTP
func DefaultOptions(secret string) Options {
	return Options{
		Secret:    secret,
		Algorithm: AlgorithmSHA1,
		Digits:    6,
		Period:    30,
		Counter:   0,
		Type:      TypeTOTP,
	}
}

// CleanSecret removes spaces, dashes and normalizes base32 secret with proper padding
func CleanSecret(secret string) (string, error) {
	s := strings.ToUpper(strings.TrimSpace(secret))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")

	if len(s) == 0 {
		return "", errors.New("empty secret key")
	}

	// Pad with '=' to make length multiple of 8
	if rem := len(s) % 8; rem != 0 {
		s += strings.Repeat("=", 8-rem)
	}

	// Validate base32
	_, err := base32.StdEncoding.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("invalid base32 secret: %w", err)
	}

	return s, nil
}

// GenerateCode calculates the OTP code at the given time or counter
func GenerateCode(opts Options, t time.Time) (string, error) {
	if opts.Type == "" {
		opts.Type = TypeTOTP
	}
	if opts.Digits <= 0 {
		opts.Digits = 6
	}
	if opts.Period <= 0 {
		opts.Period = 30
	}
	if opts.Algorithm == "" {
		opts.Algorithm = AlgorithmSHA1
	}

	cleanedSecret, err := CleanSecret(opts.Secret)
	if err != nil {
		return "", err
	}

	secretBytes, err := base32.StdEncoding.DecodeString(cleanedSecret)
	if err != nil {
		return "", fmt.Errorf("decode secret failed: %w", err)
	}

	var counter uint64
	if opts.Type == TypeHOTP {
		counter = opts.Counter
	} else {
		// TOTP or Steam
		counter = uint64(t.Unix() / int64(opts.Period))
	}

	if opts.Type == TypeSteam {
		return generateSteamCode(secretBytes, counter)
	}

	return generateHOTP(secretBytes, counter, opts.Digits, opts.Algorithm)
}

func generateHOTP(secret []byte, counter uint64, digits int, algo Algorithm) (string, error) {
	var h func() hash.Hash
	switch strings.ToUpper(string(algo)) {
	case string(AlgorithmSHA256):
		h = sha256.New
	case string(AlgorithmSHA512):
		h = sha512.New
	default:
		h = sha1.New
	}

	mac := hmac.New(h, secret)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)
	mac.Write(buf)
	sum := mac.Sum(nil)

	// Dynamic truncation (RFC 4226 section 5.4)
	offset := sum[len(sum)-1] & 0x0f
	codeInt := (int(sum[offset])&0x7f)<<24 |
		(int(sum[offset+1])&0xff)<<16 |
		(int(sum[offset+2])&0xff)<<8 |
		(int(sum[offset+3]) & 0xff)

	mod := int(math.Pow10(digits))
	code := codeInt % mod

	format := fmt.Sprintf("%%0%dd", digits)
	return fmt.Sprintf(format, code), nil
}

func generateSteamCode(secret []byte, counter uint64) (string, error) {
	mac := hmac.New(sha1.New, secret)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)
	mac.Write(buf)
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	codeInt := (uint32(sum[offset])&0x7f)<<24 |
		(uint32(sum[offset+1])&0xff)<<16 |
		(uint32(sum[offset+2])&0xff)<<8 |
		(uint32(sum[offset+3]) & 0xff)

	result := make([]byte, 5)
	for i := 0; i < 5; i++ {
		result[i] = steamAlphabet[codeInt%uint32(len(steamAlphabet))]
		codeInt /= uint32(len(steamAlphabet))
	}

	return string(result), nil
}

// RemainingSeconds returns the remaining seconds until the current code expires
func RemainingSeconds(period int, t time.Time) int {
	if period <= 0 {
		period = 30
	}
	rem := period - int(t.Unix()%int64(period))
	return rem
}

// Progress returns a 0.0 - 1.0 progress value of the current period
func Progress(period int, t time.Time) float64 {
	if period <= 0 {
		period = 30
	}
	elapsed := float64(t.Unix() % int64(period))
	return elapsed / float64(period)
}
