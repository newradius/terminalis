package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"Terminalis/internal/config"
	"Terminalis/internal/models"
	"Terminalis/internal/ssh"
	"Terminalis/internal/storage"
	"Terminalis/internal/terminal"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ActiveSession struct {
	session   terminal.TerminalSession
	sessionID string
	cancel    context.CancelFunc
	sshClient *ssh.Client      // non-nil only for SSH sessions
	sftp      *ssh.SftpSession // lazily created
}

type App struct {
	ctx            context.Context
	store          *storage.Store
	knownHosts     *ssh.KnownHosts
	config         config.AppConfig
	activeSessions map[string]*ActiveSession // tabID -> active session
	mu             sync.Mutex

	// Pending host key verification
	pendingHostKey     chan bool
	pendingHostKeyLock sync.Mutex
}

func NewApp() *App {
	return &App{
		activeSessions: make(map[string]*ActiveSession),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	store, err := storage.NewStore()
	if err != nil {
		runtime.LogError(ctx, "Failed to initialize store: "+err.Error())
		return
	}
	a.store = store

	kh, err := ssh.NewKnownHosts(store.DataDir())
	if err != nil {
		runtime.LogError(ctx, "Failed to initialize known hosts: "+err.Error())
		return
	}
	a.knownHosts = kh
	a.config = config.Load(store.DataDir())
}

func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, as := range a.activeSessions {
		if as.sftp != nil {
			as.sftp.Close()
		}
		as.cancel()
		as.session.Close()
	}
}

// ---- Session CRUD ----

func (a *App) GetSessionTree() []models.TreeNode {
	return a.store.GetTree()
}

func (a *App) GetSession(id string) *models.Session {
	return a.store.GetSession(id)
}

func (a *App) SaveSession(sess models.Session) error {
	if sess.ID == "" {
		sess.ID = uuid.New().String()
		sess.CreatedAt = time.Now().Unix()
	}
	sess.UpdatedAt = time.Now().Unix()
	if sess.Port == 0 {
		sess.Port = 22
	}
	return a.store.SaveSession(sess)
}

func (a *App) DeleteSession(id string) error {
	// Disconnect if active
	a.DisconnectTab(id)
	return a.store.DeleteSession(id)
}

// ---- Folder CRUD ----

func (a *App) SaveFolder(folder models.Folder) error {
	if folder.ID == "" {
		folder.ID = uuid.New().String()
		folder.Expanded = true
	}
	return a.store.SaveFolder(folder)
}

func (a *App) DeleteFolder(id string) error {
	return a.store.DeleteFolder(id)
}

func (a *App) ToggleFolderExpanded(id string) error {
	return a.store.ToggleFolderExpanded(id)
}

// ---- SSH Connection ----

type ConnectRequest struct {
	TabID     string `json:"tabId"`
	SessionID string `json:"sessionId"`
	Password  string `json:"password,omitempty"`
	Cols      int    `json:"cols"`
	Rows      int    `json:"rows"`
}

type QuickConnectRequest struct {
	TabID      string `json:"tabId"`
	ConnString string `json:"connString"`
	Password   string `json:"password,omitempty"`
	Cols       int    `json:"cols"`
	Rows       int    `json:"rows"`
}

func (a *App) ConnectSession(req ConnectRequest) error {
	sess := a.store.GetSession(req.SessionID)
	if sess == nil {
		return fmt.Errorf("session not found")
	}

	cfg := ssh.ConnectConfig{
		Host:           sess.Host,
		Port:           sess.Port,
		Username:       sess.Username,
		Password:       sess.Password,
		PrivateKeyPath: sess.PrivateKeyPath,
		Passphrase:     sess.Passphrase,
		UseKey:         sess.AuthMethod == models.AuthPrivateKey,
		Compression:    sess.Compression,
		KeepAlive:      sess.KeepAlive,
		Timeout:        a.config.ConnectTimeout,
		Cols:           req.Cols,
		Rows:           req.Rows,
	}

	return a.connectSSH(req.TabID, cfg)
}

