package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"twofa-vault/pkg/vault"
)

func setupTestServer(t *testing.T) (*Server, *http.ServeMux, *vault.Vault) {
	tmpDir := t.TempDir()
	vaultPath := filepath.Join(tmpDir, "vault.enc")
	v := vault.New(vaultPath)
	server := NewServer(v)
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)
	return server, mux, v
}

func setupMockWebDAV(expectedUser, expectedPass string) (*httptest.Server, map[string][]byte, *sync.Mutex) {
	var mu sync.Mutex
	files := make(map[string][]byte)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != expectedUser || pass != expectedPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="webdav"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		mu.Lock()
		defer mu.Unlock()

		switch r.Method {
		case "PROPFIND":
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?><D:multistatus xmlns:D="DAV:"></D:multistatus>`))
		case "MKCOL":
			w.WriteHeader(http.StatusCreated)
		case http.MethodPut:
			data, _ := io.ReadAll(r.Body)
			files[r.URL.Path] = data
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			data, exists := files[r.URL.Path]
			if !exists {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	return s, files, &mu
}

func TestWebDAV_ConfigEndpoints(t *testing.T) {
	_, mux, _ := setupTestServer(t)

	// 1. Initial GET /api/webdav/config
	req := httptest.NewRequest(http.MethodGet, "/api/webdav/config", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var cfgResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &cfgResp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if cfgResp["remote_dir"] != "/2fa-vault" {
		t.Errorf("expected remote_dir '/2fa-vault', got %v", cfgResp["remote_dir"])
	}
	if cfgResp["has_password"] != false {
		t.Errorf("expected has_password false, got %v", cfgResp["has_password"])
	}

	// 2. Save config: POST /api/webdav/config
	savePayload := `{"url":"https://dav.jianguoyun.com/dav/","username":"alice@test.com","password":"secret_app_token_123","auto_sync":true,"remote_dir":"/my-2fa"}`
	req = httptest.NewRequest(http.MethodPost, "/api/webdav/config", bytes.NewBufferString(savePayload))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("save config expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var savedResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &savedResp)
	if savedResp["username"] != "alice@test.com" {
		t.Errorf("expected username alice@test.com, got %v", savedResp["username"])
	}
	if savedResp["password"] != "••••••••" {
		t.Errorf("expected masked password, got %v", savedResp["password"])
	}
	if savedResp["has_password"] != true {
		t.Errorf("expected has_password true")
	}

	// 3. Update config without re-entering password
	updatePayload := `{"url":"https://dav.jianguoyun.com/dav/","username":"alice@test.com","password":"","auto_sync":false,"remote_dir":"/my-2fa-new"}`
	req = httptest.NewRequest(http.MethodPost, "/api/webdav/config", bytes.NewBufferString(updatePayload))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update config expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var updatedResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &updatedResp)
	if updatedResp["has_password"] != true {
		t.Errorf("expected password to be preserved when omitted in update")
	}
	if updatedResp["auto_sync"] != false {
		t.Errorf("expected auto_sync to be updated to false")
	}
}

func TestWebDAV_TestAndBackupAndRestore(t *testing.T) {
	mockServer, mockFiles, mockMu := setupMockWebDAV("davuser", "davpass")
	defer mockServer.Close()

	_, mux, v := setupTestServer(t)

	// 1. Initialize vault
	if err := v.Init("MasterKey999!"); err != nil {
		t.Fatalf("init vault failed: %v", err)
	}
	_, err := v.AddAccount(vault.Account{
		Issuer:      "TestService",
		AccountName: "user@test.org",
		Secret:      "JBSWY3DPEHPK3PXP",
	})
	if err != nil {
		t.Fatalf("create account failed: %v", err)
	}

	// 2. Configure WebDAV
	cfgPayload := map[string]any{
		"url":        mockServer.URL + "/dav",
		"username":   "davuser",
		"password":   "davpass",
		"auto_sync":  true,
		"remote_dir": "/2fa-vault",
	}
	cfgBytes, _ := json.Marshal(cfgPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/webdav/config", bytes.NewReader(cfgBytes))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("failed to configure webdav: %s", w.Body.String())
	}

	// 3. Test Connection
	req = httptest.NewRequest(http.MethodPost, "/api/webdav/test", bytes.NewBufferString(`{}`))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("test connection failed: %d %s", w.Code, w.Body.String())
	}

	// 4. Test Manual Backup
	req = httptest.NewRequest(http.MethodPost, "/api/webdav/backup", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("backup failed: %d %s", w.Code, w.Body.String())
	}

	mockMu.Lock()
	uploadedData, exists := mockFiles["/dav/2fa-vault/vault.enc"]
	mockMu.Unlock()
	if !exists || len(uploadedData) == 0 {
		t.Fatalf("backup file not uploaded to mock webdav")
	}

	// 5. Modify local account so we can verify restore overwrites it
	accs, _ := v.ListAccounts("", "")
	if len(accs) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accs))
	}
	_ = v.DeleteAccount(accs[0].ID)
	afterDelete, _ := v.ListAccounts("", "")
	if len(afterDelete) != 0 {
		t.Fatalf("expected 0 accounts after delete, got %d", len(afterDelete))
	}

	// 6. Test Restore
	req = httptest.NewRequest(http.MethodPost, "/api/webdav/restore", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("restore failed: %d %s", w.Code, w.Body.String())
	}

	// Verify account is restored
	restoredAccs, err := v.ListAccounts("", "")
	if err != nil {
		t.Fatalf("failed to list accounts after restore: %v", err)
	}
	if len(restoredAccs) != 1 || restoredAccs[0].Issuer != "TestService" {
		t.Fatalf("expected restored account 'TestService', got %+v", restoredAccs)
	}
}

func TestWebDAV_AutoSyncOnVaultModification(t *testing.T) {
	mockServer, mockFiles, mockMu := setupMockWebDAV("davuser", "davpass")
	defer mockServer.Close()

	server, mux, v := setupTestServer(t)

	// Configure WebDAV with AutoSync enabled
	cfgPayload := map[string]any{
		"url":        mockServer.URL + "/dav",
		"username":   "davuser",
		"password":   "davpass",
		"auto_sync":  true,
		"remote_dir": "/2fa-vault",
	}
	cfgBytes, _ := json.Marshal(cfgPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/webdav/config", bytes.NewReader(cfgBytes))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("failed to configure webdav: %s", w.Body.String())
	}

	// Initialize vault
	if err := v.Init("MasterKey999!"); err != nil {
		t.Fatalf("init vault failed: %v", err)
	}

	// Adding an account modifies the vault and calls saveUnlocked() -> triggers AutoSync
	_, err := v.AddAccount(vault.Account{
		Issuer:      "AutoSyncAccount",
		AccountName: "sync@test.org",
		Secret:      "JBSWY3DPEHPK3PXP",
	})
	if err != nil {
		t.Fatalf("create account failed: %v", err)
	}

	// Wait for debounce timer (1s) + upload to finish
	uploaded := false
	for i := 0; i < 25; i++ {
		time.Sleep(100 * time.Millisecond)
		mockMu.Lock()
		data, exists := mockFiles["/dav/2fa-vault/vault.enc"]
		mockMu.Unlock()
		if exists && len(data) > 0 {
			uploaded = true
			break
		}
	}

	if !uploaded {
		server.webdavState.mu.RLock()
		status := server.webdavState.config.LastStatus
		lastErr := server.webdavState.config.LastError
		server.webdavState.mu.RUnlock()
		t.Fatalf("auto-sync did not upload vault within timeout; status=%s, err=%s", status, lastErr)
	}
}
