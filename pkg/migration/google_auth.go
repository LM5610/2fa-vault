package migration

import (
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"twofa-vault/pkg/totp"
	"twofa-vault/pkg/vault"
)

// Protobuf wire types
const (
	wireVarint          = 0
	wireFixed64         = 1
	wireLengthDelimited = 2
	wireStartGroup      = 3
	wireEndGroup        = 4
	wireFixed32         = 5
)

// Algorithm constants for Google Authenticator migration
const (
	GoogleAlgoUnspecified = 0
	GoogleAlgoSHA1        = 1
	GoogleAlgoSHA256      = 2
	GoogleAlgoSHA512      = 3
	GoogleAlgoMD5         = 4
)

// Digit count constants for Google Authenticator migration
const (
	GoogleDigitUnspecified = 0
	GoogleDigitSix         = 1
	GoogleDigitEight       = 2
)

// OTP type constants for Google Authenticator migration
const (
	GoogleOtpTypeUnspecified = 0
	GoogleOtpTypeHOTP        = 1
	GoogleOtpTypeTOTP        = 2
)

// OtpParameters represents a single OTP entry inside Google Authenticator migration payload
type OtpParameters struct {
	Secret    []byte
	Name      string
	Issuer    string
	Algorithm int32
	Digits    int32
	Type      int32
	Counter   int64
}

// MigrationPayload represents Google Authenticator export protobuf payload
type MigrationPayload struct {
	OtpParameters []*OtpParameters
	Version       int32
	BatchSize     int32
	BatchIndex    int32
	BatchID       int32
}

// decodeVarint reads a protobuf varint from buf.
func decodeVarint(buf []byte) (uint64, int, error) {
	var val uint64
	var shift uint
	for i, b := range buf {
		if i >= 10 {
			return 0, 0, errors.New("protobuf: varint overflow")
		}
		val |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			return val, i + 1, nil
		}
		shift += 7
	}
	return 0, 0, io.ErrUnexpectedEOF
}

// encodeVarint encodes val as a protobuf varint.
func encodeVarint(val uint64) []byte {
	var buf []byte
	for val >= 0x80 {
		buf = append(buf, byte(val&0x7F|0x80))
		val >>= 7
	}
	buf = append(buf, byte(val&0x7F))
	return buf
}

func encodeTag(fieldNum int, wire int) []byte {
	return encodeVarint(uint64((fieldNum << 3) | wire))
}

func encodeLengthDelimited(fieldNum int, data []byte) []byte {
	var out []byte
	out = append(out, encodeTag(fieldNum, wireLengthDelimited)...)
	out = append(out, encodeVarint(uint64(len(data)))...)
	out = append(out, data...)
	return out
}

// skipField skips over an unknown field based on its wire type.
func skipField(wire int, buf []byte) (int, error) {
	switch wire {
	case wireVarint:
		_, n, err := decodeVarint(buf)
		return n, err
	case wireFixed64:
		if len(buf) < 8 {
			return 0, io.ErrUnexpectedEOF
		}
		return 8, nil
	case wireLengthDelimited:
		l, n, err := decodeVarint(buf)
		if err != nil {
			return 0, err
		}
		total := n + int(l)
		if int(l) < 0 || total > len(buf) {
			return 0, io.ErrUnexpectedEOF
		}
		return total, nil
	case wireFixed32:
		if len(buf) < 4 {
			return 0, io.ErrUnexpectedEOF
		}
		return 4, nil
	default:
		return 0, fmt.Errorf("protobuf: unsupported wire type %d", wire)
	}
}

