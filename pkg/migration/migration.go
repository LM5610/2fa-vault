package migration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"twofa-vault/pkg/totp"
	"twofa-vault/pkg/vault"
)

// AegisBackup represents Aegis Authenticator export format
type AegisBackup struct {
	Version int `json:"version"`
	Header  struct {
		Slots []any `json:"slots"`
	} `json:"header"`
	DB struct {
		Version int          `json:"version"`
		Entries []AegisEntry `json:"entries"`
	} `json:"db"`
}

type AegisEntry struct {
	Type   string   `json:"type"`
	Name   string   `json:"name"`
	Issuer string   `json:"issuer"`
	Note   string   `json:"note"`
	Group  string   `json:"group"`
	Info   AegisInfo `json:"info"`
}

type AegisInfo struct {
	Secret  string `json:"secret"`
	Algo    string `json:"algo"`
	Digits  int    `json:"digits"`
	Period  int    `json:"period"`
	Counter uint64 `json:"counter"`
}

// ParseImportText inspects raw input text and parses accounts from either:
// 1. 2FA Vault native JSON format
// 2. Aegis JSON format
// 3. Google Authenticator migration URIs (otpauth-migration://)
// 4. Lines of otpauth:// URIs
func ParseImportText(raw []byte) ([]vault.Account, error) {
	str := strings.TrimSpace(string(raw))

	// Try native VaultData JSON first
	var native vault.VaultData
	if err := json.Unmarshal(raw, &native); err == nil && len(native.Accounts) > 0 {
		return native.Accounts, nil
	}

	// Try Aegis JSON format
	var aegis AegisBackup
	if err := json.Unmarshal(raw, &aegis); err == nil && len(aegis.DB.Entries) > 0 {
		var accounts []vault.Account
		for _, e := range aegis.DB.Entries {
			algo := totp.AlgorithmSHA1
			switch strings.ToUpper(e.Info.Algo) {
			case "SHA256":
				algo = totp.AlgorithmSHA256
			case "SHA512":
				algo = totp.AlgorithmSHA512
			}

			accType := totp.TypeTOTP
			if strings.ToLower(e.Type) == "hotp" {
				accType = totp.TypeHOTP
			} else if strings.ToLower(e.Type) == "steam" {
				accType = totp.TypeSteam
			}

			tags := []string{}
			if e.Group != "" {
				tags = append(tags, e.Group)
			}

			cleanSec, err := totp.CleanSecret(e.Info.Secret)
			if err != nil {
				continue
			}

			accounts = append(accounts, vault.Account{
				Issuer:      e.Issuer,
				AccountName: e.Name,
				Secret:      cleanSec,
				Algorithm:   algo,
				Digits:      e.Info.Digits,
				Period:      e.Info.Period,
				Type:        accType,
				Counter:     e.Info.Counter,
				Tags:        tags,
				Notes:       e.Note,
			})
		}
		if len(accounts) > 0 {
			return accounts, nil
		}
	}

	// Try Google Authenticator migration URIs
	if strings.Contains(str, "otpauth-migration://") {
		var accounts []vault.Account
		var lastErr error
		scanner := bufio.NewScanner(strings.NewReader(str))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			idx := strings.Index(line, "otpauth-migration://")
			if idx == -1 {
				// Check for standard otpauth:// in mixed inputs
				if strings.HasPrefix(line, "otpauth://") {
					if parsed, err := totp.ParseURI(line); err == nil {
						accounts = append(accounts, vault.Account{
							Issuer:      parsed.Issuer,
							AccountName: parsed.Account,
							Secret:      parsed.Secret,
							Algorithm:   parsed.Algorithm,
							Digits:      parsed.Digits,
							Period:      parsed.Period,
							Type:        parsed.Type,
							Counter:     parsed.Counter,
						})
					}
				}
				continue
			}

			uriPart := line[idx:]
			if endIdx := strings.IndexAny(uriPart, " \t\r\n\"'<>()[]"); endIdx != -1 {
				uriPart = uriPart[:endIdx]
			}

			parsedAccs, err := ParseGoogleMigrationURI(uriPart)
			if err != nil {
				lastErr = err
				continue
			}
			accounts = append(accounts, parsedAccs...)
		}
		if len(accounts) > 0 {
			return accounts, nil
		}
		if lastErr != nil {
			return nil, fmt.Errorf("failed to parse google migration URI: %w", lastErr)
		}
	}

	// Line-by-line otpauth:// parsing
	var accounts []vault.Account
	scanner := bufio.NewScanner(strings.NewReader(str))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "otpauth://") {
			continue
		}

		parsed, err := totp.ParseURI(line)
		if err != nil {
			continue
		}

		accounts = append(accounts, vault.Account{
			Issuer:      parsed.Issuer,
			AccountName: parsed.Account,
			Secret:      parsed.Secret,
			Algorithm:   parsed.Algorithm,
			Digits:      parsed.Digits,
			Period:      parsed.Period,
			Type:        parsed.Type,
			Counter:     parsed.Counter,
		})
	}

	return accounts, nil
}

// ExportToURIs formats accounts as newline-separated otpauth:// URIs
func ExportToURIs(accounts []vault.Account) string {
	var sb strings.Builder
	for _, acc := range accounts {
		uri := totp.BuildURI(totp.ParsedOTP{
			Type:      acc.Type,
			Issuer:    acc.Issuer,
			Account:   acc.AccountName,
			Secret:    acc.Secret,
			Algorithm: acc.Algorithm,
			Digits:    acc.Digits,
			Period:    acc.Period,
			Counter:   acc.Counter,
		})
		sb.WriteString(uri)
		sb.WriteString("\n")
	}
	return sb.String()
}
