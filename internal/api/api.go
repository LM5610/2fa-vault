package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	_ "image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
	tuotoo "github.com/tuotoo/qrcode"

	"twofa-vault/pkg/migration"
	"twofa-vault/pkg/totp"
	"twofa-vault/pkg/vault"
)

// Server handles all REST API requests
type Server struct {
	vault      *vault.Vault
	httpClient *http.Client
}

// NewServer creates a new API Server instance
func NewServer(v *vault.Vault) *Server {
	return &Server{
		vault: v,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// RegisterRoutes registers all REST and SSE endpoints on mux
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/vault/status", s.handleVaultStatus)
	mux.HandleFunc("/api/vault/init", s.handleVaultInit)
	mux.HandleFunc("/api/vault/unlock", s.handleVaultUnlock)
	mux.HandleFunc("/api/vault/lock", s.handleVaultLock)
	mux.HandleFunc("/api/vault/password", s.handleVaultPassword)
	mux.HandleFunc("/api/vault/autolock", s.handleVaultAutoLock)

	mux.HandleFunc("/api/accounts", s.handleAccounts)
	mux.HandleFunc("/api/accounts/", s.handleAccountItem)
	mux.HandleFunc("/api/tags", s.handleTags)

	mux.HandleFunc("/api/qr", s.handleQRCode)
	mux.HandleFunc("/api/qr/decode-url", s.handleQRDecodeURL)
	mux.HandleFunc("/api/qr/decode-file", s.handleQRDecodeFile)
	mux.HandleFunc("/api/proxy-image", s.handleProxyImage)

	mux.HandleFunc("/api/import", s.handleImport)
	mux.HandleFunc("/api/export", s.handleExport)
	mux.HandleFunc("/api/events", s.handleSSE)
}

func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func errorResponse(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, map[string]string{"error": message})
}

// Status endpoint
func (s *Server) handleVaultStatus(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]any{
		"initialized":       s.vault.IsInitialized(),
		"locked":            s.vault.IsLocked(),
		"auto_lock_minutes": s.vault.AutoLockMinutes(),
	})
}

// Init endpoint
func (s *Server) handleVaultInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if err := s.vault.Init(req.Password); err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "vault initialized successfully",
	})
}

// Unlock endpoint
func (s *Server) handleVaultUnlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if err := s.vault.Unlock(req.Password); err != nil {
		errorResponse(w, http.StatusUnauthorized, "incorrect master password")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "vault unlocked",
	})
}

// Lock endpoint
func (s *Server) handleVaultLock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	s.vault.Lock()
	jsonResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "vault locked",
	})
}

// Change master password
func (s *Server) handleVaultPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if err := s.vault.ChangePassword(req.OldPassword, req.NewPassword); err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "password updated successfully",
	})
}