// parseOtpParameters parses an individual OtpParameters protobuf message.
func parseOtpParameters(data []byte) (*OtpParameters, error) {
	param := &OtpParameters{}
	idx := 0
	for idx < len(data) {
		tag, n, err := decodeVarint(data[idx:])
		if err != nil {
			return nil, err
		}
		idx += n

		wire := int(tag & 0x07)
		field := int(tag >> 3)

		switch field {
		case 1: // secret (bytes)
			if wire != wireLengthDelimited {
				return nil, fmt.Errorf("field 1 secret unexpected wire type %d", wire)
			}
			l, n, err := decodeVarint(data[idx:])
			if err != nil {
				return nil, err
			}
			idx += n
			if int(l) < 0 || idx+int(l) > len(data) {
				return nil, io.ErrUnexpectedEOF
			}
			param.Secret = make([]byte, l)
			copy(param.Secret, data[idx:idx+int(l)])
			idx += int(l)

		case 2: // name (string)
			if wire != wireLengthDelimited {
				return nil, fmt.Errorf("field 2 name unexpected wire type %d", wire)
			}
			l, n, err := decodeVarint(data[idx:])
			if err != nil {
				return nil, err
			}
			idx += n
			if int(l) < 0 || idx+int(l) > len(data) {
				return nil, io.ErrUnexpectedEOF
			}
			param.Name = string(data[idx : idx+int(l)])
			idx += int(l)

		case 3: // issuer (string)
			if wire != wireLengthDelimited {
				return nil, fmt.Errorf("field 3 issuer unexpected wire type %d", wire)
			}
			l, n, err := decodeVarint(data[idx:])
			if err != nil {
				return nil, err
			}
			idx += n
			if int(l) < 0 || idx+int(l) > len(data) {
				return nil, io.ErrUnexpectedEOF
			}
			param.Issuer = string(data[idx : idx+int(l)])
			idx += int(l)

		case 4: // algorithm (enum/int32)
			if wire != wireVarint {
				return nil, fmt.Errorf("field 4 algorithm unexpected wire type %d", wire)
			}
			val, n, err := decodeVarint(data[idx:])
			if err != nil {
				return nil, err
			}
			idx += n
			param.Algorithm = int32(val)

		case 5: // digits (enum/int32)
			if wire != wireVarint {
				return nil, fmt.Errorf("field 5 digits unexpected wire type %d", wire)
			}
			val, n, err := decodeVarint(data[idx:])
			if err != nil {
				return nil, err
			}
			idx += n
			param.Digits = int32(val)

		case 6: // type (enum/int32)
			if wire != wireVarint {
				return nil, fmt.Errorf("field 6 type unexpected wire type %d", wire)
			}
			val, n, err := decodeVarint(data[idx:])
			if err != nil {
				return nil, err
			}
			idx += n
			param.Type = int32(val)

		case 7: // counter (int64)
			if wire != wireVarint {
				return nil, fmt.Errorf("field 7 counter unexpected wire type %d", wire)
			}
			val, n, err := decodeVarint(data[idx:])
			if err != nil {
				return nil, err
			}
			idx += n
			param.Counter = int64(val)

		default:
			skip, err := skipField(wire, data[idx:])
			if err != nil {
				return nil, err
			}
			idx += skip
		}
	}
	return param, nil
}

// DecodeMigrationPayload parses raw protobuf bytes into a MigrationPayload struct.
func DecodeMigrationPayload(data []byte) (*MigrationPayload, error) {
	payload := &MigrationPayload{}
	idx := 0
	for idx < len(data) {
		tag, n, err := decodeVarint(data[idx:])
		if err != nil {
			return nil, err
		}
		idx += n

		wire := int(tag & 0x07)
		field := int(tag >> 3)

		switch field {
		case 1: // repeated OtpParameters otp_parameters = 1
			if wire != wireLengthDelimited {
				return nil, fmt.Errorf("field 1 otp_parameters unexpected wire type %d", wire)
			}
			l, n, err := decodeVarint(data[idx:])
			if err != nil {
				return nil, err
			}
			idx += n
			if int(l) < 0 || idx+int(l) > len(data) {
				return nil, io.ErrUnexpectedEOF
			}
			param, err := parseOtpParameters(data[idx : idx+int(l)])
			if err != nil {
				return nil, fmt.Errorf("parse OtpParameters: %w", err)
			}
			payload.OtpParameters = append(payload.OtpParameters, param)
			idx += int(l)

		case 2: // int32 version = 2
			if wire != wireVarint {
				return nil, fmt.Errorf("field 2 version unexpected wire type %d", wire)
			}
			val, n, err := decodeVarint(data[idx:])
			if err != nil {
				return nil, err
			}
			idx += n
			payload.Version = int32(val)

		case 3: // int32 batch_size = 3
			if wire != wireVarint {
				return nil, fmt.Errorf("field 3 batch_size unexpected wire type %d", wire)
			}
			val, n, err := decodeVarint(data[idx:])
			if err != nil {
				return nil, err
			}
			idx += n
			payload.BatchSize = int32(val)

		case 4: // int32 batch_index = 4
			if wire != wireVarint {
				return nil, fmt.Errorf("field 4 batch_index unexpected wire type %d", wire)
			}
			val, n, err := decodeVarint(data[idx:])
			if err != nil {
				return nil, err
			}
			idx += n
			payload.BatchIndex = int32(val)

		case 5: // int32 batch_id = 5
			if wire != wireVarint {
				return nil, fmt.Errorf("field 5 batch_id unexpected wire type %d", wire)
			}
			val, n, err := decodeVarint(data[idx:])
			if err != nil {
				return nil, err
			}
			idx += n
			payload.BatchID = int32(val)

		default:
			skip, err := skipField(wire, data[idx:])
			if err != nil {
				return nil, err
			}
			idx += skip
		}
	}
	return payload, nil
}

