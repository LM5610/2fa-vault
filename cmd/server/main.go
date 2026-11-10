package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"twofa-vault/internal/api"
	"twofa-vault/internal/web"
	"twofa-vault/pkg/vault"
)

const version = "1.0.0"

func main() {
	portFlag := flag.Int("port", 2233, "HTTP server listening port")
	hostFlag := flag.String("host", "0.0.0.0", "HTTP server listening host (0.0.0.0 allows iOS / LAN devices to connect)")
	dataFlag := flag.String("data", "data/vault.enc", "Path to encrypted vault data file")
	noBrowserFlag := flag.Bool("no-browser", false, "Do not automatically open default browser on startup")
	headlessFlag := flag.Bool("headless", false, "Run in headless server/daemon mode")
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

	// Discover LAN IP addresses for iOS access
	lanIPs := getLocalIPs()

	printBanner(*portFlag, *dataFlag, lanIPs)

	// Start server in background
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server failed to start: %v\n", err)
			os.Exit(1)
		}
	}()

	// Automatically open browser on Windows unless disabled
	if !*noBrowserFlag && !*headlessFlag {
		localURL := fmt.Sprintf("http://127.0.0.1:%d", *portFlag)
		go func() {
			time.Sleep(300 * time.Millisecond)
			openBrowser(localURL)
		}()
	}

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n[2FA Vault] Shutting down gracefully, locking vault...")
	v.Lock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced shutdown: %v\n", err)
	}

	fmt.Println("[2FA Vault] Goodbye!")
}

func printBanner(port int, dataPath string, lanIPs []string) {
	fmt.Println("================================================================")
	fmt.Printf("  2FA Vault v%s - Secure Cross-Platform Authenticator\n", version)
	fmt.Println("  Modular Go Backend · AES-256-GCM + Argon2id · Windows & iOS PWA")
	fmt.Println("================================================================")
	fmt.Printf("  • Local Access (Windows) : http://127.0.0.1:%d\n", port)
	for _, ip := range lanIPs {
		fmt.Printf("  • iOS / Mobile Access    : http://%s:%d\n", ip, port)
	}
	fmt.Printf("  • Encrypted Vault File   : %s\n", dataPath)
	fmt.Println("----------------------------------------------------------------")
	fmt.Println("  Tips for iOS: Open the mobile URL in Safari, then tap:")
	fmt.Println("  Share (分享) -> Add to Home Screen (添加到主屏幕) for full PWA!")
	fmt.Println("  Press Ctrl+C to safely lock vault and quit.")
	fmt.Println("================================================================")
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

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		// Silent ignore if browser launcher fails
	}
}
