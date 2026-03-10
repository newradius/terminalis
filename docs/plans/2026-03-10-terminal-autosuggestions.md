# Terminal Autosuggestions Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add per-session inline ghost-text autosuggestions (like zsh-autosuggestions) to the terminal.

**Architecture:** Go backend persists command history per-session in JSON files. Frontend tracks the current input buffer by intercepting keystrokes, searches history for prefix matches, and renders ghost text as an overlay div positioned at the cursor. Right-arrow accepts the suggestion. All suggestion logic is frontend-only for zero latency.

**Tech Stack:** Go (backend history storage), Svelte/TypeScript (frontend input tracking + ghost text rendering), xterm.js (terminal emulation)

---

### Task 1: Go backend — history storage methods

**Files:**
- Modify: `app.go` (add two new exported methods + helper)

**Step 1: Add SaveCommandToHistory and GetCommandHistory methods**

Add these methods to `app.go` after the Config section (after line 453):

```go
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
```

**Step 2: Regenerate Wails bindings**

Run: `wails generate module`

This will auto-generate the JS/TS bindings in `frontend/wailsjs/go/main/App.js` and `App.d.ts`.

If `wails generate module` doesn't work, manually add to `frontend/wailsjs/go/main/App.js`:

```js
export function SaveCommandToHistory(arg1, arg2) {
  return window['go']['main']['App']['SaveCommandToHistory'](arg1, arg2);
}

export function GetCommandHistory(arg1) {
  return window['go']['main']['App']['GetCommandHistory'](arg1);
}
```

And to `frontend/wailsjs/go/main/App.d.ts`:

```ts
export function SaveCommandToHistory(arg1:string,arg2:string):Promise<void>;
export function GetCommandHistory(arg1:string):Promise<Array<string>>;
```

**Step 3: Commit**

```bash
git add app.go frontend/wailsjs/go/main/App.js frontend/wailsjs/go/main/App.d.ts
git commit -m "feat: add per-session command history storage backend"
```

---

### Task 2: Pass historyId prop to TerminalView

The `TerminalView` component needs to know which session's history to use. The tab already stores `sessionId` for saved sessions. For quick-connect and local terminals, we need a fallback key.

**Files:**
- Modify: `frontend/src/lib/components/TerminalTabs.svelte:59`
- Modify: `frontend/src/lib/components/TerminalView.svelte:11`

**Step 1: Pass sessionId from tab to TerminalView**

In `TerminalTabs.svelte`, line 59, change:

```svelte
<TerminalView tabId={tab.id} active={$activeTabId === tab.id} />
```

to:

```svelte
<TerminalView tabId={tab.id} active={$activeTabId === tab.id} historyId={tab.sessionId || tab.id} />
```

**Step 2: Accept the prop in TerminalView**

In `TerminalView.svelte`, after line 11 (`export let active`), add:

```ts
export let historyId: string = "";
```

**Step 3: Commit**

```bash
git add frontend/src/lib/components/TerminalTabs.svelte frontend/src/lib/components/TerminalView.svelte
git commit -m "feat: pass historyId prop to TerminalView"
```

---

### Task 3: Frontend — input buffer tracking

Track what the user has typed on the current line by intercepting `terminal.onData()`.

**Files:**
- Modify: `frontend/src/lib/components/TerminalView.svelte`

**Step 1: Add history state and input buffer tracking**

At the top of the `<script>` block (after the imports, around line 9), add the import and state variables:

```ts
import { SaveCommandToHistory, GetCommandHistory } from "../../../wailsjs/go/main/App";

// Autosuggestion state
let commandHistory: string[] = [];
let currentInput = "";
let currentSuggestion = "";
```

**Step 2: Load history on mount**

Inside `onMount()`, right after `setTabConnected(tabId, true)` (line 108), add:

```ts
// Load command history for autosuggestions
if (historyId) {
  GetCommandHistory(historyId).then((h) => {
    commandHistory = h || [];
  });
}
```