func (a *App) QuickConnect(req QuickConnectRequest) error {
	cfg := ssh.ConnectConfig{
		Port:     22,
		Username: "root",
		Password: req.Password,
		Timeout:  a.config.ConnectTimeout,
		Cols:     req.Cols,
		Rows:     req.Rows,
	}

	// Parse user@host:port
	remaining := req.ConnString
	for i := 0; i < len(remaining); i++ {
		if remaining[i] == '@' {
			cfg.Username = remaining[:i]
			remaining = remaining[i+1:]
			break
		}
	}

	// Check for port
	for i := len(remaining) - 1; i >= 0; i-- {
		if remaining[i] == ':' {
			fmt.Sscanf(remaining[i+1:], "%d", &cfg.Port)
			remaining = remaining[:i]
			break
		}
	}
	cfg.Host = remaining

	return a.connectSSH(req.TabID, cfg)
}

func (a *App) connectSSH(tabID string, cfg ssh.ConnectConfig) error {
	hostKeyCallback := a.knownHosts.HostKeyCallback(func(result ssh.HostKeyResult) bool {
		// Emit event to frontend and wait for response
		a.pendingHostKeyLock.Lock()
		a.pendingHostKey = make(chan bool, 1)
		a.pendingHostKeyLock.Unlock()

		runtime.EventsEmit(a.ctx, "ssh:hostkey", map[string]interface{}{
			"tabId":       tabID,
			"host":        result.Host,
			"fingerprint": result.Fingerprint,
			"keyType":     result.KeyType,
			"isNew":       result.Status == ssh.HostKeyNew,
			"isMismatch":  result.Status == ssh.HostKeyMismatch,
		})

		// Wait for user response
		accepted := <-a.pendingHostKey
		return accepted
	})

	client, err := ssh.Connect(cfg, hostKeyCallback)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(a.ctx)

	a.mu.Lock()
	a.activeSessions[tabID] = &ActiveSession{
		session:   client,
		sessionID: tabID,
		cancel:    cancel,
		sshClient: client,
	}
	a.mu.Unlock()

	runtime.EventsEmit(a.ctx, "ssh:connected", tabID)

	a.startSessionIO(tabID, client, ctx)

	return nil
}

// ---- Local Terminal ----

type OpenShellRequest struct {
	TabID string `json:"tabId"`
	Shell string `json:"shell"`
	Cols  int    `json:"cols"`
	Rows  int    `json:"rows"`
}

func (a *App) OpenLocalShell(req OpenShellRequest) error {
	shell := req.Shell
	if shell == "" {
		shell = terminal.DefaultShell()
	}

	cols := req.Cols
	if cols == 0 {
		cols = 120
	}
	rows := req.Rows
	if rows == 0 {
		rows = 40
	}

	ls, err := terminal.NewLocalShell(shell, cols, rows)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(a.ctx)

	a.mu.Lock()
	a.activeSessions[req.TabID] = &ActiveSession{
		session:   ls,
		sessionID: req.TabID,
		cancel:    cancel,
	}
	a.mu.Unlock()

	a.startSessionIO(req.TabID, ls, ctx)

	return nil
}

func (a *App) GetAvailableShells() []terminal.ShellInfo {
	return terminal.GetAvailableShells()
}

// ---- External Terminal ----

func (a *App) GetAvailableTerminals() []terminal.TerminalInfo {
	return terminal.GetAvailableTerminals()
}

func (a *App) ConnectSessionExternal(sessionID string) error {
	sess := a.store.GetSession(sessionID)
	if sess == nil {
		return fmt.Errorf("session not found")
	}

	termPath := sess.SystemTerminal
	if termPath == "" {
		def := terminal.DefaultTerminal()
		if def == nil {
			return fmt.Errorf("no system terminal available")
		}
		termPath = def.Path
	}

	keyPath := ""
	if sess.AuthMethod == models.AuthPrivateKey {
		keyPath = sess.PrivateKeyPath
	}

	return terminal.LaunchExternalSSH(termPath, sess.Host, sess.Port, sess.Username, keyPath)
}

// ---- Move Session/Folder ----

func (a *App) MoveSession(sessionID, newFolderID string) error {
	return a.store.MoveSession(sessionID, newFolderID)
}

func (a *App) MoveFolder(folderID, newParentID string) error {
	return a.store.MoveFolder(folderID, newParentID)
}

// ---- Shared I/O ----

