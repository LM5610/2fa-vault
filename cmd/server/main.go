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

const version = "2.0.0"

func main() {
	portFlag := flag.Int("port", 2233, "HTTP server listening port")
	hostFlag := flag.String("host", "0.0.0.0", "HTTP server listening host (0.0.0.0 allows iOS / LAN devices to connect)")
	dataFlag := flag.String("data", "data/vault.enc", "Path to encrypted vault data file")
	noWindowFlag := flag.Bool("no-window", false, "Do not launch standalone desktop window")
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

	// Channel for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Automatically launch standalone native desktop window on Windows unless disabled
	if !*noWindowFlag && !*headlessFlag {
		localURL := fmt.Sprintf("http://127.0.0.1:%d", *portFlag)
		go func() {
			time.Sleep(200 * time.Millisecond)
			launchDesktopApp(localURL, func() {
				// Standalone desktop window was closed by the user
				select {
				case quit <- os.Interrupt:
				default:
				}
			})
		}()
	}

	// Wait for window close or interrupt signal
	<-quit

	v.Lock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func printBanner(port int, dataPath string, lanIPs []string) {
	fmt.Println("================================================================")
	fmt.Printf("  2FA Vault v%s - Standalone Native Desktop Application\n", version)
	fmt.Println("  Modular Go Backend · AES-256-GCM + Argon2id · Windows Desktop & iOS")
	fmt.Println("================================================================")
	fmt.Printf("  • Local Desktop URL      : http://127.0.0.1:%d\n", port)
	for _, ip := range lanIPs {
		fmt.Printf("  • iOS / Mobile Access    : http://%s:%d\n", ip, port)
	}
	fmt.Printf("  • Encrypted Vault File   : %s\n", dataPath)
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

func launchDesktopApp(appURL string, onWindowClose func()) {
	if runtime.GOOS == "windows" {
		edgePaths := []string{
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
			`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		}

		if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
			edgePaths = append(edgePaths,
				filepath.Join(localApp, `Microsoft\Edge\Application\msedge.exe`),
				filepath.Join(localApp, `Google\Chrome\Application\chrome.exe`),
			)
		}

		var targetExe string
		for _, p := range edgePaths {
			if _, err := os.Stat(p); err == nil {
				targetExe = p
				break
			}
		}

		if targetExe == "" {
			if path, err := exec.LookPath("msedge.exe"); err == nil {
				targetExe = path
			} else if path, err := exec.LookPath("chrome.exe"); err == nil {
				targetExe = path
			}
		}

		if targetExe != "" {
			profileDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "2FA-Vault", "app_profile")
			_ = os.MkdirAll(profileDir, 0700)

			args := []string{
				fmt.Sprintf("--app=%s", appURL),
				"--window-size=1120,780",
				fmt.Sprintf("--user-data-dir=%s", profileDir),
				"--disable-features=Translate,OptimizationHints",
				"--disable-extensions",
				"--no-first-run",
				"--no-default-browser-check",
			}

			cmd := exec.Command(targetExe, args...)
			if err := cmd.Start(); err == nil {
				if onWindowClose != nil {
					go func() {
						_ = cmd.Wait()
						onWindowClose()
					}()
				}
				return
			}
		}
	}

	// Fallback to standard browser launch if app window mode is unavailable
	openBrowser(appURL)
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
	_ = err
}