**Step 3: Replace the terminal.onData handler**

Replace the existing `terminal.onData` handler (lines 58-65) with the enhanced version that tracks input:

```ts
terminal.onData((data) => {
  // Send raw data to backend (unchanged)
  const bytes = new TextEncoder().encode(data);
  let binary = "";
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  SendInput(tabId, btoa(binary)).catch(() => {});

  // Track input buffer for autosuggestions
  if (data === "\r" || data === "\n") {
    // Enter: save command to history, reset
    const cmd = currentInput.trim();
    if (cmd && historyId) {
      SaveCommandToHistory(historyId, cmd).catch(() => {});
      // Update local cache: remove dupe, prepend
      commandHistory = commandHistory.filter((h) => h !== cmd);
      commandHistory.unshift(cmd);
    }
    currentInput = "";
    currentSuggestion = "";
  } else if (data === "\x7f" || data === "\b") {
    // Backspace
    currentInput = currentInput.slice(0, -1);
    updateSuggestion();
  } else if (data === "\x03") {
    // Ctrl+C
    currentInput = "";
    currentSuggestion = "";
  } else if (data === "\x15") {
    // Ctrl+U: clear line
    currentInput = "";
    currentSuggestion = "";
  } else if (data === "\x17") {
    // Ctrl+W: delete last word
    currentInput = currentInput.trimEnd().replace(/\S+$/, "").trimEnd();
    updateSuggestion();
  } else if (data.length > 1 && (data.charCodeAt(0) === 27 || data.length > 4)) {
    // Escape sequence (arrows, etc.) or paste (multi-char)
    // For escape sequences: reset buffer (cursor moved, lost track)
    // For paste: just append but skip suggestion update
    if (data.charCodeAt(0) === 27) {
      currentInput = "";
      currentSuggestion = "";
    } else {
      // Paste: append text but don't suggest
      currentInput += data;
      currentSuggestion = "";
    }
  } else if (data >= " ") {
    // Printable character
    currentInput += data;
    updateSuggestion();
  } else {
    // Other control chars: reset
    currentInput = "";
    currentSuggestion = "";
  }
});
```

**Step 4: Add the updateSuggestion function**

After the state variables (around line 16), add:

```ts
function updateSuggestion() {
  if (!currentInput) {
    currentSuggestion = "";
    return;
  }
  const match = commandHistory.find((h) =>
    h.startsWith(currentInput) && h !== currentInput
  );
  currentSuggestion = match ? match.slice(currentInput.length) : "";
}
```

**Step 5: Commit**

```bash
git add frontend/src/lib/components/TerminalView.svelte
git commit -m "feat: track terminal input buffer for autosuggestions"
```

---

### Task 4: Frontend — ghost text overlay rendering

Render the suggestion as grayed-out text positioned at the cursor.

**Files:**
- Modify: `frontend/src/lib/components/TerminalView.svelte`

**Step 1: Add the ghost text overlay div**

Replace the template section (line 145):

```svelte
<div class="terminal-container" class:visible={active} bind:this={container}></div>
```

with:

```svelte
<div class="terminal-wrapper" class:visible={active}>
  <div class="terminal-container" bind:this={container}></div>
  {#if currentSuggestion}
    <div
      class="ghost-suggestion"
      style="left: {ghostLeft}px; top: {ghostTop}px; font-size: {$appConfig.fontSize}px;"
    >
      {currentSuggestion}
    </div>
  {/if}
</div>
```

**Step 2: Add ghost position computation**

After the `updateSuggestion` function, add:

```ts
let ghostLeft = 0;
let ghostTop = 0;

function updateGhostPosition() {
  if (!terminal || !currentSuggestion) return;
  const cursorX = terminal.buffer.active.cursorX;
  const cursorY = terminal.buffer.active.cursorY;
  const cellWidth = terminal.element
    ? terminal.element.querySelector(".xterm-char-measure-element")?.getBoundingClientRect().width || ($appConfig.fontSize * 0.6)
    : $appConfig.fontSize * 0.6;
  const cellHeight = terminal.element
    ? (terminal.element.querySelector(".xterm-screen")?.getBoundingClientRect().height || 0) / terminal.rows
    : $appConfig.fontSize * 1.2;
  // Account for 4px padding on .xterm
  ghostLeft = cursorX * cellWidth + 4;
  ghostTop = cursorY * cellHeight + 4;
}
```

