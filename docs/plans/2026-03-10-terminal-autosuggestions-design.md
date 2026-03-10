# Terminal Autosuggestions Design

Per-session inline ghost-text suggestions (like zsh-autosuggestions) for SSH and local terminal sessions.

## History Storage

- **Key**: Session ID from existing session config
- **Storage**: Go backend persists `~/.terminalis/history/<sessionID>.json`
- **Structure**: Ordered list of commands, most recent first, deduped
- **Max entries**: 1000 per session
- **Quick-connect / local tabs**: Keyed by hash of connection string or `"local"`

## Go Backend API

Two new methods on `App`:

- `SaveCommandToHistory(sessionID string, command string)` — appends, dedupes
- `GetCommandHistory(sessionID string) []string` — returns full history

Matching stays in frontend for zero-latency suggestions.

## Frontend Input Buffer Tracking

Maintain `currentInput` string in `TerminalView.svelte`:

- `\r` / `\n` (Enter): Save to history, reset buffer
- `\x7f` (Backspace): Remove last char
- `\x03` (Ctrl+C): Reset buffer
- `\x15` (Ctrl+U): Clear buffer
- `\x17` (Ctrl+W): Remove last word
- Printable chars: Append
- Escape sequences (arrows, etc.): Reset buffer (conservative — lose track of cursor)

## Ghost Text Rendering

- Overlay div positioned after cursor using `buffer.active.cursorX/Y`
- Prefix match against session history, render remaining suffix in dim gray
- Update/hide on every input change

## Accepting Suggestions

- **Right Arrow** or **End**: Accept full suggestion, send remaining chars via `SendInput`
- **Any other key**: Dismiss, continue normal typing
- Use `terminal.attachCustomKeyEventHandler()` to intercept before `onData`

## Edge Cases

- No history: invisible, no suggestions
- Tab key: passes through to remote shell completion
- Multi-line: only track last line
- Paste: detect multi-char input, skip suggestion
- Don't save empty/whitespace-only commands
