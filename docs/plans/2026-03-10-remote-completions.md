# Remote-Enhanced Autosuggestions Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extend terminal autosuggestions to fetch remote shell history and available commands from SSH servers, providing zsh-like suggestions from three sources: local history, remote history, and remote command names.

**Architecture:** On SSH connect, the Go backend opens a separate SSH exec channel to detect the shell type, reads the remote history file via SFTP, and runs a command to list available commands. Results are returned to the frontend which merges them with local history for suggestion matching. Remote history is used for full-line suggestions; remote commands are used for first-word-only suggestions.

**Tech Stack:** Go (SSH exec channels, SFTP file reading), Svelte/TypeScript (suggestion matching logic)

---

### Task 1: Go backend — FetchRemoteCompletions method

**Files:**
- Modify: `app.go:455-509` (add new method in Command History section)
- Modify: `frontend/wailsjs/go/main/App.js` (add JS binding)
- Modify: `frontend/wailsjs/go/main/App.d.ts` (add TS declaration)

**Step 1: Add the RemoteCompletions struct and FetchRemoteCompletions method**

Add after `GetCommandHistory` (after line 509) in `app.go`:

```go
type RemoteCompletions struct {
	History  []string `json:"history"`
	Commands []string `json:"commands"`
}

func (a *App) FetchRemoteCompletions(tabID string) (*RemoteCompletions, error) {
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

	result := &RemoteCompletions{}

	// Detect shell type
	shell := detectRemoteShell(conn)

	// Read remote history file
	result.History = readRemoteHistory(conn, shell)

	// Fetch available commands
	result.Commands = fetchRemoteCommands(conn, shell)

	return result, nil
}

func detectRemoteShell(conn *gossh.Client) string {
	session, err := conn.NewSession()
	if err != nil {
		return "bash"
	}
	defer session.Close()

	out, err := session.Output("echo $SHELL")
	if err != nil {
		return "bash"
	}
	shell := strings.TrimSpace(string(out))
	if strings.Contains(shell, "zsh") {
		return "zsh"
	}
	if strings.Contains(shell, "fish") {
		return "fish"
	}
	return "bash"
}

func readRemoteHistory(conn *gossh.Client, shell string) []string {
	// Determine history file path based on shell
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

	// Process in reverse (most recent last in file = most recent first in our list)
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

func fetchRemoteCommands(conn *gossh.Client, shell string) []string {
	var cmd string
	switch shell {
	case "zsh":
		cmd = "whence -pm '*' 2>/dev/null | xargs -I{} basename {} 2>/dev/null || compgen -c 2>/dev/null || ls /usr/bin /usr/sbin /bin /sbin 2>/dev/null | sort -u"
	default:
		cmd = "compgen -c 2>/dev/null || ls /usr/bin /usr/sbin /bin /sbin 2>/dev/null | sort -u"
	}

	session, err := conn.NewSession()
	if err != nil {
		return nil
	}
	defer session.Close()

	// Request a bash shell for compgen (it's a bash builtin)
	if shell != "zsh" {
		cmd = "bash -c '" + cmd + "'"
	}

	out, err := session.Output(cmd)
	if err != nil || len(out) == 0 {
		return nil
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
```

Note: This requires adding `"strings"` and `gossh "golang.org/x/crypto/ssh"` to the imports in `app.go`. The `strings` import is already present? No — check the current imports. If not present, add it. The `gossh` import is NOT in app.go — the SSH types are used via the `ssh` internal package. We need to add `gossh "golang.org/x/crypto/ssh"` to run exec sessions directly, OR add a helper to `internal/ssh/sftp.go`.

**Better approach:** Add a generic `ExecCommand` function to `internal/ssh/sftp.go` next to `ExecPwd`, and move the shell detection/history/commands logic to a new file `internal/ssh/completions.go`. This keeps app.go clean.

Actually, the cleanest approach: add a single function `FetchCompletions(conn *gossh.Client) (history []string, commands []string)` in the `ssh` package, since `ExecPwd` already shows the pattern. Then `app.go` just calls `ssh.FetchCompletions(conn)`.

**Revised step 1: Create `internal/ssh/completions.go`**

```go
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

		if shell == "zsh" && strings.HasPrefix(line, ": ") {
			if idx := strings.Index(line, ";"); idx >= 0 {
				line = line[idx+1:]
			} else {
				continue
			}
		}

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
```

**Step 2: Add FetchRemoteCompletions to app.go**

Add after `GetCommandHistory` (after line 509):

```go
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
```

**Step 3: Add Wails bindings**

In `frontend/wailsjs/go/main/App.js`, add:

```js
export function FetchRemoteCompletions(arg1) {
  return window['go']['main']['App']['FetchRemoteCompletions'](arg1);
}
```

In `frontend/wailsjs/go/main/App.d.ts`, add:

```ts
export function FetchRemoteCompletions(arg1:string):Promise<{history:Array<string>,commands:Array<string>}>;
```

**Step 4: Verify Go compiles**

