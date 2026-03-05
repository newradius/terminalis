<script lang="ts">
  import SessionTree from "./SessionTree.svelte";
  import { showSessionForm, showFolderForm, selectedFolderId, sessionTree, showSettings } from "../stores/sessions";
  import { MoveSession, MoveFolder, GetSessionTree } from "../../../wailsjs/go/main/App";

  let dropTargetRoot = false;

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

  function handleRootDragOver(e: DragEvent) {
    // Only act as drop target when dragging over empty space (not bubbled from tree items)
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
</script>

<aside class="sidebar">
  <div class="sidebar-header">
    <span class="sidebar-title">Sessions</span>
    <div class="sidebar-actions">
      <button class="icon-btn" on:click={openLocalTerminal} title="Open Terminal">
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
    </div>
  </div>
  <div
    class="sidebar-content"
    class:drop-target-root={dropTargetRoot}
    on:dragover={handleRootDragOver}
    on:dragleave={handleRootDragLeave}
    on:drop={handleRootDrop}
  >
    <SessionTree />
  </div>
  <div class="sidebar-footer">
    <button class="icon-btn" on:click={() => showSettings.set(true)} title="Settings">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="3"/>
        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
      </svg>
    </button>
  </div>
</aside>

<style>
  .sidebar {
    width: 260px;
    min-width: 200px;
    background: #1a1b2e;
    border-right: 1px solid #2a2b3d;
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .sidebar-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
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

  .sidebar-actions {
    display: flex;
    gap: 4px;
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
  }
</style>
