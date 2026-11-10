package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"twofa-vault/pkg/totp"
)

// Real-time live OTP code preview endpoint: GET /api/otp/preview, POST /api/otp/preview
func (s *Server) handleOTPPreview(w http.ResponseWriter, r *http.Request) {
	var secret string
	var algo totp.Algorithm = totp.AlgorithmSHA1
	var digits = 6
	var period = 30
	var otpType totp.OTPType = totp.TypeTOTP
	var counter uint64 = 0

	if r.Method == http.MethodPost {
		var req struct {
			Secret    string         `json:"secret"`
			Digits    int            `json:"digits"`
			Period    int            `json:"period"`
			Algorithm totp.Algorithm `json:"algorithm"`
			Type      totp.OTPType   `json:"type"`
			Counter   uint64         `json:"counter"`
		}
		if err := jsonDecode(r, &req); err == nil {
			secret = req.Secret
			if req.Digits > 0 {
				digits = req.Digits
			}
			if req.Period > 0 {
				period = req.Period
			}
			if req.Algorithm != "" {
				algo = req.Algorithm
			}
			if req.Type != "" {
				otpType = req.Type
			}
			counter = req.Counter
		}
	} else {
		q := r.URL.Query()
		secret = q.Get("secret")
		if a := q.Get("algorithm"); a != "" {
			algo = totp.Algorithm(a)
		}
		if d := q.Get("digits"); d != "" {
			if v, err := strconv.Atoi(d); err == nil && v > 0 {
				digits = v
			}
		}
		if p := q.Get("period"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				period = v
			}
		}
		if t := q.Get("type"); t != "" {
			otpType = totp.OTPType(t)
		}
		if c := q.Get("counter"); c != "" {
			if v, err := strconv.ParseUint(c, 10, 64); err == nil {
				counter = v
			}
		}
	}

	secret = strings.TrimSpace(secret)
	if secret == "" {
		jsonResponse(w, http.StatusOK, map[string]any{"valid": false, "error": "secret is empty"})
		return
	}

	cleaned, err := totp.CleanSecret(secret)
	if err != nil {
		jsonResponse(w, http.StatusOK, map[string]any{"valid": false, "error": err.Error()})
		return
	}

	opts := totp.Options{
		Secret:    cleaned,
		Algorithm: algo,
		Digits:    digits,
		Period:    period,
		Type:      otpType,
		Counter:   counter,
	}

	now := time.Now()
	code, err := totp.GenerateCode(opts, now)
	if err != nil {
		jsonResponse(w, http.StatusOK, map[string]any{"valid": false, "error": err.Error()})
		return
	}

	formatted := code
	if len(code) == 6 {
		formatted = code[:3] + " " + code[3:]
	} else if len(code) == 8 {
		formatted = code[:4] + " " + code[4:]
	}

	rem := totp.RemainingSeconds(period, now)
	prog := totp.Progress(period, now)
	if otpType == totp.TypeHOTP {
		rem = 0
		prog = 0
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"valid":          true,
		"code":           code,
		"formatted_code": formatted,
		"remaining":      rem,
		"progress":       prog,
		"period":         period,
	})
}

// Helper function to decode JSON from request body
func jsonDecode(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// SSE live countdown endpoint: GET /api/events
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
