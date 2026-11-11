package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"twofa-vault/pkg/totp"
	"twofa-vault/pkg/vault"
)

func TestTimeDrift_Endpoint(t *testing.T) {
	tmpDir := t.TempDir()
	v := vault.New(filepath.Join(tmpDir, "vault.enc"))
	server := NewServer(v)

	// Set up mock HTTP time server
	mockTime := time.Now().Add(7 * time.Second).UTC()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Date", mockTime.Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	server.ntpClient = &totp.NTPClient{
		NTPServers:  []string{"127.0.0.1:1"},
		HTTPServers: []string{ts.URL},
		Timeout:     200 * time.Millisecond,
		HTTPClient:  ts.Client(),
	}

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	// 1. Valid GET /api/time/drift
	req := httptest.NewRequest(http.MethodGet, "/api/time/drift", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp TimeDriftResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if resp.SystemTime == "" || resp.NetworkTime == "" {
		t.Errorf("system_time or network_time is empty")
	}
	if !resp.IsDrifted {
		t.Errorf("expected is_drifted true for ~7s offset")
	}
	if resp.DriftSeconds < 5.0 {
		t.Errorf("expected drift_seconds > 5.0, got %f", resp.DriftSeconds)
	}

	// 2. Disallowed method POST /api/time/drift
	reqPost := httptest.NewRequest(http.MethodPost, "/api/time/drift", nil)
	wPost := httptest.NewRecorder()
	mux.ServeHTTP(wPost, reqPost)
	if wPost.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 Method Not Allowed, got %d", wPost.Code)
	}

	// 3. Graceful offline fallback
	server.ntpClient = &totp.NTPClient{
		NTPServers:  []string{"127.0.0.1:1"},
		HTTPServers: []string{"http://127.0.0.1:1"},
		Timeout:     50 * time.Millisecond,
		HTTPClient:  &http.Client{Timeout: 50 * time.Millisecond},
	}
	reqOffline := httptest.NewRequest(http.MethodGet, "/api/time/drift", nil)
	wOffline := httptest.NewRecorder()
	mux.ServeHTTP(wOffline, reqOffline)

	if wOffline.Code != http.StatusOK {
		t.Fatalf("expected 200 OK graceful response on offline, got %d", wOffline.Code)
	}
	var respOffline TimeDriftResponse
	if err := json.Unmarshal(wOffline.Body.Bytes(), &respOffline); err != nil {
		t.Fatalf("unmarshal offline resp failed: %v", err)
	}
	if respOffline.IsDrifted {
		t.Errorf("expected is_drifted false when offline")
	}
	if respOffline.Error == "" {
		t.Errorf("expected non-empty error message when offline")
	}
}
