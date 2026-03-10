<script lang="ts">
  import SessionTree from "./SessionTree.svelte";
  import ScpPanel from "./ScpPanel.svelte";
  import { showSessionForm, showFolderForm, selectedFolderId, sessionTree, showSettings } from "../stores/sessions";
  import { tabs, activeTabId } from "../stores/terminals";
  import { MoveSession, MoveFolder, GetSessionTree, ExportSessions, ImportSessions, IsSSHSession } from "../../../wailsjs/go/main/App";
  import { EventsOn } from "../../../wailsjs/runtime/runtime";
  import { onMount } from "svelte";

  let dropTargetRoot = false;
  let searchQuery = "";
  let searchInput: HTMLInputElement;
  let sidebarWidth = 260;
  let isResizing = false;
  let showScp = false;
  let isCurrentSSH = false;
  let sshConnectedTick = 0;

  onMount(() => {
    window.addEventListener("focus-search", () => {
      searchInput?.focus();
    });

    EventsOn("ssh:connected", () => {
      sshConnectedTick++;
    });
  });

  // Check if current tab is SSH — re-check on tab change and ssh:connected
  $: {
    const currentId = $activeTabId;
    const currentTabs = $tabs;
    const _tick = sshConnectedTick;
    const activeTab = currentTabs.find(t => t.id === currentId);
    if (currentId && activeTab?.connected) {
      IsSSHSession(currentId).then((v) => {
        isCurrentSSH = v;
        if (!v) showScp = false;
      });
    } else {
      isCurrentSSH = false;
      showScp = false;
    }
  }

  function addSession() {
    selectedFolderId.set("");
    showSessionForm.set(true);
  }

  function addFolder() {
    showFolderForm.set(true);
  }

  function openLocalTerminal() {
    window.dispatchEvent(new CustomEvent("open-local-terminal", { detail: {} }));
  }

  async function handleExport() {
    try {
      const path = await ExportSessions();
      if (path) {
        window.dispatchEvent(new CustomEvent("toast-success", {
          detail: { message: "Sessions exported successfully" },
        }));
      }
    } catch (err: any) {
      window.dispatchEvent(new CustomEvent("connection-error", {
        detail: { message: "Export failed: " + err },
      }));
    }
  }

  async function handleImport() {
    try {
      const count = await ImportSessions();
      if (count > 0) {
        const tree = await GetSessionTree();
        sessionTree.set(tree || []);
        window.dispatchEvent(new CustomEvent("toast-success", {
          detail: { message: `Imported ${count} items successfully` },
        }));
      }
    } catch (err: any) {
      window.dispatchEvent(new CustomEvent("connection-error", {
        detail: { message: "Import failed: " + err },
      }));
    }
  }

  function handleRootDragOver(e: DragEvent) {
    const target = e.target as HTMLElement;
    if (target.closest(".tree-item")) return;
    e.preventDefault();
    if (e.dataTransfer) e.dataTransfer.dropEffect = "move";
    dropTargetRoot = true;
  }

  function handleRootDragLeave(e: DragEvent) {
    const related = e.relatedTarget as HTMLElement | null;
    const container = e.currentTarget as HTMLElement;
    if (related && container.contains(related)) return;
    dropTargetRoot = false;
  }

  async function handleRootDrop(e: DragEvent) {
    e.preventDefault();
    dropTargetRoot = false;
    if (!e.dataTransfer) return;

    let data: { id: string; type: string };
    try {
      data = JSON.parse(e.dataTransfer.getData("text/plain"));
    } catch { return; }

    try {
      if (data.type === "session") {
        await MoveSession(data.id, "");
      } else if (data.type === "folder") {
        await MoveFolder(data.id, "");
      }
      const tree = await GetSessionTree();
      sessionTree.set(tree || []);
    } catch (err: any) {
      console.error("Move to root failed:", err);
    }
  }

  // Resize handlers
  function startResize(e: MouseEvent) {
    isResizing = true;
    e.preventDefault();
    const onMove = (ev: MouseEvent) => {
      sidebarWidth = Math.max(180, Math.min(500, ev.clientX));
    };
    const onUp = () => {
      isResizing = false;
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    };
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
  }

  // Count total sessions
  function countSessions(nodes: any[]): number {
    let count = 0;
    for (const n of nodes) {
      if (n.type === "session") count++;
      if (n.children) count += countSessions(n.children);
    }
    return count;
  }

  $: totalSessions = countSessions($sessionTree);
</script>

