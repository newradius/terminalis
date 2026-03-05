package models

type AuthMethod string

const (
	AuthPassword   AuthMethod = "password"
	AuthPrivateKey AuthMethod = "privatekey"
)

type Session struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Host            string     `json:"host"`
	Port            int        `json:"port"`
	Username        string     `json:"username"`
	AuthMethod      AuthMethod `json:"authMethod"`
	Password        string     `json:"password,omitempty"`
	PrivateKeyPath  string     `json:"privateKeyPath,omitempty"`
	Passphrase      string     `json:"passphrase,omitempty"`
	FolderID        string     `json:"folderId"`
	Color           string     `json:"color,omitempty"`
	Compression     bool       `json:"compression"`
	KeepAlive       int        `json:"keepAlive"` // seconds, 0 = disabled
	TerminalType    string     `json:"terminalType,omitempty"`    // "embedded" or "system", empty = embedded
	SystemTerminal  string     `json:"systemTerminal,omitempty"`  // path to system terminal, empty = auto-detect
	CreatedAt       int64      `json:"createdAt"`
	UpdatedAt       int64      `json:"updatedAt"`
}

type Folder struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parentId,omitempty"` // empty = root
	Color    string `json:"color,omitempty"`
	Expanded bool   `json:"expanded"`
}

type SessionData struct {
	Sessions []Session `json:"sessions"`
	Folders  []Folder  `json:"folders"`
}

// TreeNode is used by the frontend for rendering
type TreeNode struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Type     string     `json:"type"` // "folder" or "session"
	Color    string     `json:"color,omitempty"`
	Expanded bool       `json:"expanded,omitempty"`
	Children []TreeNode `json:"children,omitempty"`
	Session  *Session   `json:"session,omitempty"`
}