func (a *App) startSessionIO(tabID string, sess terminal.TerminalSession, ctx context.Context) {
	// Read stdout and emit to frontend
	go func() {
		buf := make([]byte, 32*1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := sess.Read(buf)
				if n > 0 {
					encoded := base64.StdEncoding.EncodeToString(buf[:n])
					runtime.EventsEmit(a.ctx, "terminal:data:"+tabID, encoded)
				}
				if err != nil {
					runtime.EventsEmit(a.ctx, "terminal:closed:"+tabID, err.Error())
					return
				}
			}
		}
	}()

	// Read stderr and emit
	go func() {
		buf := make([]byte, 32*1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := sess.ReadStderr(buf)
				if n > 0 {
					encoded := base64.StdEncoding.EncodeToString(buf[:n])
					runtime.EventsEmit(a.ctx, "terminal:data:"+tabID, encoded)
				}
				if err != nil {
					return
				}
			}
		}
	}()

	// Watch for session end
	go func() {
		select {
		case <-sess.Done():
			runtime.EventsEmit(a.ctx, "terminal:closed:"+tabID, "session ended")
		case <-ctx.Done():
		}
	}()
}

func (a *App) SendInput(tabID string, data string) error {
	a.mu.Lock()
	as, ok := a.activeSessions[tabID]
	a.mu.Unlock()

	if !ok {
		return fmt.Errorf("no active session for tab %s", tabID)
	}

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return err
	}

	_, err = as.session.Write(decoded)
	return err
}

func (a *App) ResizeTerminal(tabID string, cols int, rows int) error {
	a.mu.Lock()
	as, ok := a.activeSessions[tabID]
	a.mu.Unlock()

	if !ok {
		return nil // Silently ignore if no session
	}

	return as.session.Resize(cols, rows)
}

func (a *App) DisconnectTab(tabID string) {
	a.mu.Lock()
	as, ok := a.activeSessions[tabID]
	if ok {
		delete(a.activeSessions, tabID)
	}
	a.mu.Unlock()

	if ok {
		if as.sftp != nil {
			as.sftp.Close()
		}
		as.cancel()
		as.session.Close()
	}
}

// ---- Host Key ----

func (a *App) AcceptHostKey(accepted bool) {
	a.pendingHostKeyLock.Lock()
	defer a.pendingHostKeyLock.Unlock()
	if a.pendingHostKey != nil {
		a.pendingHostKey <- accepted
	}
}

// ---- Config ----

func (a *App) GetConfig() config.AppConfig {
	return a.config
}

func (a *App) SaveConfig(cfg config.AppConfig) error {
	a.config = cfg
	return config.Save(a.store.DataDir(), cfg)
}

// ---- Command History ----

func (a *App) SaveCommandToHistory(sessionID string, command string) error {
	if sessionID == "" || command == "" {
		return nil
	}

	histDir := filepath.Join(a.store.DataDir(), "history")
	if err := os.MkdirAll(histDir, 0700); err != nil {
		return err
	}

	histFile := filepath.Join(histDir, sessionID+".json")

	var history []string
	if data, err := os.ReadFile(histFile); err == nil {
		json.Unmarshal(data, &history)
	}

	// Remove duplicate if exists
	for i, h := range history {
		if h == command {
			history = append(history[:i], history[i+1:]...)
			break
		}
	}

	// Prepend (most recent first)
	history = append([]string{command}, history...)

	// Cap at 1000
	if len(history) > 1000 {
		history = history[:1000]
	}

	data, err := json.Marshal(history)
	if err != nil {
		return err
	}
	return os.WriteFile(histFile, data, 0600)
}

func (a *App) GetCommandHistory(sessionID string) []string {
	if sessionID == "" {
		return nil
	}

	histFile := filepath.Join(a.store.DataDir(), "history", sessionID+".json")

	var history []string
	if data, err := os.ReadFile(histFile); err == nil {
		json.Unmarshal(data, &history)
	}
	return history
}

// ---- Remote Completions ----

func (a *App) FetchRemoteCompletions(tabID string) (*ssh.RemoteCompletions, error) {
	a.mu.Lock()
	as, ok := a.activeSessions[tabID]
	a.mu.Unlock()

	if !ok || as.sshClient == nil {
		return nil, fmt.Errorf("no SSH session for tab %s", tabID)
	}

	conn := as.sshClient.Conn()
	if conn == nil {
		return nil, fmt.Errorf("SSH connection closed")
	}

	return ssh.FetchCompletions(conn), nil
}

