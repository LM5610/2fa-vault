package api

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type cachedLogo struct {
	data        []byte
	contentType string
	expiresAt   time.Time
}

var (
	logoCacheMu sync.RWMutex
	logoCache   = make(map[string]cachedLogo)
)

// Mainstream domain mappings for common issuers
var issuerDomainMap = map[string]string{
	"github":      "github.com",
	"google":      "google.com",
	"gmail":       "google.com",
	"microsoft":   "microsoft.com",
	"live":        "microsoft.com",
	"outlook":     "microsoft.com",
	"apple":       "apple.com",
	"icloud":      "apple.com",
	"steam":       "steampowered.com",
	"binance":     "binance.com",
	"aws":         "aws.amazon.com",
	"amazon":      "amazon.com",
	"cloudflare":  "cloudflare.com",
	"discord":     "discord.com",
	"telegram":    "telegram.org",
	"twitter":     "twitter.com",
	"x":           "x.com",
	"gitlab":      "gitlab.com",
	"bitwarden":   "bitwarden.com",
	"openai":      "openai.com",
	"chatgpt":     "openai.com",
	"reddit":      "reddit.com",
	"spotify":     "spotify.com",
	"netflix":     "netflix.com",
	"paypal":      "paypal.com",
	"stripe":      "stripe.com",
	"epic games":  "epicgames.com",
	"riot games":  "riotgames.com",
	"riot":        "riotgames.com",
	"proton":      "proton.me",
	"protonmail":  "proton.me",
	"dropbox":     "dropbox.com",
	"notion":      "notion.so",
	"slack":       "slack.com",
	"facebook":    "facebook.com",
	"meta":        "meta.com",
	"instagram":   "instagram.com",
	"okx":         "okx.com",
	"huobi":       "huobi.com",
	"coinbase":    "coinbase.com",
	"linode":      "linode.com",
	"digitalocean": "digitalocean.com",
	"vultr":       "vultr.com",
}

// Logo fetcher: GET /api/logo?issuer=...&domain=...&account=...
func (s *Server) handleLogo(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	issuer := strings.ToLower(strings.TrimSpace(q.Get("issuer")))
	domain := strings.ToLower(strings.TrimSpace(q.Get("domain")))
	account := strings.TrimSpace(q.Get("account"))

	cacheKey := issuer + ":" + domain + ":" + account
	logoCacheMu.RLock()
	if c, ok := logoCache[cacheKey]; ok && time.Now().Before(c.expiresAt) {
		logoCacheMu.RUnlock()
		w.Header().Set("Content-Type", c.contentType)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(c.data)
		return
	}
	logoCacheMu.RUnlock()

	// Try fetching GitHub user avatar if issuer is GitHub and account has no email '@'
	var fetchURL string
	if issuer == "github" && account != "" && !strings.Contains(account, "@") {
		fetchURL = fmt.Sprintf("https://github.com/%s.png?size=128", url.PathEscape(account))
	} else {
		// Determine domain
		targetDomain := domain
		if targetDomain == "" {
			if d, ok := issuerDomainMap[issuer]; ok {
				targetDomain = d
			} else if strings.Contains(issuer, ".") {
				targetDomain = issuer
			}
		}

		if targetDomain == "" {
			errorResponse(w, http.StatusNotFound, "unknown domain for logo")
			return
		}

		// Use high-resolution Google Favicon API
		fetchURL = fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=128", url.QueryEscape(targetDomain))
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, fetchURL, nil)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 2FA-Vault/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		errorResponse(w, http.StatusBadGateway, "failed to fetch brand logo")
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil || len(data) == 0 {
		errorResponse(w, http.StatusBadGateway, "empty logo response")
		return
	}

	cType := resp.Header.Get("Content-Type")
	if cType == "" {
		cType = "image/png"
	}

	// Cache for 24 hours
	logoCacheMu.Lock()
	logoCache[cacheKey] = cachedLogo{
		data:        data,
		contentType: cType,
		expiresAt:   time.Now().Add(24 * time.Hour),
	}
	logoCacheMu.Unlock()

	w.Header().Set("Content-Type", cType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