// MarshalProtobuf serializes OtpParameters to protobuf wire format.
func (p *OtpParameters) MarshalProtobuf() []byte {
	var out []byte
	if len(p.Secret) > 0 {
		out = append(out, encodeLengthDelimited(1, p.Secret)...)
	}
	if p.Name != "" {
		out = append(out, encodeLengthDelimited(2, []byte(p.Name))...)
	}
	if p.Issuer != "" {
		out = append(out, encodeLengthDelimited(3, []byte(p.Issuer))...)
	}
	if p.Algorithm != 0 {
		out = append(out, encodeTag(4, wireVarint)...)
		out = append(out, encodeVarint(uint64(p.Algorithm))...)
	}
	if p.Digits != 0 {
		out = append(out, encodeTag(5, wireVarint)...)
		out = append(out, encodeVarint(uint64(p.Digits))...)
	}
	if p.Type != 0 {
		out = append(out, encodeTag(6, wireVarint)...)
		out = append(out, encodeVarint(uint64(p.Type))...)
	}
	if p.Counter != 0 {
		out = append(out, encodeTag(7, wireVarint)...)
		out = append(out, encodeVarint(uint64(p.Counter))...)
	}
	return out
}

// MarshalProtobuf serializes MigrationPayload to protobuf wire format.
func (m *MigrationPayload) MarshalProtobuf() []byte {
	var out []byte
	for _, param := range m.OtpParameters {
		encodedParam := param.MarshalProtobuf()
		out = append(out, encodeLengthDelimited(1, encodedParam)...)
	}
	if m.Version != 0 {
		out = append(out, encodeTag(2, wireVarint)...)
		out = append(out, encodeVarint(uint64(m.Version))...)
	}
	if m.BatchSize != 0 {
		out = append(out, encodeTag(3, wireVarint)...)
		out = append(out, encodeVarint(uint64(m.BatchSize))...)
	}
	if m.BatchIndex != 0 {
		out = append(out, encodeTag(4, wireVarint)...)
		out = append(out, encodeVarint(uint64(m.BatchIndex))...)
	}
	if m.BatchID != 0 {
		out = append(out, encodeTag(5, wireVarint)...)
		out = append(out, encodeVarint(uint64(m.BatchID))...)
	}
	return out
}

// decodeBase64Payload robustly decodes standard or URL base64, with or without padding.
func decodeBase64Payload(raw string) ([]byte, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, errors.New("empty base64 string")
	}

	// Try standard and URL decoders directly first
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		if b, err := enc.DecodeString(s); err == nil && len(b) > 0 {
			return b, nil
		}
	}

	// Handle case where '+' characters were parsed as spaces ' ' in URL query strings
	if strings.Contains(s, " ") {
		sWithPlus := strings.ReplaceAll(s, " ", "+")
		for _, enc := range []*base64.Encoding{
			base64.StdEncoding,
			base64.RawStdEncoding,
		} {
			if b, err := enc.DecodeString(sWithPlus); err == nil && len(b) > 0 {
				return b, nil
			}
		}
	}

	// Normalize URL-safe characters and add standard padding
	norm := strings.ReplaceAll(s, "-", "+")
	norm = strings.ReplaceAll(norm, "_", "/")
	norm = strings.ReplaceAll(norm, " ", "+")
	if rem := len(norm) % 4; rem != 0 {
		norm += strings.Repeat("=", 4-rem)
	}

	return base64.StdEncoding.DecodeString(norm)
}

