package webdav

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestURLHelpers(t *testing.T) {
	tests := []struct {
		base     string
		rel      string
		expected string
	}{
		{
			base:     "https://dav.jianguoyun.com/dav/",
			rel:      "vault.enc",
			expected: "https://dav.jianguoyun.com/dav/vault.enc",
		},
		{
			base:     "dav.jianguoyun.com/dav",
			rel:      "/2fa-vault/vault.enc",
			expected: "https://dav.jianguoyun.com/dav/2fa-vault/vault.enc",
		},
		{
			base:     "http://localhost:8080/remote.php/webdav",
			rel:      "sub/dir/test.txt",
			expected: "http://localhost:8080/remote.php/webdav/sub/dir/test.txt",
		},
		{
			base:     "https://nextcloud.org",
			rel:      "",
			expected: "https://nextcloud.org/",
		},
	}

	for _, tc := range tests {
		got, err := joinURL(tc.base, tc.rel)
		if err != nil {
			t.Fatalf("unexpected error for base %q, rel %q: %v", tc.base, tc.rel, err)
		}
		if got != tc.expected {
			t.Errorf("joinURL(%q, %q) = %q; want %q", tc.base, tc.rel, got, tc.expected)
		}
	}
}

func TestWebDAVClient(t *testing.T) {
	expectedUser := "testuser"
	expectedPass := "testsecret123"

	var mu sync.Mutex
	files := make(map[string][]byte)
	dirs := make(map[string]bool)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != expectedUser || pass != expectedPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="webdav"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		mu.Lock()
		defer mu.Unlock()

		cleanPath := strings.TrimPrefix(r.URL.Path, "/dav")
		if cleanPath == "" {
			cleanPath = "/"
		}

		switch r.Method {
		case "PROPFIND":
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?><D:multistatus xmlns:D="DAV:"></D:multistatus>`))
		case "MKCOL":
			dirs[cleanPath] = true
			w.WriteHeader(http.StatusCreated)
		case http.MethodPut:
			data, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			files[cleanPath] = data
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			data, exists := files[cleanPath]
			if !exists {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		case http.MethodOptions:
			w.Header().Set("DAV", "1, 2")
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	webdavURL := server.URL + "/dav"

	// 1. Test TestConnection success
	t.Run("TestConnection success", func(t *testing.T) {
		err := TestConnection(webdavURL, expectedUser, expectedPass)
		if err != nil {
			t.Fatalf("expected TestConnection to succeed, got %v", err)
		}
	})

	// 2. Test TestConnection unauthorized
	t.Run("TestConnection unauthorized", func(t *testing.T) {
		err := TestConnection(webdavURL, "wronguser", "wrongpass")
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})

	// 3. Test UploadFile with subdirectories
	testPayload := []byte("encrypted vault data blob 123456")
	remotePath := "/backups/vault/vault.enc"

	t.Run("UploadFile success", func(t *testing.T) {
		err := UploadFile(webdavURL, expectedUser, expectedPass, remotePath, testPayload)
		if err != nil {
			t.Fatalf("expected UploadFile to succeed, got %v", err)
		}

		mu.Lock()
		saved, exists := files["/backups/vault/vault.enc"]
		dir1 := dirs["/backups"]
		dir2 := dirs["/backups/vault"]
		mu.Unlock()

		if !exists {
			t.Fatalf("file was not stored on mock WebDAV server")
		}
		if string(saved) != string(testPayload) {
			t.Fatalf("expected saved content %s, got %s", testPayload, saved)
		}
		if !dir1 || !dir2 {
			t.Fatalf("expected MKCOL parent directories to be created, dir1=%v, dir2=%v", dir1, dir2)
		}
	})

	// 4. Test DownloadFile success
	t.Run("DownloadFile success", func(t *testing.T) {
		downloaded, err := DownloadFile(webdavURL, expectedUser, expectedPass, remotePath)
		if err != nil {
			t.Fatalf("expected DownloadFile to succeed, got %v", err)
		}
		if string(downloaded) != string(testPayload) {
			t.Fatalf("expected downloaded %s, got %s", testPayload, downloaded)
		}
	})

	// 5. Test DownloadFile not found
	t.Run("DownloadFile not found", func(t *testing.T) {
		_, err := DownloadFile(webdavURL, expectedUser, expectedPass, "/nonexistent.enc")
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	// 6. Test Client methods with Context
	t.Run("Client methods", func(t *testing.T) {
		client := NewClient(webdavURL, expectedUser, expectedPass)
		if err := client.TestConnection(context.Background()); err != nil {
			t.Fatalf("client.TestConnection failed: %v", err)
		}

		newData := []byte("new-data-v2")
		if err := client.UploadFile(context.Background(), "test.bin", newData); err != nil {
			t.Fatalf("client.UploadFile failed: %v", err)
		}

		got, err := client.DownloadFile(context.Background(), "test.bin")
		if err != nil {
			t.Fatalf("client.DownloadFile failed: %v", err)
		}
		if string(got) != string(newData) {
			t.Fatalf("downloaded data mismatch: got %s, want %s", got, newData)
		}
	})
}
