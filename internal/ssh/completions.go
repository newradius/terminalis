package ssh

import (
	"strings"

	gossh "golang.org/x/crypto/ssh"
)

type RemoteCompletions struct {
	History  []string `json:"history"`
	Commands []string `json:"commands"`
}

func FetchCompletions(conn *gossh.Client) *RemoteCompletions {
	if conn == nil {
		return &RemoteCompletions{}
	}

	shell := detectShell(conn)
	return &RemoteCompletions{
		History:  readHistory(conn, shell),
		Commands: fetchCommands(conn, shell),
	}
}

func detectShell(conn *gossh.Client) string {
	session, err := conn.NewSession()
	if err != nil {
		return "bash"
	}
	defer session.Close()

	out, err := session.Output("echo $SHELL")
	if err != nil {
		return "bash"
	}
	s := strings.TrimSpace(string(out))
	if strings.Contains(s, "zsh") {
		return "zsh"
	}
	if strings.Contains(s, "fish") {
		return "fish"
	}
	return "bash"
}

func readHistory(conn *gossh.Client, shell string) []string {
	var cmd string
	switch shell {
	case "zsh":
		cmd = "cat ~/.zsh_history 2>/dev/null || cat ~/.histfile 2>/dev/null"
	case "fish":
		cmd = "cat ~/.local/share/fish/fish_history 2>/dev/null"
	default:
		cmd = "cat ~/.bash_history 2>/dev/null"
	}

	session, err := conn.NewSession()
	if err != nil {
		return nil
	}
	defer session.Close()

	out, err := session.Output(cmd)
	if err != nil || len(out) == 0 {
		return nil
	}

	lines := strings.Split(string(out), "\n")
	seen := make(map[string]bool)
	var history []string

	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// zsh history format: ": timestamp:0;command"
		if shell == "zsh" && strings.HasPrefix(line, ": ") {
			if idx := strings.Index(line, ";"); idx >= 0 {
				line = line[idx+1:]
			} else {
				continue
			}
		}

		// fish history format: "- cmd: command"
		if shell == "fish" {
			if strings.HasPrefix(line, "- cmd: ") {
				line = strings.TrimPrefix(line, "- cmd: ")
			} else {
				continue
			}
		}

		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		history = append(history, line)

		if len(history) >= 5000 {
			break
		}
	}

	return history
}

func fetchCommands(conn *gossh.Client, shell string) []string {
	cmd := "bash -ic 'compgen -c 2>/dev/null' || { ls /usr/bin /usr/sbin /bin /sbin 2>/dev/null | sort -u; }"

	session, err := conn.NewSession()
	if err != nil {
		return nil
	}
	defer session.Close()

	out, err := session.Output(cmd)
	if err != nil || len(out) == 0 {
		// Fallback: just list common bin dirs
		session2, err := conn.NewSession()
		if err != nil {
			return nil
		}
		defer session2.Close()
		out, err = session2.Output("ls /usr/bin /usr/sbin /bin /sbin 2>/dev/null | sort -u")
		if err != nil {
			return nil
		}
	}

	lines := strings.Split(string(out), "\n")
	seen := make(map[string]bool)
	var commands []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		commands = append(commands, line)
	}

	return commands
}