// ---- SFTP / SCP ----

// ensureSftp lazily creates an SFTP session for the given tab.
func (a *App) ensureSftp(tabID string) (*ssh.SftpSession, *ssh.Client, error) {
	a.mu.Lock()
	as, ok := a.activeSessions[tabID]
	a.mu.Unlock()

	if !ok {
		return nil, nil, fmt.Errorf("no active session for tab %s", tabID)
	}
	if as.sshClient == nil {
		return nil, nil, fmt.Errorf("not an SSH session")
	}

	if as.sftp != nil {
		return as.sftp, as.sshClient, nil
	}

	conn := as.sshClient.Conn()
	if conn == nil {
		return nil, nil, fmt.Errorf("SSH connection closed")
	}

	sftp, err := ssh.NewSftpSession(conn)
	if err != nil {
		return nil, nil, err
	}

	a.mu.Lock()
	as.sftp = sftp
	a.mu.Unlock()

	return sftp, as.sshClient, nil
}

type SftpListResult struct {
	Path  string          `json:"path"`
	Files []ssh.FileEntry `json:"files"`
}

func (a *App) SftpListDir(tabID string, dirPath string) (*SftpListResult, error) {
	sftp, _, err := a.ensureSftp(tabID)
	if err != nil {
		return nil, err
	}

	if dirPath == "" || dirPath == "~" {
		dirPath = sftp.GetHome()
	}

	files, err := sftp.ListDir(dirPath)
	if err != nil {
		return nil, err
	}

	return &SftpListResult{
		Path:  dirPath,
		Files: files,
	}, nil
}

func (a *App) SftpDownloadFile(tabID string, remotePath string) error {
	sftp, _, err := a.ensureSftp(tabID)
	if err != nil {
		return err
	}

	// Get just the filename for the save dialog default
	parts := splitPath(remotePath)
	defaultName := "file"
	if len(parts) > 0 {
		defaultName = parts[len(parts)-1]
	}

	localPath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save File",
		DefaultFilename: defaultName,
	})
	if err != nil || localPath == "" {
		return err
	}

	return sftp.ReadFileToLocal(remotePath, localPath)
}

func (a *App) SftpUploadFile(tabID string, remoteDir string) error {
	sftp, _, err := a.ensureSftp(tabID)
	if err != nil {
		return err
	}

	localPath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select File to Upload",
	})
	if err != nil || localPath == "" {
		return err
	}

	// Extract filename from local path
	localParts := splitPath(localPath)
	fileName := localParts[len(localParts)-1]
	remotePath := remoteDir + "/" + fileName

	return sftp.UploadFile(localPath, remotePath)
}

// SftpUploadPaths uploads local files (by absolute path) to the remote directory.
func (a *App) SftpUploadPaths(tabID string, remoteDir string, localPaths []string) (int, error) {
	sftpSession, _, err := a.ensureSftp(tabID)
	if err != nil {
		return 0, err
	}

	uploaded := 0
	for _, lp := range localPaths {
		info, err := os.Stat(lp)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue // skip directories
		}
		fileName := filepath.Base(lp)
		remotePath := remoteDir + "/" + fileName
		if err := sftpSession.UploadFile(lp, remotePath); err != nil {
			return uploaded, fmt.Errorf("failed to upload %s: %w", fileName, err)
		}
		uploaded++
	}
	return uploaded, nil
}

// SftpDownloadToDownloads downloads a remote file to the user's Downloads folder and returns the local path.
func (a *App) SftpDownloadToDownloads(tabID string, remotePath string) (string, error) {
	sftpSession, _, err := a.ensureSftp(tabID)
	if err != nil {
		return "", err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find home directory: %w", err)
	}
	downloadsDir := filepath.Join(homeDir, "Downloads")
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		return "", err
	}

	parts := splitPath(remotePath)
	fileName := "file"
	if len(parts) > 0 {
		fileName = parts[len(parts)-1]
	}

	localPath := filepath.Join(downloadsDir, fileName)
	// Avoid overwriting: append (1), (2), etc.
	if _, err := os.Stat(localPath); err == nil {
		ext := filepath.Ext(fileName)
		base := fileName[:len(fileName)-len(ext)]
		for i := 1; ; i++ {
			localPath = filepath.Join(downloadsDir, fmt.Sprintf("%s (%d)%s", base, i, ext))
			if _, err := os.Stat(localPath); os.IsNotExist(err) {
				break
			}
		}
	}

	if err := sftpSession.ReadFileToLocal(remotePath, localPath); err != nil {
		return "", err
	}
	return localPath, nil
}

