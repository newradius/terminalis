package terminal

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"

	gopty "github.com/aymanbagabas/go-pty"
)

// ShellInfo describes an available shell.
type ShellInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// LocalShell implements TerminalSession for a local shell process.
type LocalShell struct {
	mu   sync.Mutex
	pty  gopty.Pty
	cmd  *gopty.Cmd
	done chan struct{}
}

// NewLocalShell spawns a local shell with a PTY.
func NewLocalShell(shell string, cols, rows int) (*LocalShell, error) {
	ptty, err := gopty.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create PTY: %w", err)
	}

	if err := ptty.Resize(cols, rows); err != nil {
		ptty.Close()
		return nil, fmt.Errorf("failed to resize PTY: %w", err)
	}

	cmd := ptty.Command(shell)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")

	if err := cmd.Start(); err != nil {
		ptty.Close()
		return nil, fmt.Errorf("failed to start shell: %w", err)
	}

	ls := &LocalShell{
		pty:  ptty,
		cmd:  cmd,
		done: make(chan struct{}),
	}

	go func() {
		cmd.Wait()
		close(ls.done)
	}()

	return ls, nil
}

func (l *LocalShell) Read(buf []byte) (int, error) {
	return l.pty.Read(buf)
}

func (l *LocalShell) ReadStderr(_ []byte) (int, error) {
	return 0, io.EOF
}

func (l *LocalShell) Write(data []byte) (int, error) {
	return l.pty.Write(data)
}

func (l *LocalShell) Resize(cols, rows int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.pty == nil {
		return fmt.Errorf("PTY closed")
	}
	return l.pty.Resize(cols, rows)
}

func (l *LocalShell) Done() <-chan struct{} {
	return l.done
}

func (l *LocalShell) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.cmd != nil && l.cmd.Process != nil {
		l.cmd.Process.Kill()
		l.cmd = nil
	}
	if l.pty != nil {
		l.pty.Close()
		l.pty = nil
	}
	return nil
}

// GetAvailableShells returns shells available on the current system.
func GetAvailableShells() []ShellInfo {
	var shells []ShellInfo

	if runtime.GOOS == "windows" {
		candidates := []ShellInfo{
			{Name: "PowerShell", Path: "powershell.exe"},
			{Name: "CMD", Path: "cmd.exe"},
			{Name: "Git Bash", Path: "bash.exe"},
		}
		// Also check for pwsh (PowerShell Core)
		if path, err := exec.LookPath("pwsh.exe"); err == nil {
			shells = append(shells, ShellInfo{Name: "PowerShell 7", Path: path})
		}
		for _, c := range candidates {
			if _, err := exec.LookPath(c.Path); err == nil {
				shells = append(shells, c)
			}
		}
	} else {
		candidates := []ShellInfo{
			{Name: "Zsh", Path: "/bin/zsh"},
			{Name: "Bash", Path: "/bin/bash"},
			{Name: "Fish", Path: "/usr/bin/fish"},
			{Name: "sh", Path: "/bin/sh"},
		}
		for _, c := range candidates {
			if _, err := os.Stat(c.Path); err == nil {
				shells = append(shells, c)
			}
		}
	}

	return shells
}

// DefaultShell returns the default shell for the current system.
func DefaultShell() string {
	if runtime.GOOS == "windows" {
		if path, err := exec.LookPath("pwsh.exe"); err == nil {
			return path
		}
		return "powershell.exe"
	}
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/sh"
}
