package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"twofa-vault/pkg/webdav"
)

// WebDAVConfig holds persistent settings for WebDAV cloud backup & sync
type WebDAVConfig struct {
	URL        string    `json:"url"`
	Username   string    `json:"username"`
	Password   string    `json:"password,omitempty"`
	AutoSync   bool      `json:"auto_sync"`
	RemoteDir  string    `json:"remote_dir"`
	LastSync   time.Time `json:"last_sync,omitempty"`
	LastStatus string    `json:"last_status,omitempty"`
	LastError  string    `json:"last_error,omitempty"`
}

// WebDAVState manages runtime synchronization and configuration for WebDAV
type WebDAVState struct {
	mu            sync.RWMutex
	config        WebDAVConfig
	debounceTimer *time.Timer
	isSyncing     bool
}

func (s *Server) initWebDAV() {
	if s.webdavState == nil {
		s.webdavState = &WebDAVState{}
	}
	s.loadWebDAVConfig()

	// Register vault save hook for auto-sync
	s.vault.SetOnSaveHook(func() {
		s.handleVaultSaveForWebDAV()
	})
}

func (s *Server) configFilePath() string {
	dir := filepath.Dir(s.vault.StoragePath())
	return filepath.Join(dir, "webdav.json")
}

func (s *Server) loadWebDAVConfig() {
	filePath := s.configFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		s.webdavState.config.RemoteDir = "/2fa-vault"
		return
	}

	var cfg WebDAVConfig
	if err := json.Unmarshal(data, &cfg); err == nil {
		if cfg.RemoteDir == "" {
			cfg.RemoteDir = "/2fa-vault"
		}
		s.webdavState.config = cfg
	}
}

func (s *Server) saveWebDAVConfigLocked() error {
	filePath := s.configFilePath()
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.webdavState.config, "", "  ")
	if err != nil {
		return err
	}

	tmp := filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, filePath)
}

func (s *Server) handleVaultSaveForWebDAV() {
	if s.webdavState == nil {
		return
	}
	s.webdavState.mu.Lock()
	defer s.webdavState.mu.Unlock()

	if !s.webdavState.config.AutoSync || strings.TrimSpace(s.webdavState.config.URL) == "" {
		return
	}

	if s.webdavState.debounceTimer != nil {
		s.webdavState.debounceTimer.Stop()
	}

	s.webdavState.debounceTimer = time.AfterFunc(1*time.Second, func() {
		s.executeWebDAVBackupAsync()
	})
}

func (s *Server) executeWebDAVBackupAsync() {
	s.webdavState.mu.Lock()
	if s.webdavState.isSyncing {
		s.webdavState.mu.Unlock()
		return
	}
	s.webdavState.isSyncing = true
	cfg := s.webdavState.config
	s.webdavState.mu.Unlock()

	defer func() {
		s.webdavState.mu.Lock()
		s.webdavState.isSyncing = false
		s.webdavState.mu.Unlock()
	}()

	data, err := s.vault.GetRawEncryptedData()
	if err != nil || len(data) == 0 {
		return
	}

	remoteDir := cfg.RemoteDir
	if remoteDir == "" {
		remoteDir = "/2fa-vault"
	}
	remotePath := path.Join(remoteDir, "vault.enc")

	err = webdav.UploadFile(cfg.URL, cfg.Username, cfg.Password, remotePath, data)

	s.webdavState.mu.Lock()
	defer s.webdavState.mu.Unlock()
	if err != nil {
		s.webdavState.config.LastStatus = "error"
		s.webdavState.config.LastError = err.Error()
	} else {
		s.webdavState.config.LastSync = time.Now()
		s.webdavState.config.LastStatus = "success"
		s.webdavState.config.LastError = ""
	}
	_ = s.saveWebDAVConfigLocked()
}