// ParseGoogleMigrationPayload converts raw protobuf bytes into []vault.Account.
func ParseGoogleMigrationPayload(data []byte) ([]vault.Account, error) {
	payload, err := DecodeMigrationPayload(data)
	if err != nil {
		return nil, fmt.Errorf("decode migration payload: %w", err)
	}

	var accounts []vault.Account
	for _, p := range payload.OtpParameters {
		if len(p.Secret) == 0 {
			continue
		}

		// Base32 encode the raw HMAC secret bytes
		rawB32 := base32.StdEncoding.EncodeToString(p.Secret)
		cleanSec, err := totp.CleanSecret(rawB32)
		if err != nil {
			continue
		}

		issuer := strings.TrimSpace(p.Issuer)
		account := strings.TrimSpace(p.Name)

		if idx := strings.Index(account, ":"); idx != -1 {
			prefix := strings.TrimSpace(account[:idx])
			account = strings.TrimSpace(account[idx+1:])
			if issuer == "" {
				issuer = prefix
			}
		}

		if account == "" {
			account = issuer
		}
		if account == "" {
			account = "Unnamed"
		}

		// Algorithm mapping: 0=unspecified, 1=SHA1, 2=SHA256, 3=SHA512, 4=MD5
		algo := totp.AlgorithmSHA1
		switch p.Algorithm {
		case GoogleAlgoSHA256:
			algo = totp.AlgorithmSHA256
		case GoogleAlgoSHA512:
			algo = totp.AlgorithmSHA512
		default:
			algo = totp.AlgorithmSHA1
		}

		// Digits mapping: 0=unspecified (6), 1=six (6), 2=eight (8)
		digits := 6
		if p.Digits == GoogleDigitEight {
			digits = 8
		}

		// Type mapping: 0=unspecified (TOTP), 1=HOTP, 2=TOTP
		accType := totp.TypeTOTP
		var counter uint64 = 0
		if p.Type == GoogleOtpTypeHOTP {
			accType = totp.TypeHOTP
			if p.Counter > 0 {
				counter = uint64(p.Counter)
			}
		}

		accounts = append(accounts, vault.Account{
			Issuer:      issuer,
			AccountName: account,
			Secret:      cleanSec,
			Algorithm:   algo,
			Digits:      digits,
			Period:      30,
			Type:        accType,
			Counter:     counter,
		})
	}

	return accounts, nil
}

// ParseGoogleMigrationURI parses an otpauth-migration:// URI into []vault.Account.
func ParseGoogleMigrationURI(uriStr string) ([]vault.Account, error) {
	uriStr = strings.TrimSpace(uriStr)
	if uriStr == "" {
		return nil, errors.New("empty migration URI")
	}

	u, err := url.Parse(uriStr)
	if err != nil {
		return nil, fmt.Errorf("invalid migration URI: %w", err)
	}

	if !strings.EqualFold(u.Scheme, "otpauth-migration") {
		return nil, errors.New("scheme must be 'otpauth-migration'")
	}

	// Extract data parameter
	dataParam := u.Query().Get("data")
	if dataParam == "" {
		// Fallback check in RawQuery
		for _, part := range strings.Split(u.RawQuery, "&") {
			if strings.HasPrefix(part, "data=") {
				unescaped, err := url.QueryUnescape(strings.TrimPrefix(part, "data="))
				if err == nil {
					dataParam = unescaped
				}
				break
			}
		}
	}

	if dataParam == "" {
		return nil, errors.New("missing data query parameter in migration URI")
	}

	rawPayload, err := decodeBase64Payload(dataParam)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 payload: %w", err)
	}

	return ParseGoogleMigrationPayload(rawPayload)
}

// BuildGoogleMigrationURI converts accounts into an otpauth-migration:// URI.
func BuildGoogleMigrationURI(accounts []vault.Account) (string, error) {
	payload := &MigrationPayload{
		Version:    1,
		BatchSize:  1,
		BatchIndex: 0,
		BatchID:    1,
	}

	for _, acc := range accounts {
		cleanSec, err := totp.CleanSecret(acc.Secret)
		if err != nil {
			return "", fmt.Errorf("invalid secret for %s: %w", acc.AccountName, err)
		}
		secretBytes, err := base32.StdEncoding.DecodeString(cleanSec)
		if err != nil {
			return "", fmt.Errorf("base32 decode failed for %s: %w", acc.AccountName, err)
		}

		name := acc.AccountName
		if acc.Issuer != "" {
			name = fmt.Sprintf("%s:%s", acc.Issuer, acc.AccountName)
		}

		algo := int32(GoogleAlgoSHA1)
		switch acc.Algorithm {
		case totp.AlgorithmSHA256:
			algo = GoogleAlgoSHA256
		case totp.AlgorithmSHA512:
			algo = GoogleAlgoSHA512
		}

		digits := int32(GoogleDigitSix)
		if acc.Digits == 8 {
			digits = GoogleDigitEight
		}

		otpType := int32(GoogleOtpTypeTOTP)
		if acc.Type == totp.TypeHOTP {
			otpType = GoogleOtpTypeHOTP
		}

		payload.OtpParameters = append(payload.OtpParameters, &OtpParameters{
			Secret:    secretBytes,
			Name:      name,
			Issuer:    acc.Issuer,
			Algorithm: algo,
			Digits:    digits,
			Type:      otpType,
			Counter:   int64(acc.Counter),
		})
	}

	protoBytes := payload.MarshalProtobuf()
	b64 := base64.StdEncoding.EncodeToString(protoBytes)

	u := url.URL{
		Scheme: "otpauth-migration",
		Host:   "offline",
	}
	q := u.Query()
	q.Set("data", b64)
	u.RawQuery = q.Encode()

	return u.String(), nil
}