Run: `cd C:/Users/sabin/Desktop/Terminalis && go build ./...`
Expected: No errors

**Step 5: Commit**

```bash
git add internal/ssh/completions.go app.go frontend/wailsjs/go/main/App.js frontend/wailsjs/go/main/App.d.ts
git commit -m "feat: add remote shell history and command fetching backend"
```

---

### Task 2: Frontend — integrate remote completions into suggestion matching

**Files:**
- Modify: `frontend/src/lib/components/TerminalView.svelte:6,23-37,205-210`

**Step 1: Add import and state variables**

In `TerminalView.svelte`, update the import on line 6 to add `FetchRemoteCompletions`:

```ts
import { SendInput, ResizeTerminal, DisconnectTab, SaveCommandToHistory, GetCommandHistory, FetchRemoteCompletions } from "../../../wailsjs/go/main/App";
```

Add new state variables after `currentSuggestion` (line 26):

```ts
let remoteHistory: string[] = [];
let remoteCommands: string[] = [];
```

**Step 2: Update updateSuggestion to search all sources with first-word logic**

Replace the existing `updateSuggestion` function (lines 28-37) with:

```ts
function updateSuggestion() {
  if (!currentInput) {
    currentSuggestion = "";
    return;
  }

  const isFirstWord = !currentInput.includes(" ");

  // Search local history first (full line match)
  let match = commandHistory.find((h) =>
    h.startsWith(currentInput) && h !== currentInput
  );

  // Then remote history (full line match)
  if (!match) {
    match = remoteHistory.find((h) =>
      h.startsWith(currentInput) && h !== currentInput
    );
  }

  // Then remote commands (first word only)
  if (!match && isFirstWord) {
    const cmdMatch = remoteCommands.find((c) =>
      c.startsWith(currentInput) && c !== currentInput
    );
    if (cmdMatch) {
      match = cmdMatch;
    }
  }

  currentSuggestion = match ? match.slice(currentInput.length) : "";
}
```

**Step 3: Fetch remote completions after SSH connect**

Add an `EventsOn` handler inside `onMount()`, after the history load block (after line 210). Also fetch if already connected:

```ts
// Fetch remote completions for SSH sessions
EventsOn("ssh:connected", (connectedTabId: string) => {
  if (connectedTabId === tabId) {
    FetchRemoteCompletions(tabId).then((r) => {
      if (r) {
        remoteHistory = r.history || [];
        remoteCommands = r.commands || [];
      }
    }).catch(() => {});
  }
});
```

**Step 4: Clean up the event listener on destroy**

In `onDestroy()`, add after the existing `EventsOff` calls:

```ts
EventsOff("ssh:connected:" + tabId);
```

Wait — the `ssh:connected` event doesn't use a tab-specific suffix. The handler already filters by `connectedTabId === tabId`. But we need a way to clean up. We can use a different approach: store the unsubscribe and call it. Actually, `EventsOff` with just the event name will unsubscribe ALL listeners for that event, which would break other tabs.

Better approach: Since `ssh:connected` is already emitted globally and filtered inside the handler, and the handler does nothing if the tabId doesn't match, we can safely skip cleanup — the handler will be garbage collected with the component. But to be safe, let's use a tab-specific approach:

Actually looking at the existing code, `EventsOn("terminal:data:" + tabId, ...)` uses tab-specific events and `EventsOff("terminal:data:" + tabId)` cleans them up. But `ssh:connected` is a global event. The simplest safe approach: just try fetching right after history load (line 210), since by the time the TerminalView mounts, the SSH connection is already established:

```ts
// Fetch remote completions for SSH sessions (non-blocking)
FetchRemoteCompletions(tabId).then((r) => {
  if (r) {
    remoteHistory = r.history || [];
    remoteCommands = r.commands || [];
  }
}).catch(() => {});
```

This is simpler — `FetchRemoteCompletions` will return an error for non-SSH tabs (which we catch and ignore), and for SSH tabs it will fetch immediately. No event listener needed.

**Step 5: Build frontend**

Run: `cd C:/Users/sabin/Desktop/Terminalis/frontend && npm run build`
Expected: Build succeeds

**Step 6: Commit**

```bash
git add frontend/src/lib/components/TerminalView.svelte
git commit -m "feat: integrate remote history and commands into autosuggestions"
```

---

### Task 3: Build and verify

**Step 1: Verify Go compiles**

Run: `cd C:/Users/sabin/Desktop/Terminalis && go build ./...`

**Step 2: Verify frontend builds**

Run: `cd C:/Users/sabin/Desktop/Terminalis/frontend && npm run build`

**Step 3: Manual test checklist**

- [ ] Connect to an SSH server with bash — remote history loads
- [ ] Type first word of a known remote command — command name suggestion appears
- [ ] Type partial of a previous remote command (with args) — full line suggestion appears
- [ ] Local history still takes priority over remote
- [ ] Local terminal tabs don't error (FetchRemoteCompletions fails silently)
- [ ] Right Arrow still accepts suggestions
- [ ] Tab key still passes through to remote shell
