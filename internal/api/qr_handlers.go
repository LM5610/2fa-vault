package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
	tuotoo "github.com/tuotoo/qrcode"

	"twofa-vault/pkg/totp"
)

// QR Code generator endpoint: GET /api/qr?data=...
func (s *Server) handleQRCode(w http.ResponseWriter, r *http.Request) {
	data := r.URL.Query().Get("data")
	if data == "" {
		errorResponse(w, http.StatusBadRequest, "missing data parameter")
		return
	}

	size := 256
	if s := r.URL.Query().Get("size"); s != "" {
		if val, err := strconv.Atoi(s); err == nil && val > 64 && val <= 1024 {
			size = val
		}
	}

	png, err := qrcode.Encode(data, qrcode.Medium, size)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("qr encode failed: %v", err))
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

// Decode QR from Image URL or Data URI: POST /api/qr/decode-url
func (s *Server) handleQRDecodeURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid json body")
		return
	}

	rawURL := strings.TrimSpace(req.URL)
	if rawURL == "" {
		errorResponse(w, http.StatusBadRequest, "url cannot be empty")
		return
	}

	var imgBytes []byte
	var err error

	// Handle Data URI (data:image/...;base64,...)
	if strings.HasPrefix(rawURL, "data:image/") {
		commaIdx := strings.Index(rawURL, ",")
		if commaIdx == -1 {
			errorResponse(w, http.StatusBadRequest, "invalid data uri format")
			return
		}
		imgBytes, err = base64.StdEncoding.DecodeString(rawURL[commaIdx+1:])
		if err != nil {
			errorResponse(w, http.StatusBadRequest, "invalid base64 in data uri")
			return
		}
	} else {
		// HTTP/HTTPS URL
		u, err := url.Parse(rawURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			errorResponse(w, http.StatusBadRequest, "url must be http or https")
			return
		}

		httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, rawURL, nil)
		if err != nil {
			errorResponse(w, http.StatusBadRequest, "invalid request")
			return
		}
		httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) 2FA-Vault/1.0")

		resp, err := s.httpClient.Do(httpReq)
		if err != nil {
			errorResponse(w, http.StatusBadRequest, fmt.Sprintf("fetch image failed: %v", err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errorResponse(w, http.StatusBadRequest, fmt.Sprintf("image server returned status %d", resp.StatusCode))
			return
		}

		imgBytes, err = io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
		if err != nil {
			errorResponse(w, http.StatusBadRequest, "read image response failed")
			return
		}
	}

	result, err := decodeQRBytes(imgBytes)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, fmt.Sprintf("qr decode failed: %v", err))
		return
	}

	jsonResponse(w, http.StatusOK, result)
}

// Decode QR from File upload: POST /api/qr/decode-file
func (s *Server) handleQRDecodeFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "missing 'file' form field")
		return
	}
	defer file.Close()

	imgBytes, err := io.ReadAll(io.LimitReader(file, 10*1024*1024))
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "read file failed")
		return
	}

	result, err := decodeQRBytes(imgBytes)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, fmt.Sprintf("qr decode failed: %v", err))
		return
	}

	jsonResponse(w, http.StatusOK, result)
}

// Proxy image to bypass browser CORS: GET /api/proxy-image?url=...
func (s *Server) handleProxyImage(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		errorResponse(w, http.StatusBadRequest, "missing url")
		return
	}

	u, err := url.Parse(targetURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		errorResponse(w, http.StatusBadRequest, "invalid url scheme")
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 2FA-Vault/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		errorResponse(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// decodeQRBytes parses and extracts QR matrix from image bytes
func decodeQRBytes(data []byte) (map[string]any, error) {
	matrix, err := tuotoo.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("no QR code found or image is too blurry")
	}

	content := strings.TrimSpace(matrix.Content)
	res := map[string]any{
		"text": content,
	}

	// If content is an otpauth URI, parse details
	if strings.HasPrefix(content, "otpauth://") {
		parsed, err := totp.ParseURI(content)
		if err == nil {
			res["parsed"] = map[string]any{
				"issuer":       parsed.Issuer,
				"account_name": parsed.Account,
				"secret":       parsed.Secret,
				"algorithm":    parsed.Algorithm,
				"digits":       parsed.Digits,
				"period":       parsed.Period,
				"type":         parsed.Type,
				"counter":      parsed.Counter,
			}
		}
	}

	return res, nil
}
