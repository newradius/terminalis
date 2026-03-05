<script lang="ts">
  import { showSessionForm, editingSession, selectedFolderId } from "../stores/sessions";
  import type { Session, TerminalInfo } from "../types";
  import { SaveSession, GetSession, SelectPrivateKeyFile, GetSessionTree, GetAvailableTerminals } from "../../../wailsjs/go/main/App";
  import { sessionTree } from "../stores/sessions";
  import { onMount } from "svelte";
  import type { Folder } from "../types";

  let name = "";
  let host = "";
  let port = 22;
  let username = "root";
  let authMethod: "password" | "privatekey" = "password";
  let password = "";
  let privateKeyPath = "";
  let passphrase = "";
  let folderId = "";
  let color = "#45a049";
  let compression = false;
  let keepAlive = 0;
  let terminalType: string = "embedded";
  let systemTerminal: string = "";
  let availableTerminals: TerminalInfo[] = [];
  let loading = false;

  const colors = ["#45a049", "#2196F3", "#FF9800", "#E91E63", "#9C27B0", "#00BCD4", "#FF5722", "#607D8B"];

  onMount(async () => {
    folderId = $selectedFolderId || "";

    // Load available system terminals
    try {
      availableTerminals = await GetAvailableTerminals() || [];
    } catch { availableTerminals = []; }

    if ($editingSession) {
      const sess = await GetSession($editingSession);
      if (sess) {
        name = sess.name;
        host = sess.host;
        port = sess.port;
        username = sess.username;
        authMethod = sess.authMethod;
        password = sess.password || "";
        privateKeyPath = sess.privateKeyPath || "";
        passphrase = sess.passphrase || "";
        folderId = sess.folderId;
        color = sess.color || "#45a049";
        compression = sess.compression;
        keepAlive = sess.keepAlive;
        terminalType = sess.terminalType || "embedded";
        systemTerminal = sess.systemTerminal || "";
      }
    }
  });

  async function selectKey() {
    const path = await SelectPrivateKeyFile();
    if (path) {
      privateKeyPath = path;
    }
  }

  async function save() {
    if (!name.trim() || !host.trim()) return;
    loading = true;

    const sess: any = {
      id: $editingSession || "",
      name: name.trim(),
      host: host.trim(),
      port,
      username: username.trim(),
      authMethod,
      password: authMethod === "password" ? password : "",
      privateKeyPath: authMethod === "privatekey" ? privateKeyPath : "",
      passphrase: authMethod === "privatekey" ? passphrase : "",
      folderId,
      color,
      compression,
      keepAlive,
      terminalType,
      systemTerminal: terminalType === "system" ? systemTerminal : "",
      createdAt: 0,
      updatedAt: 0,
    };

    try {
      await SaveSession(sess);
      const tree = await GetSessionTree();
      sessionTree.set(tree || []);
      close();
    } finally {
      loading = false;
    }
  }

  function close() {
    showSessionForm.set(false);
    editingSession.set(null);
    selectedFolderId.set("");
  }
</script>