**Step 3: Call updateGhostPosition when suggestion changes**

Add a reactive statement after the existing reactive blocks (around line 141):

```ts
$: if (currentSuggestion && terminal) {
  // Use tick to ensure DOM is updated
  setTimeout(updateGhostPosition, 0);
}
```

**Step 4: Add ghost text styles**

In the `<style>` section, replace the existing `.terminal-container` styles with:

```css
.terminal-wrapper {
  width: 100%;
  height: 100%;
  display: none;
  position: relative;
}

.terminal-wrapper.visible {
  display: block;
}

.terminal-container {
  width: 100%;
  height: 100%;
}

.terminal-container :global(.xterm) {
  height: 100%;
  padding: 4px;
}

.ghost-suggestion {
  position: absolute;
  color: #555;
  pointer-events: none;
  white-space: pre;
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
  z-index: 1;
  opacity: 0.5;
}
```

Keep the existing scrollbar styles unchanged.

**Step 5: Commit**

```bash
git add frontend/src/lib/components/TerminalView.svelte
git commit -m "feat: render ghost-text suggestion overlay"
```

---

### Task 5: Frontend — accept suggestion with Right Arrow

Intercept Right Arrow and End key to accept the suggestion before it reaches the shell.

**Files:**
- Modify: `frontend/src/lib/components/TerminalView.svelte`

**Step 1: Add custom key event handler**

Inside `onMount()`, after `terminal.open(container)` and before `fitAddon.fit()` (around line 54-55), add:

```ts
// Intercept Right Arrow / End to accept autosuggestion
terminal.attachCustomKeyEventHandler((e: KeyboardEvent) => {
  if (!currentSuggestion) return true; // no suggestion, pass through

  if (e.type !== "keydown") return true;

  if (e.key === "ArrowRight" || e.key === "End") {
    e.preventDefault();
    // Send the suggestion text to backend
    const bytes = new TextEncoder().encode(currentSuggestion);
    let binary = "";
    for (let i = 0; i < bytes.length; i++) {
      binary += String.fromCharCode(bytes[i]);
    }
    SendInput(tabId, btoa(binary)).catch(() => {});
    currentInput += currentSuggestion;
    currentSuggestion = "";
    return false; // prevent xterm from handling it
  }

  return true; // let everything else pass through
});
```

**Step 2: Commit**

```bash
git add frontend/src/lib/components/TerminalView.svelte
git commit -m "feat: accept autosuggestion with Right Arrow or End key"
```

---

### Task 6: Integration testing and polish

**Files:**
- All modified files

**Step 1: Build the app**

Run: `wails build` or `wails dev`

**Step 2: Manual test checklist**

- [ ] Open a local terminal, type commands, verify they're saved to history
- [ ] Type a prefix of a previous command — ghost text appears
- [ ] Press Right Arrow — suggestion is accepted and sent to shell
- [ ] Press any other key — suggestion disappears, normal typing continues
- [ ] Backspace updates the suggestion
- [ ] Ctrl+C clears the suggestion
- [ ] Tab key passes through to shell completion (not intercepted)
- [ ] Open an SSH session to a remote host, verify separate history
- [ ] Reconnect to same session — previous history loads
- [ ] Quick-connect session uses tab ID as history key

**Step 3: Fix ghost text positioning if needed**

The cell width calculation may need adjustment. If the ghost text is misaligned:
- Try using `terminal._core._renderService.dimensions.css.cell.width` (proposed API, needs `allowProposedApi: true` which is already set)
- Or measure a rendered character's width

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat: terminal autosuggestions - final polish"
```
