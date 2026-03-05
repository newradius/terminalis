# Terminalis - SSH Session Manager

## Product Requirements Document

### Overview
Terminalis is a cross-platform SSH session manager built in Go with a modern web-based UI (Wails + Svelte + xterm.js). It provides organized session management with folder grouping, private key authentication, and an integrated terminal emulator.

### Tech Stack
- **Backend:** Go 1.24 + `golang.org/x/crypto/ssh`
- **Desktop Framework:** Wails v2 (native binaries for Windows/Linux/macOS)
- **Frontend:** Svelte + TypeScript
- **Terminal:** xterm.js + xterm-addon-fit + xterm-addon-web-links
- **Storage:** JSON file (~/.terminalis/sessions.json)

### Target Platforms
- Windows (amd64)
- Linux (amd64)
- macOS (amd64, arm64)

---

## Features

### 1. Session Management
- Create, edit, delete SSH sessions
- Each session stores: name, host, port, username, auth method, private key path, color label
- Sessions organized in folders/groups (tree view)
- Quick connect bar (type host directly without saving)
- Import/export sessions as JSON
- Drag-and-drop reordering (future)

### 2. SSH Connection
- Password authentication
- Private key authentication (RSA, Ed25519, ECDSA)
- Passphrase-protected private keys
- Configurable port (default 22)
- Host key verification (accept/reject on first connect, remember)
- Connection timeout setting
- Keep-alive interval
- Compression support

### 3. Terminal Emulator
- Full terminal emulation via xterm.js
- Multiple tabs for concurrent sessions
- Tab naming from session name
- Copy/paste support
- Scrollback buffer
- Font size adjustment
- Color theme (dark default)

### 4. UI Layout
- **Left sidebar:** Session tree with folders and sessions
- **Main area:** Terminal tabs
- **Top bar:** Quick connect, app controls
- **Session dialog:** Modal form for creating/editing sessions

### 5. Data Storage
- Sessions stored in `~/.terminalis/sessions.json`
- Known hosts in `~/.terminalis/known_hosts`
- App settings in `~/.terminalis/config.json`

---

## Architecture

```
terminalis/
├── main.go                    # Wails app entry
├── app.go                     # App struct (Wails bindings)
├── go.mod / go.sum
├── wails.json
├── internal/
│   ├── models/
│   │   └── session.go         # Session, Folder data models
│   ├── ssh/
│   │   ├── client.go          # SSH client wrapper
│   │   └── known_hosts.go     # Host key management
│   ├── storage/
│   │   └── store.go           # JSON file persistence
│   └── config/
│       └── config.go          # App configuration
├── frontend/
│   ├── src/
│   │   ├── App.svelte
│   │   ├── main.ts
│   │   ├── lib/
│   │   │   ├── components/
│   │   │   │   ├── Sidebar.svelte
│   │   │   │   ├── SessionTree.svelte
│   │   │   │   ├── SessionForm.svelte
│   │   │   │   ├── TerminalTabs.svelte
│   │   │   │   ├── TerminalView.svelte
│   │   │   │   ├── QuickConnect.svelte
│   │   │   │   └── HostKeyDialog.svelte
│   │   │   ├── stores/
│   │   │   │   ├── sessions.ts
│   │   │   │   └── terminals.ts
│   │   │   └── types.ts
│   │   └── assets/
│   ├── package.json
│   └── index.html
└── build/
    └── appicon.png
```

---

## Task Breakdown

### Phase 1: Project Setup
1. Initialize Wails project with Svelte template
2. Set up Go module and dependencies
3. Configure build for cross-platform

### Phase 2: Data Layer
4. Define session/folder data models (Go)
5. Implement JSON storage layer
6. Implement app configuration

### Phase 3: SSH Engine
7. Build SSH client wrapper with password auth
8. Add private key authentication support
9. Implement host key verification
10. Add session I/O streaming (stdin/stdout bridging)

### Phase 4: Backend API (Wails Bindings)
11. Session CRUD operations (Go → Frontend)
12. Folder CRUD operations
13. Connect/disconnect session management
14. Terminal I/O event bridge (Go ↔ Frontend)
15. Quick connect functionality

### Phase 5: Frontend - Layout & Session Management
16. App layout (sidebar + main area)
17. Session tree component with folders
18. Session create/edit form modal
19. Quick connect bar

### Phase 6: Frontend - Terminal
20. xterm.js terminal component
21. Tab management for multiple sessions
22. Terminal ↔ Go backend I/O wiring
23. Copy/paste, font sizing, scrollback

### Phase 7: Polish & Build
24. App icon and window configuration
25. Error handling and connection status indicators
26. Cross-platform build and test