// WebDAV Config endpoint: GET /api/webdav/config, POST /api/webdav/config
func (s *Server) handleWebDAVConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.webdavState.mu.RLock()
		cfg := s.webdavState.config
		s.webdavState.mu.RUnlock()

		maskedPassword := ""
		if cfg.Password != "" {
			maskedPassword = "••••••••"
		}

		jsonResponse(w, http.StatusOK, map[string]any{
			"url":          cfg.URL,
			"username":     cfg.Username,
			"password":     maskedPassword,
			"has_password": cfg.Password != "",
			"auto_sync":    cfg.AutoSync,
			"remote_dir":   cfg.RemoteDir,
			"last_sync":    cfg.LastSync,
			"last_status":  cfg.LastStatus,
			"last_error":   cfg.LastError,
		})

	case http.MethodPost:
		var req struct {
			URL       string `json:"url"`
			Username  string `json:"username"`
			Password  string `json:"password"`
			AutoSync  bool   `json:"auto_sync"`
			RemoteDir string `json:"remote_dir"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errorResponse(w, http.StatusBadRequest, "invalid json payload: "+err.Error())
			return
		}

		s.webdavState.mu.Lock()
		defer s.webdavState.mu.Unlock()

		s.webdavState.config.URL = strings.TrimSpace(req.URL)
		s.webdavState.config.Username = strings.TrimSpace(req.Username)

		// Retain existing password if omitted or masked
		if req.Password != "" && req.Password != "••••••••" && req.Password != "********" {
			s.webdavState.config.Password = req.Password
		}

		s.webdavState.config.AutoSync = req.AutoSync
		remoteDir := strings.TrimSpace(req.RemoteDir)
		if remoteDir == "" {
			remoteDir = "/2fa-vault"
		}
		s.webdavState.config.RemoteDir = remoteDir

		if err := s.saveWebDAVConfigLocked(); err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to save config: "+err.Error())
			return
		}

		maskedPassword := ""
		if s.webdavState.config.Password != "" {
			maskedPassword = "••••••••"
		}

		jsonResponse(w, http.StatusOK, map[string]any{
			"success":      true,
			"message":      "WebDAV 配置已成功保存",
			"url":          s.webdavState.config.URL,
			"username":     s.webdavState.config.Username,
			"password":     maskedPassword,
			"has_password": s.webdavState.config.Password != "",
			"auto_sync":    s.webdavState.config.AutoSync,
			"remote_dir":   s.webdavState.config.RemoteDir,
			"last_sync":    s.webdavState.config.LastSync,
			"last_status":  s.webdavState.config.LastStatus,
			"last_error":   s.webdavState.config.LastError,
		})

	default:
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// WebDAV Test endpoint: POST /api/webdav/test
func (s *Server) handleWebDAVTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		URL      string `json:"url"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	s.webdavState.mu.RLock()
	cfg := s.webdavState.config
	s.webdavState.mu.RUnlock()

	testURL := strings.TrimSpace(req.URL)
	if testURL == "" {
		testURL = cfg.URL
	}
	testUser := strings.TrimSpace(req.Username)
	if testUser == "" {
		testUser = cfg.Username
	}
	testPass := req.Password
	if testPass == "" || testPass == "••••••••" || testPass == "********" {
		testPass = cfg.Password
	}

	if testURL == "" {
		errorResponse(w, http.StatusBadRequest, "请输入 WebDAV 服务器地址")
		return
	}

	if err := webdav.TestConnection(testURL, testUser, testPass); err != nil {
		errorResponse(w, http.StatusBadRequest, "WebDAV 连接失败: "+err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "WebDAV 服务器连接成功，认证通过！",
	})
}

// WebDAV Backup endpoint: POST /api/webdav/backup
func (s *Server) handleWebDAVBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	data, err := s.vault.GetRawEncryptedData()
	if err != nil || len(data) == 0 {
		errorResponse(w, http.StatusBadRequest, "保险库尚未初始化或暂无加密数据")
		return
	}

	s.webdavState.mu.RLock()
	cfg := s.webdavState.config
	s.webdavState.mu.RUnlock()

	if strings.TrimSpace(cfg.URL) == "" {
		errorResponse(w, http.StatusBadRequest, "请先配置 WebDAV 服务器地址")
		return
	}

	remoteDir := cfg.RemoteDir
	if remoteDir == "" {
		remoteDir = "/2fa-vault"
	}
	remotePath := path.Join(remoteDir, "vault.enc")

	err = webdav.UploadFile(cfg.URL, cfg.Username, cfg.Password, remotePath, data)

	s.webdavState.mu.Lock()
	defer s.webdavState.mu.Unlock()
	if err != nil {
		s.webdavState.config.LastStatus = "error"
		s.webdavState.config.LastError = err.Error()
		_ = s.saveWebDAVConfigLocked()
		errorResponse(w, http.StatusInternalServerError, "WebDAV 备份上传失败: "+err.Error())
		return
	}

	s.webdavState.config.LastSync = time.Now()
	s.webdavState.config.LastStatus = "success"
	s.webdavState.config.LastError = ""
	_ = s.saveWebDAVConfigLocked()

	jsonResponse(w, http.StatusOK, map[string]any{
		"success":   true,
		"message":   "已成功将加密保险库备份至 WebDAV 云端！",
		"last_sync": s.webdavState.config.LastSync,
	})
}

// WebDAV Restore endpoint: POST /api/webdav/restore
func (s *Server) handleWebDAVRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	s.webdavState.mu.RLock()
	cfg := s.webdavState.config
	s.webdavState.mu.RUnlock()

	if strings.TrimSpace(cfg.URL) == "" {
		errorResponse(w, http.StatusBadRequest, "请先配置 WebDAV 服务器地址")
		return
	}

	remoteDir := cfg.RemoteDir
	if remoteDir == "" {
		remoteDir = "/2fa-vault"
	}
	remotePath := path.Join(remoteDir, "vault.enc")

	data, err := webdav.DownloadFile(cfg.URL, cfg.Username, cfg.Password, remotePath)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "从 WebDAV 下载云端备份失败: "+err.Error())
		return
	}
	if len(data) == 0 {
		errorResponse(w, http.StatusBadRequest, "云端备份文件内容为空")
		return
	}

	if err := s.vault.RestoreEncryptedVault(data); err != nil {
		errorResponse(w, http.StatusInternalServerError, "还原保险库失败: "+err.Error())
		return
	}

	s.webdavState.mu.Lock()
	s.webdavState.config.LastSync = time.Now()
	s.webdavState.config.LastStatus = "success"
	s.webdavState.config.LastError = ""
	_ = s.saveWebDAVConfigLocked()
	s.webdavState.mu.Unlock()

	jsonResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "已成功从 WebDAV 云端还原保险库！",
		"locked":  s.vault.IsLocked(),
	})
}
