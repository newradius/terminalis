export interface Session {
  id: string;
  name: string;
  host: string;
  port: number;
  username: string;
  authMethod: "password" | "privatekey";
  password?: string;
  privateKeyPath?: string;
  passphrase?: string;
  folderId: string;
  color?: string;
  compression: boolean;
  keepAlive: number;
  terminalType?: string;     // "embedded" or "system", empty = embedded
  systemTerminal?: string;   // path to system terminal, empty = auto-detect
  createdAt: number;
  updatedAt: number;
}

export interface TerminalInfo {
  name: string;
  path: string;
}

export interface Folder {
  id: string;
  name: string;
  parentId?: string;
  color?: string;
  expanded: boolean;
}

export interface TreeNode {
  id: string;
  name: string;
  type: "folder" | "session";
  color?: string;
  expanded?: boolean;
  children?: TreeNode[];
  session?: Session;
}

export interface Tab {
  id: string;
  title: string;
  sessionId?: string;
  connected: boolean;
}

export interface HostKeyInfo {
  tabId: string;
  host: string;
  fingerprint: string;
  keyType: string;
  isNew: boolean;
  isMismatch: boolean;
}

export interface ConnectRequest {
  tabId: string;
  sessionId: string;
  password?: string;
  cols: number;
  rows: number;
}

export interface QuickConnectRequest {
  tabId: string;
  connString: string;
  password?: string;
  cols: number;
  rows: number;
}
