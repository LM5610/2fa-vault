package api

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

type pairToken struct {
	token     string
	createdAt time.Time
}

var (
	pairMu     sync.RWMutex
	pairTokens = make(map[string]pairToken)
)

func generateToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func isValidPairToken(token string) bool {
	pairMu.Lock()
	defer pairMu.Unlock()

	// Clean expired tokens (> 5 minutes)
	now := time.Now()
	for k, v := range pairTokens {
		if now.Sub(v.createdAt) > 5*time.Minute {
			delete(pairTokens, k)
		}
	}

	p, ok := pairTokens[token]
	if !ok {
		return false
	}
	return now.Sub(p.createdAt) <= 5*time.Minute
}

// LAN Sync Info endpoint: GET /api/sync/info
func (s *Server) handleSyncInfo(w http.ResponseWriter, r *http.Request) {
	lanIPs := getLocalLANIPs()
	jsonResponse(w, http.StatusOK, map[string]any{
		"lan_ips": lanIPs,
	})
}

// Generate Pairing Token: POST /api/sync/pair/init
func (s *Server) handleSyncPairInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := generateToken()
	pairMu.Lock()
	pairTokens[token] = pairToken{
		token:     token,
		createdAt: time.Now(),
	}
	pairMu.Unlock()

	lanIPs := getLocalLANIPs()
	host := r.Host
	if len(lanIPs) > 0 {
		// Use detected LAN IP if available
		_, port, err := net.SplitHostPort(r.Host)
		if err == nil {
			host = net.JoinHostPort(lanIPs[0], port)
		}
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	pairingURL := scheme + "://" + host + "/?sync_pair=" + token

	jsonResponse(w, http.StatusOK, map[string]any{
		"token":       token,
		"expires_in":  300,
		"pairing_url": pairingURL,
		"lan_ips":     lanIPs,
	})
}

// Download Encrypted Vault via Pairing Token: GET /api/sync/pair/vault?token=...
func (s *Server) handleSyncPairDownload(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if !isValidPairToken(token) {
		errorResponse(w, http.StatusUnauthorized, "invalid or expired pairing token")
		return
	}

	data, err := s.vault.GetRawEncryptedData()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "failed to read vault: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\"vault.enc\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// Upload Encrypted Vault via Pairing Token: POST /api/sync/pair/vault?token=...
func (s *Server) handleSyncPairUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := r.URL.Query().Get("token")
	if !isValidPairToken(token) {
		errorResponse(w, http.StatusUnauthorized, "invalid or expired pairing token")
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil || len(data) == 0 {
		errorResponse(w, http.StatusBadRequest, "empty or invalid vault payload")
		return
	}

	if err := s.vault.ImportRawEncryptedData(data); err != nil {
		errorResponse(w, http.StatusInternalServerError, "failed to write vault: "+err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "vault synchronized successfully",
	})
}

func getLocalLANIPs() []string {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips
}
