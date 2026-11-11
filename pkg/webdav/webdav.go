package webdav

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

var (
	ErrUnauthorized = errors.New("webdav: unauthorized (invalid username or password)")
	ErrForbidden    = errors.New("webdav: forbidden (access denied)")
	ErrNotFound     = errors.New("webdav: remote file or collection not found")
	ErrInvalidURL   = errors.New("webdav: invalid server URL")
)

// Client is a lightweight, zero-dependency WebDAV client using standard net/http.
type Client struct {
	BaseURL    string
	Username   string
	Password   string
	HTTPClient *http.Client
}

// NewClient creates a new WebDAV client instance.
func NewClient(baseURL, username, password string) *Client {
	return &Client{
		BaseURL:  normalizeURL(baseURL),
		Username: username,
		Password: password,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// normalizeURL ensures the URL has an HTTP/HTTPS scheme and clean formatting.
func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	return raw
}

// joinURL joins baseURL with a relative WebDAV path, preserving base path prefixes.
func joinURL(baseURL, relPath string) (string, error) {
	baseURL = normalizeURL(baseURL)
	if baseURL == "" {
		return "", ErrInvalidURL
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	basePath := strings.TrimRight(u.Path, "/")
	cleanRel := strings.Trim(relPath, "/")

	if cleanRel != "" {
		u.Path = basePath + "/" + cleanRel
	} else {
		u.Path = basePath
		if u.Path == "" {
			u.Path = "/"
		}
	}
	return u.String(), nil
}

// newRequest constructs an HTTP request with basic auth headers.
func (c *Client) newRequest(ctx context.Context, method, targetURL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
	if err != nil {
		return nil, err
	}
	if c.Username != "" || c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}
	return req, nil
}

// TestConnection verifies connectivity and authentication with the WebDAV server.
func (c *Client) TestConnection(ctx context.Context) error {
	targetURL, err := joinURL(c.BaseURL, "")
	if err != nil {
		return err
	}

	// First try PROPFIND Depth: 0 (standard WebDAV discovery)
	req, err := c.newRequest(ctx, "PROPFIND", targetURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Depth", "0")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("webdav connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusMultiStatus || (resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return nil
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	if resp.StatusCode == http.StatusForbidden {
		return ErrForbidden
	}

	// If PROPFIND is not allowed (e.g. 405), fallback to OPTIONS or GET/HEAD
	if resp.StatusCode == http.StatusMethodNotAllowed {
		optReq, err := c.newRequest(ctx, http.MethodOptions, targetURL, nil)
		if err == nil {
			optResp, err := c.HTTPClient.Do(optReq)
			if err == nil {
				defer optResp.Body.Close()
				if optResp.StatusCode >= 200 && optResp.StatusCode < 300 {
					return nil
				}
			}
		}
	}

	return fmt.Errorf("webdav test failed with HTTP status %d (%s)", resp.StatusCode, resp.Status)
}

// EnsureDirectories ensures all parent directory collections exist using MKCOL.
func (c *Client) EnsureDirectories(ctx context.Context, remoteFilePath string) error {
	dir := path.Dir(strings.Trim(remoteFilePath, "/"))
	if dir == "." || dir == "/" || dir == "" {
		return nil
	}

	parts := strings.Split(dir, "/")
	currentPath := ""
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = currentPath + "/" + part
		}

		dirURL, err := joinURL(c.BaseURL, currentPath)
		if err != nil {
			return err
		}

		req, err := c.newRequest(ctx, "MKCOL", dirURL, nil)
		if err != nil {
			return err
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to create directory '%s': %w", currentPath, err)
		}
		resp.Body.Close()

		// 201 Created: successfully created
		// 405 Method Not Allowed: collection already exists (RFC 4918)
		// 200/204/301: accepted
		if resp.StatusCode == http.StatusCreated ||
			resp.StatusCode == http.StatusMethodNotAllowed ||
			resp.StatusCode == http.StatusOK ||
			resp.StatusCode == http.StatusNoContent {
			continue
		}

		if resp.StatusCode == http.StatusUnauthorized {
			return ErrUnauthorized
		}
		if resp.StatusCode == http.StatusForbidden {
			return ErrForbidden
		}

		// Some servers return 409 Conflict if parent doesn't exist, but we iterate top-down
		return fmt.Errorf("mkdir '%s' failed with status %d", currentPath, resp.StatusCode)
	}

	return nil
}

// UploadFile uploads file contents to remotePath on the WebDAV server.
func (c *Client) UploadFile(ctx context.Context, remotePath string, data []byte) error {
	if err := c.EnsureDirectories(ctx, remotePath); err != nil {
		return err
	}

	targetURL, err := joinURL(c.BaseURL, remotePath)
	if err != nil {
		return err
	}

	req, err := c.newRequest(ctx, http.MethodPut, targetURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("webdav upload failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	if resp.StatusCode == http.StatusForbidden {
		return ErrForbidden
	}

	bodySample, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("webdav upload failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodySample)))
}

// DownloadFile downloads file contents from remotePath on the WebDAV server.
func (c *Client) DownloadFile(ctx context.Context, remotePath string) ([]byte, error) {
	targetURL, err := joinURL(c.BaseURL, remotePath)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webdav download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, ErrForbidden
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodySample, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("webdav download failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodySample)))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// Package-level convenience functions:

// TestConnection tests connectivity to a WebDAV server using url, user, and password.
func TestConnection(url, user, pass string) error {
	return NewClient(url, user, pass).TestConnection(context.Background())
}

// UploadFile uploads data to remotePath on WebDAV server using url, user, and password.
func UploadFile(url, user, pass, remotePath string, data []byte) error {
	return NewClient(url, user, pass).UploadFile(context.Background(), remotePath, data)
}

// DownloadFile downloads file contents from remotePath on WebDAV server.
func DownloadFile(url, user, pass, remotePath string) ([]byte, error) {
	return NewClient(url, user, pass).DownloadFile(context.Background(), remotePath)
}