<aside class="sidebar" style="width: {sidebarWidth}px; min-width: {sidebarWidth}px">
  <div class="sidebar-header">
    {#if showScp}
      <span class="sidebar-title">SCP Files</span>
    {:else}
      <span class="sidebar-title">Sessions</span>
      <span class="session-count">{totalSessions}</span>
    {/if}
    <div class="sidebar-actions">
      {#if isCurrentSSH}
        <button
          class="icon-btn scp-toggle-btn"
          class:active={showScp}
          on:click={() => showScp = !showScp}
          title={showScp ? "Show sessions" : "Show SCP files"}
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          </svg>
        </button>
      {/if}
      {#if !showScp}
        <button class="icon-btn" on:click={openLocalTerminal} title="Open Terminal (Ctrl+T)">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="4 17 10 11 4 5"/>
            <line x1="12" y1="19" x2="20" y2="19"/>
          </svg>
        </button>
        <button class="icon-btn" on:click={addFolder} title="New Folder">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
            <line x1="12" y1="11" x2="12" y2="17"/>
            <line x1="9" y1="14" x2="15" y2="14"/>
          </svg>
        </button>
        <button class="icon-btn" on:click={addSession} title="New Session">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19"/>
            <line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
        </button>
      {/if}
    </div>
  </div>
  {#if showScp && $activeTabId}
    <div class="sidebar-content scp-view">
      <ScpPanel tabId={$activeTabId} active={true} />
    </div>
  {:else}
    <div class="search-bar">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#666" stroke-width="2">
        <circle cx="11" cy="11" r="8"/>
        <line x1="21" y1="21" x2="16.65" y2="16.65"/>
      </svg>
      <input
        type="text"
        bind:value={searchQuery}
        bind:this={searchInput}
        placeholder="Search sessions... (Ctrl+K)"
      />
      {#if searchQuery}
        <button class="clear-search" on:click={() => searchQuery = ""}>
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      {/if}
    </div>
    <div
      class="sidebar-content"
      class:drop-target-root={dropTargetRoot}
      on:dragover={handleRootDragOver}
      on:dragleave={handleRootDragLeave}
      on:drop={handleRootDrop}
    >
      <SessionTree {searchQuery} />
    </div>
  {/if}
  {#if !showScp}
    <div class="sidebar-footer">
      <button class="icon-btn" on:click={handleImport} title="Import Sessions">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
          <polyline points="7 10 12 15 17 10"/>
          <line x1="12" y1="15" x2="12" y2="3"/>
        </svg>
      </button>
      <button class="icon-btn" on:click={handleExport} title="Export Sessions">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
          <polyline points="17 8 12 3 7 8"/>
          <line x1="12" y1="3" x2="12" y2="15"/>
        </svg>
      </button>
      <div class="spacer"></div>
      <button class="icon-btn" on:click={() => showSettings.set(true)} title="Settings">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="3"/>
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
        </svg>
      </button>
    </div>
  {/if}
  <div class="resize-handle" on:mousedown={startResize}></div>
</aside>

<style>
  .sidebar {
    width: 260px;
    background: #1a1b2e;
    border-right: 1px solid #2a2b3d;
    display: flex;
    flex-direction: column;
    height: 100%;
    position: relative;
    flex-shrink: 0;
  }

  .sidebar-header {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 12px 16px;
    border-bottom: 1px solid #2a2b3d;
  }

  .sidebar-title {
    font-size: 13px;
    font-weight: 600;
    color: #e0e0e0;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .session-count {
    font-size: 11px;
    color: #888;
    background: #2a2b3d;
    padding: 1px 6px;
    border-radius: 8px;
    font-variant-numeric: tabular-nums;
  }

  .sidebar-actions {
    display: flex;
    gap: 4px;
    margin-left: auto;
  }

  .icon-btn {
    background: none;
    border: none;
    color: #888;
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .icon-btn:hover {
    color: #fff;
    background: #2a2b3d;
  }

  .search-bar {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 12px;
    margin: 6px 8px;
    background: #15162a;
    border: 1px solid #2a2b3d;
    border-radius: 6px;
    transition: border-color 0.2s;
  }

  .search-bar:focus-within {
    border-color: #4a6cf7;
  }

  .search-bar input {
    flex: 1;
    background: none;
    border: none;
    color: #e0e0e0;
    font-size: 12px;
    outline: none;
    min-width: 0;
  }

  .search-bar input::placeholder {
    color: #555;
  }

  .clear-search {
    background: none;
    border: none;
    color: #666;
    cursor: pointer;
    padding: 2px;
    display: flex;
    border-radius: 3px;
  }

  .clear-search:hover {
    color: #ccc;
  }

  .sidebar-content {
    flex: 1;
    overflow-y: auto;
    padding: 8px 0;
    transition: background 0.15s, outline 0.15s;
  }

  .sidebar-content.drop-target-root {
    background: rgba(74, 108, 247, 0.05);
    outline: 1.5px dashed #4a6cf7;
    outline-offset: -3px;
  }

  .sidebar-content::-webkit-scrollbar {
    width: 6px;
  }

  .sidebar-content::-webkit-scrollbar-track {
    background: transparent;
  }

  .sidebar-content::-webkit-scrollbar-thumb {
    background: #3a3b4d;
    border-radius: 3px;
  }

  .sidebar-footer {
    padding: 8px 12px;
    border-top: 1px solid #2a2b3d;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .spacer {
    flex: 1;
  }

  .resize-handle {
    position: absolute;
    top: 0;
    right: -3px;
    width: 6px;
    height: 100%;
    cursor: col-resize;
    z-index: 10;
  }

  .resize-handle:hover {
    background: #4a6cf7;
    opacity: 0.5;
  }

  .scp-toggle-btn.active {
    color: #4a6cf7;
    background: rgba(74, 108, 247, 0.15);
  }

  .scp-view {
    padding: 0;
  }
</style>
