package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/jchv/go-webview2"
	"twofa-vault/internal/api"
	"twofa-vault/internal/web"
	"twofa-vault/pkg/vault"
)

const version = "2.0.0"

func main() {
	// Lock OS thread for Win32 GUI message pump
	runtime.LockOSThread()

	portFlag := flag.Int("port", 2233, "HTTP server listening port")
	hostFlag := flag.String("host", "0.0.0.0", "HTTP server listening host (0.0.0.0 allows iOS / LAN devices to connect)")
	dataFlag := flag.String("data", "data/vault.enc", "Path to encrypted vault data file")
	headlessFlag := flag.Bool("headless", false, "Run in headless server/daemon mode (no desktop window)")
	versionFlag := flag.Bool("version", false, "Show version and exit")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("2FA Vault v%s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return
	}

	// Ensure data directory exists
	dataDir := filepath.Dir(*dataFlag)
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create data directory %s: %v\n", dataDir, err)
		os.Exit(1)
	}

	// Initialize Vault and API server
	v := vault.New(*dataFlag)
	apiServer := api.NewServer(v)

	mux := http.NewServeMux()

	// Register REST API routes
	apiServer.RegisterRoutes(mux)

	// Register embedded Web frontend for all other paths
	webHandler := web.Handler()
	mux.Handle("/", webHandler)

	addr := fmt.Sprintf("%s:%d", *hostFlag, *portFlag)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server failed to start: %v\n", err)
			os.Exit(1)
		}
	}()

	localURL := fmt.Sprintf("http://127.0.0.1:%d", *portFlag)

	// Native Windows Desktop App Window via Embedded WebView2
	if !*headlessFlag && runtime.GOOS == "windows" {
		profileDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "2FA-Vault", "webview2_data")
		_ = os.MkdirAll(profileDir, 0700)

		w := webview2.NewWithOptions(webview2.WebViewOptions{
			Debug:    false,
			DataPath: profileDir,
			WindowOptions: webview2.WindowOptions{
				Title:  "2FA Vault",
				Width:  1120,
				Height: 780,
				Center: true,
			},
		})

		if w != nil {
			defer w.Destroy()
			w.Navigate(localURL)

			// Run blocks until the user closes the native desktop window
			w.Run()

			// Desktop window closed: gracefully shutdown
			v.Lock()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx)
			return
		}
	}

	// Fallback to waiting for interrupt signal if headless or on non-Windows
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	v.Lock()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func getLocalIPs() []string {
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
