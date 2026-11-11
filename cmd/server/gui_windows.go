//go:build windows

package main

import (
	"os"
	"path/filepath"

	"github.com/jchv/go-webview2"
)

func runDesktopGUI(url string, title string, width, height int) (func(), error) {
	profileDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "2FA-Vault", "webview2_data")
	_ = os.MkdirAll(profileDir, 0700)

	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:    false,
		DataPath: profileDir,
		WindowOptions: webview2.WindowOptions{
			Title:  title,
			Width:  uint(width),
			Height: uint(height),
			Center: true,
		},
	})
	if w == nil {
		return nil, nil
	}

	w.Navigate(url)
	return func() {
		defer w.Destroy()
		w.Run()
	}, nil
}
