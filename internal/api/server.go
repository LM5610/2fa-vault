package api

import (
	"encoding/json"
	"net/http"
	"time"

	"twofa-vault/pkg/totp"
	"twofa-vault/pkg/vault"
)

// Server handles RESTful API requests for 2FA vault
type Server struct {
	vault      *vault.Vault
	httpClient *http.Client
	ntpClient  *totp.NTPClient
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

// RegisterRoutes registers all REST and SSE endpoints on mux with clean aliases
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// Vault lifecycle & security
	mux.HandleFunc("/api/vault/status", s.handleVaultStatus)
	mux.HandleFunc("/api/status", s.handleVaultStatus)
	mux.HandleFunc("/api/vault/init", s.handleVaultInit)
	mux.HandleFunc("/api/init", s.handleVaultInit)
	mux.HandleFunc("/api/vault/unlock", s.handleVaultUnlock)
	mux.HandleFunc("/api/unlock", s.handleVaultUnlock)
	mux.HandleFunc("/api/vault/lock", s.handleVaultLock)
	mux.HandleFunc("/api/lock", s.handleVaultLock)
	mux.HandleFunc("/api/vault/password", s.handleVaultPassword)
	mux.HandleFunc("/api/settings/password", s.handleVaultPassword)
	mux.HandleFunc("/api/vault/autolock", s.handleVaultAutoLock)
	mux.HandleFunc("/api/settings/autolock", s.handleVaultAutoLock)
	mux.HandleFunc("/api/vault/snapshots", s.handleListSnapshots)
	mux.HandleFunc("/api/vault/snapshots/rollback", s.handleRollbackSnapshot)

	// Account management
	mux.HandleFunc("/api/accounts/reorder", s.handleReorderAccounts)
	mux.HandleFunc("/api/accounts", s.handleAccounts)
	mux.HandleFunc("/api/accounts/", s.handleAccountItem)
	mux.HandleFunc("/api/tags", s.handleTags)

	// QR code & Image processing & Brand Logos
	mux.HandleFunc("/api/qr", s.handleQRCode)
	mux.HandleFunc("/api/qr/decode-url", s.handleQRDecodeURL)
	mux.HandleFunc("/api/qr/decode-file", s.handleQRDecodeFile)
	mux.HandleFunc("/api/proxy-image", s.handleProxyImage)
	mux.HandleFunc("/api/logo", s.handleLogo)

	// Real-time OTP & streaming
	mux.HandleFunc("/api/otp/preview", s.handleOTPPreview)
	mux.HandleFunc("/api/events", s.handleSSE)

	// Time drift & clock synchronization
	mux.HandleFunc("/api/time/drift", s.handleTimeDrift)

	// Migration (Import & Export)
	mux.HandleFunc("/api/import", s.handleImport)
	mux.HandleFunc("/api/export", s.handleExport)

	// P2P LAN Direct Sync
	mux.HandleFunc("/api/sync/info", s.handleSyncInfo)
	mux.HandleFunc("/api/sync/pair/init", s.handleSyncPairInit)
	mux.HandleFunc("/api/sync/pair/vault", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.handleSyncPairDownload(w, r)
		} else if r.Method == http.MethodPost {
			s.handleSyncPairUpload(w, r)
		} else {
			errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
}

func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func errorResponse(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, map[string]string{"error": message})
}
