package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	qrcode "github.com/skip2/go-qrcode"

	"twofa-vault/pkg/totp"
	"twofa-vault/pkg/vault"
)

// Accounts collection handler (GET, POST)
func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	if s.vault.IsLocked() {
		errorResponse(w, http.StatusUnauthorized, "vault is locked")
		return
	}

	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query().Get("q")
		if q == "" {
			q = r.URL.Query().Get("query")
		}
		tag := r.URL.Query().Get("tag")
		accounts, err := s.vault.ListAccounts(q, tag)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, accounts)

	case http.MethodPost:
		var req struct {
			URI         string         `json:"uri,omitempty"`
			Issuer      string         `json:"issuer"`
			AccountName string         `json:"account_name"`
			Secret      string         `json:"secret"`
			Algorithm   totp.Algorithm `json:"algorithm"`
			Digits      int            `json:"digits"`
			Period      int            `json:"period"`
			Type        totp.OTPType   `json:"type"`
			Counter     uint64         `json:"counter"`
			Tags        []string       `json:"tags"`
			Notes       string         `json:"notes"`
			Pinned      bool           `json:"pinned"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errorResponse(w, http.StatusBadRequest, "invalid json body")
			return
		}

		// If URI is passed, parse URI first
		if strings.TrimSpace(req.URI) != "" {
			parsed, err := totp.ParseURI(req.URI)
			if err != nil {
				errorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid otpauth URI: %v", err))
				return
			}
			req.Issuer = parsed.Issuer
			req.AccountName = parsed.Account
			req.Secret = parsed.Secret
			req.Algorithm = parsed.Algorithm
			req.Digits = parsed.Digits
			req.Period = parsed.Period
			req.Type = parsed.Type
			req.Counter = parsed.Counter
		}

		newAcc, err := s.vault.AddAccount(vault.Account{
			Issuer:      req.Issuer,
			AccountName: req.AccountName,
			Secret:      req.Secret,
			Algorithm:   req.Algorithm,
			Digits:      req.Digits,
			Period:      req.Period,
			Type:        req.Type,
			Counter:     req.Counter,
			Tags:        req.Tags,
			Notes:       req.Notes,
			Pinned:      req.Pinned,
		})
		if err != nil {
			errorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		jsonResponse(w, http.StatusCreated, newAcc)

	default:
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// Account item handler (GET, PUT, DELETE, /hotp, /increment, /pin, /uri, /qr)
func (s *Server) handleAccountItem(w http.ResponseWriter, r *http.Request) {
	if s.vault.IsLocked() {
		errorResponse(w, http.StatusUnauthorized, "vault is locked")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/accounts/")
	parts := strings.Split(path, "/")
	id := parts[0]
	if id == "" {
		errorResponse(w, http.StatusBadRequest, "missing account id")
		return
	}

	// Sub-actions: /hotp, /increment, /pin, /uri, /qr
	if len(parts) > 1 {
		subAction := parts[1]
		switch subAction {
		case "hotp", "increment":
			if r.Method != http.MethodPost {
				errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			code, counter, err := s.vault.IncrementHOTP(id)
			if err != nil {
				errorResponse(w, http.StatusBadRequest, err.Error())
				return
			}
			jsonResponse(w, http.StatusOK, map[string]any{
				"code":    code,
				"counter": counter,
			})
			return

		case "pin":
			if r.Method != http.MethodPost {
				errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			acc, err := s.vault.GetAccount(id)
			if err != nil {
				errorResponse(w, http.StatusNotFound, err.Error())
				return
			}
			acc.Pinned = !acc.Pinned
			updated, err := s.vault.UpdateAccount(*acc)
			if err != nil {
				errorResponse(w, http.StatusInternalServerError, err.Error())
				return
			}
			jsonResponse(w, http.StatusOK, updated)
			return

		case "uri":
			acc, err := s.vault.GetAccount(id)
			if err != nil {
				errorResponse(w, http.StatusNotFound, err.Error())
				return
			}
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
			jsonResponse(w, http.StatusOK, map[string]string{"uri": uri})
			return

		case "qr":
			acc, err := s.vault.GetAccount(id)
			if err != nil {
				errorResponse(w, http.StatusNotFound, err.Error())
				return
			}
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
			png, err := qrcode.Encode(uri, qrcode.Medium, 256)
			if err != nil {
				errorResponse(w, http.StatusInternalServerError, err.Error())
				return
			}
			qrData := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
			jsonResponse(w, http.StatusOK, map[string]string{
				"qr_data":     qrData,
				"otpauth_uri": uri,
			})
			return

		default:
			errorResponse(w, http.StatusNotFound, "unknown sub-action")
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		acc, err := s.vault.GetAccount(id)
		if err != nil {
			errorResponse(w, http.StatusNotFound, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, acc)

	case http.MethodPut:
		var acc vault.Account
		if err := json.NewDecoder(r.Body).Decode(&acc); err != nil {
			errorResponse(w, http.StatusBadRequest, "invalid json body")
			return
		}
		acc.ID = id
		updated, err := s.vault.UpdateAccount(acc)
		if err != nil {
			errorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, updated)

	case http.MethodDelete:
		if err := s.vault.DeleteAccount(id); err != nil {
			errorResponse(w, http.StatusNotFound, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, map[string]any{"success": true})

	default:
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// Tags endpoint: GET /api/tags
func (s *Server) handleTags(w http.ResponseWriter, r *http.Request) {
	if s.vault.IsLocked() {
		errorResponse(w, http.StatusUnauthorized, "vault is locked")
		return
	}
	tags, err := s.vault.Tags()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, tags)
}
