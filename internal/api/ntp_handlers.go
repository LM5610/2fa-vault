package api

import (
	"net/http"
	"time"

	"twofa-vault/pkg/totp"
)

// TimeDriftResponse represents the JSON payload returned by GET /api/time/drift
type TimeDriftResponse struct {
	SystemTime   string  `json:"system_time"`
	NetworkTime  string  `json:"network_time"`
	DriftSeconds float64 `json:"drift_seconds"`
	IsDrifted    bool    `json:"is_drifted"`
	Source       string  `json:"source,omitempty"`
	Error        string  `json:"error,omitempty"`
}

// handleTimeDrift returns system and network time drift detection results
func (s *Server) handleTimeDrift(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var res *totp.TimeDriftResult
	var err error

	if s.ntpClient != nil {
		res, err = s.ntpClient.DetectTimeDrift()
	} else {
		res, err = totp.DetectTimeDrift()
	}

	if err != nil {
		now := time.Now().UTC()
		jsonResponse(w, http.StatusOK, TimeDriftResponse{
			SystemTime:   now.Format(time.RFC3339),
			NetworkTime:  now.Format(time.RFC3339),
			DriftSeconds: 0,
			IsDrifted:    false,
			Error:        err.Error(),
		})
		return
	}

	jsonResponse(w, http.StatusOK, TimeDriftResponse{
		SystemTime:   res.SystemTime.UTC().Format(time.RFC3339),
		NetworkTime:  res.NetworkTime.UTC().Format(time.RFC3339),
		DriftSeconds: res.DriftSeconds,
		IsDrifted:    res.IsDrifted,
		Source:       res.Source,
	})
}