<div class="overlay" on:click|self={close}>
  <div class="modal">
    <div class="modal-header">
      <h2>{$editingSession ? "Edit Session" : "New Session"}</h2>
      <button class="close-btn" on:click={close}>&times;</button>
    </div>

    <div class="modal-body">
      <div class="form-row">
        <label>
          <span>Name</span>
          <input type="text" bind:value={name} placeholder="My Server" autofocus />
        </label>
      </div>

      <div class="form-row three-col">
        <label class="flex-2">
          <span>Host</span>
          <input type="text" bind:value={host} placeholder="192.168.1.1" />
        </label>
        <label>
          <span>Port</span>
          <input type="number" bind:value={port} min="1" max="65535" />
        </label>
        <label>
          <span>Username</span>
          <input type="text" bind:value={username} placeholder="root" />
        </label>
      </div>

      <div class="form-row">
        <label>
          <span>Authentication</span>
          <div class="auth-tabs">
            <button class:active={authMethod === "password"} on:click={() => (authMethod = "password")}>
              Password
            </button>
            <button class:active={authMethod === "privatekey"} on:click={() => (authMethod = "privatekey")}>
              Private Key
            </button>
          </div>
        </label>
      </div>

      {#if authMethod === "password"}
        <div class="form-row">
          <label>
            <span>Password</span>
            <input type="password" bind:value={password} placeholder="Enter password" />
          </label>
        </div>
      {:else}
        <div class="form-row">
          <label>
            <span>Private Key</span>
            <div class="file-picker">
              <input type="text" bind:value={privateKeyPath} placeholder="Select private key file" readonly />
              <button on:click={selectKey}>Browse</button>
            </div>
          </label>
        </div>
        <div class="form-row">
          <label>
            <span>Passphrase (optional)</span>
            <input type="password" bind:value={passphrase} placeholder="Key passphrase" />
          </label>
        </div>
      {/if}

      <div class="form-row two-col">
        <label>
          <span>Keep Alive (seconds)</span>
          <input type="number" bind:value={keepAlive} min="0" placeholder="0 = disabled" />
        </label>
        <label class="checkbox-label">
          <input type="checkbox" bind:checked={compression} />
          <span>Compression</span>
        </label>
      </div>

      <div class="form-row">
        <label>
          <span>Terminal</span>
          <div class="auth-tabs">
            <button class:active={terminalType === "embedded"} on:click={() => (terminalType = "embedded")}>
              Embedded
            </button>
            <button class:active={terminalType === "system"} on:click={() => (terminalType = "system")}>
              System Terminal
            </button>
          </div>
        </label>
      </div>

      {#if terminalType === "system"}
        <div class="form-row">
          <label>
            <span>System Terminal</span>
            <select bind:value={systemTerminal}>
              <option value="">Auto-detect</option>
              {#each availableTerminals as t}
                <option value={t.path}>{t.name}</option>
              {/each}
            </select>
          </label>
        </div>
      {/if}

      <div class="form-row">
        <label>
          <span>Color</span>
          <div class="color-picker">
            {#each colors as c}
              <button
                class="color-swatch"
                class:selected={color === c}
                style:background={c}
                on:click={() => (color = c)}
              ></button>
            {/each}
          </div>
        </label>
      </div>
    </div>

    <div class="modal-footer">
      <button class="btn-secondary" on:click={close}>Cancel</button>
      <button class="btn-primary" on:click={save} disabled={loading || !name.trim() || !host.trim()}>
        {loading ? "Saving..." : "Save"}
      </button>
    </div>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
    backdrop-filter: blur(2px);
  }

  .modal {
    background: #1e1f33;
    border: 1px solid #2a2b3d;
    border-radius: 12px;
    width: 520px;
    max-height: 90vh;
    overflow-y: auto;
    overflow-x: hidden;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 20px 24px 16px;
    border-bottom: 1px solid #2a2b3d;
  }

  .modal-header h2 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: #e0e0e0;
  }

  .close-btn {
    background: none;
    border: none;
    color: #888;
    font-size: 24px;
    cursor: pointer;
    padding: 0 4px;
    line-height: 1;
  }

  .close-btn:hover {
    color: #fff;
  }

  .modal-body {
    padding: 20px 24px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .form-row {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .form-row.two-col, .form-row.three-col {
    flex-direction: row;
    gap: 12px;
  }

  .form-row.two-col > label,
  .form-row.three-col > label {
    flex: 1;
  }

  .form-row.three-col > label.flex-2 {
    flex: 2;
  }

  label {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  label > span {
    font-size: 12px;
    color: #999;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }

  input[type="text"],
  input[type="password"],
  input[type="number"] {
    background: #15162a;
    border: 1px solid #2a2b3d;
    border-radius: 6px;
    padding: 8px 12px;
    color: #e0e0e0;
    font-size: 14px;
    outline: none;
    transition: border-color 0.2s;
  }

  input:focus, select:focus {
    border-color: #4a6cf7;
  }

  select {
    background: #15162a;
    border: 1px solid #2a2b3d;
    border-radius: 6px;
    padding: 8px 12px;
    color: #e0e0e0;
    font-size: 14px;
    outline: none;
    transition: border-color 0.2s;
    cursor: pointer;
  }

  .auth-tabs {
    display: flex;
    gap: 0;
    border-radius: 6px;
    overflow: hidden;
    border: 1px solid #2a2b3d;
  }

  .auth-tabs button {
    flex: 1;
    padding: 8px 16px;
    background: #15162a;
    border: none;
    color: #888;
    font-size: 13px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .auth-tabs button.active {
    background: #4a6cf7;
    color: white;
  }

  .file-picker {
    display: flex;
    gap: 8px;
  }

  .file-picker input {
    flex: 1;
  }

  .file-picker button {
    background: #2a2b3d;
    border: none;
    border-radius: 6px;
    color: #ccc;
    padding: 8px 16px;
    cursor: pointer;
    font-size: 13px;
    white-space: nowrap;
  }

  .file-picker button:hover {
    background: #3a3b4d;
  }

  .checkbox-label {
    flex-direction: row !important;
    align-items: center;
    gap: 8px !important;
    padding-top: 22px;
  }

  .checkbox-label input[type="checkbox"] {
    width: 16px;
    height: 16px;
    accent-color: #4a6cf7;
  }

  .checkbox-label span {
    text-transform: none !important;
    font-size: 14px !important;
    color: #ccc !important;
  }

  .color-picker {
    display: flex;
    gap: 8px;
  }

  .color-swatch {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    border: 2px solid transparent;
    cursor: pointer;
    transition: all 0.15s;
  }

  .color-swatch.selected {
    border-color: white;
    transform: scale(1.15);
  }

  .color-swatch:hover {
    transform: scale(1.1);
  }

  .modal-footer {
    padding: 16px 24px 20px;
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    border-top: 1px solid #2a2b3d;
  }

  .btn-secondary {
    background: #2a2b3d;
    border: none;
    border-radius: 6px;
    color: #ccc;
    padding: 10px 20px;
    font-size: 14px;
    cursor: pointer;
  }

  .btn-secondary:hover {
    background: #3a3b4d;
  }

  .btn-primary {
    background: #4a6cf7;
    border: none;
    border-radius: 6px;
    color: white;
    padding: 10px 24px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
  }

  .btn-primary:hover {
    background: #5b7bf8;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
