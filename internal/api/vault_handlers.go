package api

import (
	"encoding/json"
	"net/http"
)

// Vault Status endpoint: GET /api/vault/status, /api/status
func (s *Server) handleVaultStatus(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]any{
		"initialized":       s.vault.IsInitialized(),
		"locked":            s.vault.IsLocked(),
		"auto_lock_minutes": s.vault.AutoLockMinutes(),
	})
}

// Vault Init endpoint: POST /api/vault/init, /api/init
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

	if len(req.Password) < 6 {
		errorResponse(w, http.StatusBadRequest, "master password must be at least 6 characters")
		return
	}

	if err := s.vault.Init(req.Password); err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "vault initialized and unlocked successfully",
	})
}

// Vault Unlock endpoint: POST /api/vault/unlock, /api/unlock
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
		errorResponse(w, http.StatusUnauthorized, "invalid master password")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "vault unlocked successfully",
	})
}

// Vault Lock endpoint: POST /api/vault/lock, /api/lock
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

// Vault Change Password endpoint: POST /api/vault/password, /api/settings/password
func (s *Server) handleVaultPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.vault.IsLocked() {
		errorResponse(w, http.StatusUnauthorized, "vault is locked")
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

	if len(req.NewPassword) < 6 {
		errorResponse(w, http.StatusBadRequest, "new master password must be at least 6 characters")
		return
	}

	if err := s.vault.ChangePassword(req.OldPassword, req.NewPassword); err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "master password changed successfully",
	})
}

// Vault Auto-lock setting: POST /api/vault/autolock, /api/settings/autolock
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

	if req.Minutes < 0 || req.Minutes > 1440 {
		errorResponse(w, http.StatusBadRequest, "minutes must be between 0 and 1440")
		return
	}

	s.vault.SetAutoLockMinutes(req.Minutes)
	jsonResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"minutes": req.Minutes,
	})
}

// Vault Snapshots: GET /api/vault/snapshots
func (s *Server) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	snapshots, err := s.vault.ListSnapshots()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, snapshots)
}

// Vault Snapshot Rollback: POST /api/vault/snapshots/rollback
func (s *Server) handleRollbackSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if err := s.vault.RollbackSnapshot(req.Filename); err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "vault successfully rolled back to snapshot",
	})
}