func (a *App) SftpGetPwd(tabID string) (string, error) {
	_, sshClient, err := a.ensureSftp(tabID)
	if err != nil {
		return "", err
	}

	conn := sshClient.Conn()
	if conn == nil {
		return "", fmt.Errorf("SSH connection closed")
	}

	return ssh.ExecPwd(conn)
}

func (a *App) SftpGetHome(tabID string) (string, error) {
	sftp, _, err := a.ensureSftp(tabID)
	if err != nil {
		return "", err
	}
	return sftp.GetHome(), nil
}

// IsSSHSession returns true if the tab is an SSH session (not local terminal).
func (a *App) IsSSHSession(tabID string) bool {
	a.mu.Lock()
	as, ok := a.activeSessions[tabID]
	a.mu.Unlock()
	return ok && as.sshClient != nil
}

func splitPath(p string) []string {
	var parts []string
	for _, s := range []string{"/", "\\"} {
		if idx := lastIndexOf(p, s); idx >= 0 {
			parts = append(parts, p[idx+1:])
			return parts
		}
	}
	return []string{p}
}

func lastIndexOf(s, sep string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == sep[0] {
			return i
		}
	}
	return -1
}

// ---- Duplicate Session ----

func (a *App) DuplicateSession(id string) error {
	sess := a.store.GetSession(id)
	if sess == nil {
		return fmt.Errorf("session not found")
	}
	dupe := *sess
	dupe.ID = uuid.New().String()
	dupe.Name = sess.Name + " (copy)"
	dupe.CreatedAt = time.Now().Unix()
	dupe.UpdatedAt = time.Now().Unix()
	return a.store.SaveSession(dupe)
}

// ---- Import / Export ----

func (a *App) ExportSessions() (string, error) {
	data := a.store.ExportAll()
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export Sessions",
		DefaultFilename: "terminalis-sessions.json",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files", Pattern: "*.json"},
		},
	})
	if err != nil || path == "" {
		return "", err
	}

	if err := os.WriteFile(path, jsonBytes, 0600); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) ImportSessions() (int, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import Sessions",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files", Pattern: "*.json"},
		},
	})
	if err != nil || path == "" {
		return 0, err
	}

	jsonBytes, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	var data models.SessionData
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return 0, fmt.Errorf("invalid JSON: %w", err)
	}

	count := 0
	// Build ID mapping: old ID -> new ID
	idMap := make(map[string]string)
	for _, f := range data.Folders {
		idMap[f.ID] = uuid.New().String()
	}

	// Import folders with remapped IDs
	for _, f := range data.Folders {
		oldID := f.ID
		f.ID = idMap[oldID]
		if f.ParentID != "" {
			if mapped, ok := idMap[f.ParentID]; ok {
				f.ParentID = mapped
			} else {
				f.ParentID = ""
			}
		}
		f.Expanded = true
		if err := a.store.SaveFolder(f); err == nil {
			count++
		}
	}

	// Import sessions with new IDs
	for _, s := range data.Sessions {
		s.ID = uuid.New().String()
		if mapped, ok := idMap[s.FolderID]; ok {
			s.FolderID = mapped
		} else {
			s.FolderID = ""
		}
		s.CreatedAt = time.Now().Unix()
		s.UpdatedAt = time.Now().Unix()
		if err := a.store.SaveSession(s); err == nil {
			count++
		}
	}

	return count, nil
}

// ---- File Picker ----

func (a *App) SelectPrivateKeyFile() string {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Private Key File",
		Filters: []runtime.FileFilter{
			{DisplayName: "All Files", Pattern: "*.*"},
			{DisplayName: "PEM Files", Pattern: "*.pem"},
			{DisplayName: "Key Files", Pattern: "*.key"},
		},
	})
	if err != nil {
		return ""
	}
	return path
}
