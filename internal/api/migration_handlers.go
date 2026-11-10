package api

import (
	"fmt"
	"io"
	"net/http"

	"twofa-vault/pkg/migration"
	"twofa-vault/pkg/vault"
)

// Import endpoint: POST /api/import
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

// Export endpoint: GET /api/export?format=json|uri
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
