package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// TerminalInfo describes an available system terminal emulator.
type TerminalInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// GetAvailableTerminals returns terminal emulators available on the current system.
func GetAvailableTerminals() []TerminalInfo {
	var terminals []TerminalInfo

	switch runtime.GOOS {
	case "windows":
		candidates := []struct {
			name string
			path string
		}{
			{"Windows Terminal", "wt.exe"},
			{"PowerShell 7", "pwsh.exe"},
			{"PowerShell", "powershell.exe"},
			{"CMD", "cmd.exe"},
			{"Git Bash", "mintty.exe"},
		}
		for _, c := range candidates {
			if path, err := exec.LookPath(c.path); err == nil {
				terminals = append(terminals, TerminalInfo{Name: c.name, Path: path})
			}
		}

	case "darwin":
		// Terminal.app is always available
		terminals = append(terminals, TerminalInfo{Name: "Terminal.app", Path: "/System/Applications/Utilities/Terminal.app"})

		macApps := []struct {
			name string
			path string
		}{
			{"iTerm2", "/Applications/iTerm.app"},
			{"Alacritty", "/Applications/Alacritty.app"},
			{"Kitty", "/Applications/kitty.app"},
			{"WezTerm", "/Applications/WezTerm.app"},
		}
		for _, app := range macApps {
			if _, err := os.Stat(app.path); err == nil {
				terminals = append(terminals, TerminalInfo{Name: app.name, Path: app.path})
			}
		}

	default: // linux
		candidates := []struct {
			name string
			path string
		}{
			{"GNOME Terminal", "gnome-terminal"},
			{"Konsole", "konsole"},
			{"Xfce Terminal", "xfce4-terminal"},
			{"Alacritty", "alacritty"},
			{"Kitty", "kitty"},
			{"WezTerm", "wezterm"},
			{"Tilix", "tilix"},
			{"xterm", "xterm"},
		}
		for _, c := range candidates {
			if path, err := exec.LookPath(c.path); err == nil {
				terminals = append(terminals, TerminalInfo{Name: c.name, Path: path})
			}
		}
	}

	return terminals
}

// DefaultTerminal returns the best default terminal for the current OS.
func DefaultTerminal() *TerminalInfo {
	terminals := GetAvailableTerminals()
	if len(terminals) > 0 {
		return &terminals[0]
	}
	return nil
}

// BuildSSHCommand builds the command arguments to launch SSH in an external terminal.
func BuildSSHCommand(terminalPath, host string, port int, username, privateKeyPath string) *exec.Cmd {
	// Build the ssh command parts
	sshArgs := []string{"ssh"}
	if privateKeyPath != "" {
		sshArgs = append(sshArgs, "-i", privateKeyPath)
	}
	if port != 0 && port != 22 {
		sshArgs = append(sshArgs, "-p", fmt.Sprintf("%d", port))
	}
	sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", username, host))

	termName := strings.ToLower(terminalPath)

	switch runtime.GOOS {
	case "windows":
		return buildWindowsCommand(terminalPath, termName, sshArgs)
	case "darwin":
		return buildDarwinCommand(terminalPath, termName, sshArgs)
	default:
		return buildLinuxCommand(terminalPath, termName, sshArgs)
	}
}

func buildWindowsCommand(terminalPath, termName string, sshArgs []string) *exec.Cmd {
	switch {
	case strings.Contains(termName, "wt.exe") || strings.Contains(termName, "wt"):
		// Windows Terminal: wt new-tab -- ssh ...
		args := []string{"new-tab", "--"}
		args = append(args, sshArgs...)
		return exec.Command(terminalPath, args...)

	case strings.Contains(termName, "mintty"):
		// Git Bash mintty: mintty -e ssh ...
		args := []string{"-e"}
		args = append(args, sshArgs...)
		return exec.Command(terminalPath, args...)

	default:
		// CMD / PowerShell: cmd /c ssh ... or powershell -Command ssh ...
		if strings.Contains(termName, "powershell") || strings.Contains(termName, "pwsh") {
			sshCmd := strings.Join(sshArgs, " ")
			return exec.Command(terminalPath, "-NoExit", "-Command", sshCmd)
		}
		// CMD
		sshCmd := strings.Join(sshArgs, " ")
		return exec.Command(terminalPath, "/c", sshCmd)
	}
}

func buildDarwinCommand(terminalPath, termName string, sshArgs []string) *exec.Cmd {
	sshCmd := strings.Join(sshArgs, " ")

	switch {
	case strings.Contains(termName, "iterm"):
		// Use osascript to open iTerm2 with SSH command
		script := fmt.Sprintf(`tell application "iTerm"
	create window with default profile command "%s"
end tell`, sshCmd)
		return exec.Command("osascript", "-e", script)

	case strings.Contains(termName, "terminal.app"):
		// Use osascript to open Terminal.app
		script := fmt.Sprintf(`tell application "Terminal"
	activate
	do script "%s"
end tell`, sshCmd)
		return exec.Command("osascript", "-e", script)

	default:
		// For Alacritty, Kitty, WezTerm - use open -a with --args
		args := []string{"-a", terminalPath}
		// Alacritty/Kitty/WezTerm support -e flag
		args = append(args, "--args", "-e")
		args = append(args, sshArgs...)
		return exec.Command("open", args...)
	}
}

func buildLinuxCommand(terminalPath, termName string, sshArgs []string) *exec.Cmd {
	switch {
	case strings.Contains(termName, "gnome-terminal"):
		// gnome-terminal -- ssh ...
		args := []string{"--"}
		args = append(args, sshArgs...)
		return exec.Command(terminalPath, args...)

	case strings.Contains(termName, "konsole"):
		// konsole -e ssh ...
		args := []string{"-e"}
		args = append(args, sshArgs...)
		return exec.Command(terminalPath, args...)

	case strings.Contains(termName, "xfce4-terminal"):
		// xfce4-terminal -e "ssh ..."
		sshCmd := strings.Join(sshArgs, " ")
		return exec.Command(terminalPath, "-e", sshCmd)

	case strings.Contains(termName, "tilix"):
		// tilix -e ssh ...
		args := []string{"-e"}
		args = append(args, sshArgs...)
		return exec.Command(terminalPath, args...)

	default:
		// Alacritty, Kitty, WezTerm, xterm all support -e
		args := []string{"-e"}
		args = append(args, sshArgs...)
		return exec.Command(terminalPath, args...)
	}
}

// LaunchExternalSSH launches an SSH connection in an external terminal emulator.
// The process is started detached and not waited on.
func LaunchExternalSSH(terminalPath, host string, port int, username, privateKeyPath string) error {
	cmd := BuildSSHCommand(terminalPath, host, port, username, privateKeyPath)

	// Detach the process so it doesn't die when we exit
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to launch external terminal: %w", err)
	}

	// Release the process so it runs independently
	go cmd.Wait()

	return nil
}
