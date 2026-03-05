<script lang="ts">
  import Sidebar from "./lib/components/Sidebar.svelte";
  import TerminalTabs from "./lib/components/TerminalTabs.svelte";
  import QuickConnect from "./lib/components/QuickConnect.svelte";
  import SessionForm from "./lib/components/SessionForm.svelte";
  import FolderForm from "./lib/components/FolderForm.svelte";
  import HostKeyDialog from "./lib/components/HostKeyDialog.svelte";
  import PasswordDialog from "./lib/components/PasswordDialog.svelte";
  import SettingsForm from "./lib/components/SettingsForm.svelte";
  import { showSessionForm, showFolderForm, showSettings } from "./lib/stores/sessions";
  import { loadConfig } from "./lib/stores/config";
  import { createTab, closeTab, setTabConnected } from "./lib/stores/terminals";
  import { ConnectSession, QuickConnect as QuickConnectApi, OpenLocalShell } from "../wailsjs/go/main/App";
  import { EventsOn } from "../wailsjs/runtime/runtime";
  import type { HostKeyInfo, Session } from "./lib/types";
  import { onMount } from "svelte";

  let hostKeyInfo: HostKeyInfo | null = null;
  let passwordPrompt: { tabId: string; session: Session } | null = null;
  let connectionError: string | null = null;

  function showError(msg: string) {
    connectionError = msg;
    setTimeout(() => { connectionError = null; }, 5000);
  }

  onMount(() => {
    loadConfig();

    EventsOn("ssh:hostkey", (info: HostKeyInfo) => {
      hostKeyInfo = info;
    });

    window.addEventListener("connect-session", ((e: CustomEvent) => {
      const { tabId, session } = e.detail as { tabId: string; session: Session };

      // If password auth and no saved password, prompt
      if (session.authMethod === "password" && !session.password) {
        passwordPrompt = { tabId, session };
        return;
      }

      doConnect(tabId, session.id, session.password || "");
    }) as EventListener);

    window.addEventListener("open-local-terminal", ((e: CustomEvent) => {
      const { shell } = e.detail || {};
      const tabId = createTab(shell ? shell.split(/[/\\]/).pop() : "Terminal");
      setTabConnected(tabId, true);
      OpenLocalShell({
        tabId,
        shell: shell || "",
        cols: 120,
        rows: 40,
      }).catch((err: any) => {
        showError("Failed to open terminal: " + err);
        setTabConnected(tabId, false);
      });
    }) as EventListener);

    window.addEventListener("quick-connect", ((e: CustomEvent) => {
      const { tabId, connString, password } = e.detail;
      QuickConnectApi({
        tabId,
        connString,
        password: password || "",
        cols: 120,
        rows: 40,
      }).catch((err: any) => {
        showError("Connection failed: " + err);
        setTabConnected(tabId, false);
      });
    }) as EventListener);

    window.addEventListener("connection-error", ((e: CustomEvent) => {
      showError(e.detail?.message || "Connection error");
    }) as EventListener);
  });

  function doConnect(tabId: string, sessionId: string, password: string) {
    ConnectSession({
      tabId,
      sessionId,
      password,
      cols: 120,
      rows: 40,
    }).catch((err: any) => {
      showError("Connection failed: " + err);
      setTabConnected(tabId, false);
    });
  }

  function handlePasswordSubmit(password: string) {
    if (!passwordPrompt) return;
    doConnect(passwordPrompt.tabId, passwordPrompt.session.id, password);
    passwordPrompt = null;
  }

  function handlePasswordCancel() {
    if (passwordPrompt) {
      closeTab(passwordPrompt.tabId);
    }
    passwordPrompt = null;
  }
</script>

<main class="app">
  <Sidebar />
  <div class="main-area">
    <QuickConnect />
    <TerminalTabs />
  </div>
</main>

{#if $showSessionForm}
  <SessionForm />
{/if}

{#if $showFolderForm}
  <FolderForm />
{/if}

{#if $showSettings}
  <SettingsForm />
{/if}

{#if hostKeyInfo}
  <HostKeyDialog
    info={hostKeyInfo}
    onClose={() => (hostKeyInfo = null)}
  />
{/if}

{#if passwordPrompt}
  <PasswordDialog
    host="{passwordPrompt.session.username}@{passwordPrompt.session.host}"
    onSubmit={handlePasswordSubmit}
    onCancel={handlePasswordCancel}
  />
{/if}

{#if connectionError}
  <div class="toast-error">
    <span>{connectionError}</span>
    <button on:click={() => (connectionError = null)}>&times;</button>
  </div>
{/if}

<style>
  :global(*) {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  :global(body) {
    background: #14142a;
    color: #e0e0e0;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    overflow: hidden;
  }

  :global(html, body, #app) {
    height: 100%;
    width: 100%;
  }

  .app {
    display: flex;
    height: 100vh;
    width: 100vw;
    overflow: hidden;
  }

  .main-area {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  :global(.toast-error) {
    position: fixed;
    bottom: 20px;
    right: 20px;
    background: #5c2020;
    border: 1px solid #ff6b6b;
    color: #ff6b6b;
    padding: 12px 16px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    gap: 12px;
    z-index: 300;
    font-size: 13px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
    animation: slideIn 0.3s ease;
  }

  @keyframes slideIn {
    from {
      transform: translateY(20px);
      opacity: 0;
    }
    to {
      transform: translateY(0);
      opacity: 1;
    }
  }

  :global(.toast-error button) {
    background: none;
    border: none;
    color: #ff6b6b;
    font-size: 18px;
    cursor: pointer;
    padding: 0;
    line-height: 1;
  }
</style>
