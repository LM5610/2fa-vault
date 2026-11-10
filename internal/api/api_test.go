package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	qrcode "github.com/skip2/go-qrcode"

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

	// 6. QR decode-url test with generated QR PNG
	qrPNG, err := qrcode.Encode("otpauth://totp/GitHub:testuser?secret=JBSWY3DPEHPK3PXP&issuer=GitHub", qrcode.Medium, 256)
	if err != nil {
		t.Fatalf("generate qr png failed: %v", err)
	}

	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrPNG)
	decodeReqBody, _ := json.Marshal(map[string]string{"url": dataURI})
	req = httptest.NewRequest(http.MethodPost, "/api/qr/decode-url", bytes.NewReader(decodeReqBody))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("qr decode-url expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var decodeResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &decodeResp); err != nil {
		t.Fatalf("unmarshal decode resp failed: %v", err)
	}
	if !strings.Contains(decodeResp["text"].(string), "GitHub:testuser") {
		t.Fatalf("expected GitHub:testuser in decoded text, got %v", decodeResp["text"])
	}

	// 7. Test OTP Live Preview
	req = httptest.NewRequest(http.MethodGet, "/api/otp/preview?secret=JBSWY3DPEHPK3PXP&digits=6&period=30", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("otp preview expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var previewResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &previewResp); err != nil {
		t.Fatalf("unmarshal preview failed: %v", err)
	}
	if previewResp["valid"] != true {
		t.Fatalf("expected valid preview, got %v", previewResp)
	}
	if len(previewResp["code"].(string)) != 6 {
		t.Fatalf("expected 6-digit code, got %v", previewResp["code"])
	}

	// 8. Lock vault
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