// Update auto-lock duration
func (s *Server) handleVaultAutoLock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Minutes int `json:"minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if err := s.vault.SetAutoLockMinutes(req.Minutes); err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"minutes": req.Minutes,
	})
}

// Accounts collection handler (GET, POST)
func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	if s.vault.IsLocked() {
		errorResponse(w, http.StatusUnauthorized, "vault is locked")
		return
	}

	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query().Get("q")
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

// Account item handler (GET, PUT, DELETE, /hotp, /pin)
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

	// Sub-actions: /hotp, /pin, /uri
	if len(parts) > 1 {
		subAction := parts[1]
		switch subAction {
		case "hotp":
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

// Tags endpoint
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

// QR Code generator endpoint: /api/qr?data=...
func (s *Server) handleQRCode(w http.ResponseWriter, r *http.Request) {
	data := r.URL.Query().Get("data")
	if data == "" {
		errorResponse(w, http.StatusBadRequest, "missing data parameter")
		return
	}

	size := 256
	if s := r.URL.Query().Get("size"); s != "" {
		if val, err := strconv.Atoi(s); err == nil && val > 64 && val <= 1024 {
			size = val
		}
	}

	png, err := qrcode.Encode(data, qrcode.Medium, size)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("qr encode failed: %v", err))
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

// Decode QR from Image URL endpoint: POST /api/qr/decode-url
func (s *Server) handleQRDecodeURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid json body")
		return
	}

	rawURL := strings.TrimSpace(req.URL)
	if rawURL == "" {
		errorResponse(w, http.StatusBadRequest, "url cannot be empty")
		return
	}

	var imgBytes []byte
	var err error

	// Handle Data URI (data:image/...;base64,...)
	if strings.HasPrefix(rawURL, "data:image/") {
		commaIdx := strings.Index(rawURL, ",")
		if commaIdx == -1 {
			errorResponse(w, http.StatusBadRequest, "invalid data uri format")
			return
		}
		imgBytes, err = base64.StdEncoding.DecodeString(rawURL[commaIdx+1:])
		if err != nil {
			errorResponse(w, http.StatusBadRequest, "invalid base64 in data uri")
			return
		}
	} else {
		// HTTP/HTTPS URL
		u, err := url.Parse(rawURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			errorResponse(w, http.StatusBadRequest, "url must be http or https")
			return
		}

		httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, rawURL, nil)
		if err != nil {
			errorResponse(w, http.StatusBadRequest, "invalid request")
			return
		}
		httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) 2FA-Vault/1.0")

		resp, err := s.httpClient.Do(httpReq)
		if err != nil {
			errorResponse(w, http.StatusBadRequest, fmt.Sprintf("fetch image failed: %v", err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errorResponse(w, http.StatusBadRequest, fmt.Sprintf("image server returned http %d", resp.StatusCode))
			return
		}

		// Limit max image size to 10MB
		imgBytes, err = io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
		if err != nil {
			errorResponse(w, http.StatusBadRequest, "read image failed")
			return
		}
	}

	result, err := decodeQRBytes(imgBytes)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, fmt.Sprintf("qr decode failed: %v", err))
		return
	}

	jsonResponse(w, http.StatusOK, result)
}

// Decode QR from File upload: POST /api/qr/decode-file
func (s *Server) handleQRDecodeFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "missing 'file' form field")
		return
	}
	defer file.Close()

	imgBytes, err := io.ReadAll(io.LimitReader(file, 10*1024*1024))
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "read file failed")
		return
	}

	result, err := decodeQRBytes(imgBytes)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, fmt.Sprintf("qr decode failed: %v", err))
		return
	}

	jsonResponse(w, http.StatusOK, result)
}

// Proxy image to bypass browser CORS: GET /api/proxy-image?url=...
func (s *Server) handleProxyImage(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		errorResponse(w, http.StatusBadRequest, "missing url")
		return
	}

	u, err := url.Parse(targetURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		errorResponse(w, http.StatusBadRequest, "invalid url scheme")
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 2FA-Vault/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		errorResponse(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// decodeQRBytes parses and extracts QR matrix from image bytes
func decodeQRBytes(data []byte) (map[string]any, error) {
	matrix, err := tuotoo.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("no QR code found or image is too blurry")
	}

	content := strings.TrimSpace(matrix.Content)
	res := map[string]any{
		"text": content,
	}

	// If content is an otpauth URI, parse details
	if strings.HasPrefix(content, "otpauth://") {
		parsed, err := totp.ParseURI(content)
		if err == nil {
			res["parsed"] = map[string]any{
				"issuer":       parsed.Issuer,
				"account_name": parsed.Account,
				"secret":       parsed.Secret,
				"algorithm":    parsed.Algorithm,
				"digits":       parsed.Digits,
				"period":       parsed.Period,
				"type":         parsed.Type,
				"counter":      parsed.Counter,
			}
		}
	}

	return res, nil
}

// Import endpoint
func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.vault.IsLocked() {
		errorResponse(w, http.StatusUnauthorized, "vault is locked")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "read body failed")
		return
	}

	accounts, err := migration.ParseImportText(body)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, fmt.Sprintf("parse import failed: %v", err))
		return
	}

	count, err := s.vault.ImportAccounts(accounts)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"success":  true,
		"imported": count,
	})
}

// Export endpoint
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if s.vault.IsLocked() {
		errorResponse(w, http.StatusUnauthorized, "vault is locked")
		return
	}

	format := r.URL.Query().Get("format")
	switch format {
	case "uri":
		accounts, err := s.vault.ListAccounts("", "")
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		rawAccs := make([]vault.Account, len(accounts))
		for i, a := range accounts {
			rawAccs[i] = a.Account
		}
		uris := migration.ExportToURIs(rawAccs)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=\"2fa-vault-export.txt\"")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(uris))

	default: // json
		plain, err := s.vault.ExportPlain()
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=\"2fa-vault-backup.json\"")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(plain)
	}
}

// SSE live countdown endpoint
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case now := <-ticker.C:
			rem := totp.RemainingSeconds(30, now)
			locked := s.vault.IsLocked()
			fmt.Fprintf(w, "data: {\"time\": %d, \"remaining\": %d, \"locked\": %t}\n\n", now.Unix(), rem, locked)
			flusher.Flush()
		}
	}
}
