package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"twofa-vault/pkg/vault"
)

func TestAPI_Flow(t *testing.T) {
	tmpDir := t.TempDir()
	vaultPath := filepath.Join(tmpDir, "api_vault.enc")

	v := vault.New(vaultPath)
	server := NewServer(v)

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	// 1. Check initial status
	req := httptest.NewRequest(http.MethodGet, "/api/vault/status", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var statusResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	if statusResp["initialized"] != false {
		t.Fatalf("expected initialized false, got %v", statusResp["initialized"])
	}

	// 2. Init vault
	initBody := `{"password":"MasterPassword123"}`
	req = httptest.NewRequest(http.MethodPost, "/api/vault/init", bytes.NewBufferString(initBody))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("init expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 3. Add account
	accBody := `{
		"issuer": "GitHub",
		"account_name": "developer",
		"secret": "JBSWY3DPEHPK3PXP",
		"tags": ["Dev", "Tech"]
	}`
	req = httptest.NewRequest(http.MethodPost, "/api/accounts", bytes.NewBufferString(accBody))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("add account expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var createdAcc vault.Account
	if err := json.Unmarshal(w.Body.Bytes(), &createdAcc); err != nil {
		t.Fatalf("unmarshal created account failed: %v", err)
	}

	// 4. List accounts
	req = httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list accounts expected 200, got %d", w.Code)
	}

	var list []vault.AccountDTO
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal list failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 account, got %d", len(list))
	}
	if list[0].Code == "" {
		t.Fatal("expected non-empty OTP code")
	}

	// 5. QR code generation test
	req = httptest.NewRequest(http.MethodGet, "/api/qr?data=otpauth://totp/Test?secret=JBSWY3DPEHPK3PXP", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("qr expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("expected image/png content type, got %s", w.Header().Get("Content-Type"))
	}
	if w.Body.Len() == 0 {
		t.Fatal("expected non-empty PNG bytes")
	}

	// 6. Lock vault
	req = httptest.NewRequest(http.MethodPost, "/api/vault/lock", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("lock expected 200, got %d", w.Code)
	}

	// 7. Verify locked rejection
	req = httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when locked, got %d", w.Code)
	}
}
