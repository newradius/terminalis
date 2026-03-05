<script lang="ts">
  import { sessionTree, showSessionForm, editingSession, showFolderForm, editingFolder, selectedFolderId } from "../stores/sessions";
  import { createTab } from "../stores/terminals";
  import type { TreeNode } from "../types";
  import { GetSessionTree, ToggleFolderExpanded, DeleteSession, DeleteFolder, MoveSession, MoveFolder, ConnectSessionExternal } from "../../../wailsjs/go/main/App";
  import { onMount } from "svelte";

  let contextMenu: { x: number; y: number; node: TreeNode } | null = null;

  // Drag & drop state
  let dragItem: { id: string; type: "session" | "folder" } | null = null;
  let dropTarget: string | null = null;

  onMount(async () => {
    await refreshTree();
  });

  export async function refreshTree() {
    const tree = await GetSessionTree();
    sessionTree.set(tree || []);
  }

  async function toggleFolder(id: string) {
    await ToggleFolderExpanded(id);
    await refreshTree();
  }

  function handleDoubleClick(node: TreeNode) {
    if (node.type === "session" && node.session) {
      connectToSession(node);
    }
  }

  async function connectToSession(node: TreeNode) {
    if (!node.session) return;

    // Check if this session should use an external terminal
    if (node.session.terminalType === "system") {
      try {
        await ConnectSessionExternal(node.session.id);
      } catch (err: any) {
        window.dispatchEvent(
          new CustomEvent("connection-error", {
            detail: { message: "Failed to launch external terminal: " + err },
          })
        );
      }
      return;
    }

    const tabId = createTab(node.name, node.session.id);

    // Dispatch custom event to trigger connection
    window.dispatchEvent(
      new CustomEvent("connect-session", {
        detail: { tabId, session: node.session },
      })
    );
  }

  function handleContextMenu(e: MouseEvent, node: TreeNode) {
    e.preventDefault();
    contextMenu = { x: e.clientX, y: e.clientY, node };
  }

  function closeContextMenu() {
    contextMenu = null;
  }

  function editNode() {
    if (!contextMenu) return;
    const node = contextMenu.node;
    if (node.type === "session") {
      editingSession.set(node.id);
      showSessionForm.set(true);
    } else {
      editingFolder.set(node.id);
      showFolderForm.set(true);
    }
    closeContextMenu();
  }

  async function deleteNode() {
    if (!contextMenu) return;
    const node = contextMenu.node;
    if (node.type === "session") {
      await DeleteSession(node.id);
    } else {
      await DeleteFolder(node.id);
    }
    await refreshTree();
    closeContextMenu();
  }

  function addSessionToFolder() {
    if (!contextMenu || contextMenu.node.type !== "folder") return;
    selectedFolderId.set(contextMenu.node.id);
    showSessionForm.set(true);
    closeContextMenu();
  }

  // ---- Drag & Drop ----

  function handleDragStart(e: DragEvent, id: string, type: "session" | "folder") {
    if (!e.dataTransfer) return;
    dragItem = { id, type };
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("text/plain", JSON.stringify({ id, type }));
  }

  function handleDragOver(e: DragEvent, folderId: string) {
    if (!dragItem) return;
    // Don't allow dropping a folder onto itself
    if (dragItem.type === "folder" && dragItem.id === folderId) return;
    e.preventDefault();
    if (e.dataTransfer) e.dataTransfer.dropEffect = "move";
    dropTarget = folderId;
  }

  function handleDragLeave(e: DragEvent, folderId: string) {
    if (dropTarget === folderId) {
      dropTarget = null;
    }
  }

  async function handleDrop(e: DragEvent, targetFolderId: string) {
    e.preventDefault();
    e.stopPropagation();
    dropTarget = null;
    if (!dragItem) return;

    try {
      if (dragItem.type === "session") {
        await MoveSession(dragItem.id, targetFolderId);
      } else {
        if (dragItem.id === targetFolderId) return;
        await MoveFolder(dragItem.id, targetFolderId);
      }
      await refreshTree();
    } catch (err: any) {
      console.error("Move failed:", err);
    }

    dragItem = null;
  }

  function handleDragEnd() {
    dragItem = null;
    dropTarget = null;
  }
</script>

<svelte:window on:click={closeContextMenu} />

<div class="tree">
  {#each $sessionTree as node}
    {#if node.type === "folder"}
      <div class="tree-folder">
        <button
          class="tree-item folder"
          class:drop-target={dropTarget === node.id}
          draggable="true"
          on:dragstart={(e) => handleDragStart(e, node.id, "folder")}
          on:dragover={(e) => handleDragOver(e, node.id)}
          on:dragleave={(e) => handleDragLeave(e, node.id)}
          on:drop={(e) => handleDrop(e, node.id)}
          on:dragend={handleDragEnd}
          on:click={() => toggleFolder(node.id)}
          on:contextmenu={(e) => handleContextMenu(e, node)}
        >
          <svg class="chevron" class:expanded={node.expanded} width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="9 18 15 12 9 6"/>
          </svg>
          <svg width="14" height="14" viewBox="0 0 24 24" fill={node.color || "#6c7086"} stroke="none">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          </svg>
          <span class="node-name">{node.name}</span>
        </button>
        {#if node.expanded && node.children}
          <div class="tree-children">
            {#each node.children as child}
              {#if child.type === "folder"}
                <div class="tree-folder">
                  <button
                    class="tree-item folder"
                    class:drop-target={dropTarget === child.id}
                    draggable="true"
                    on:dragstart={(e) => handleDragStart(e, child.id, "folder")}
                    on:dragover={(e) => handleDragOver(e, child.id)}
                    on:dragleave={(e) => handleDragLeave(e, child.id)}
                    on:drop={(e) => handleDrop(e, child.id)}
                    on:dragend={handleDragEnd}
                    on:click={() => toggleFolder(child.id)}
                    on:contextmenu={(e) => handleContextMenu(e, child)}
                  >
                    <svg class="chevron" class:expanded={child.expanded} width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="9 18 15 12 9 6"/>
                    </svg>
                    <svg width="14" height="14" viewBox="0 0 24 24" fill={child.color || "#6c7086"} stroke="none">
                      <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
                    </svg>
                    <span class="node-name">{child.name}</span>
                  </button>
                  {#if child.expanded && child.children}
                    <div class="tree-children">
                      {#each child.children as grandchild}
                        <button
                          class="tree-item session"
                          draggable="true"
                          on:dragstart={(e) => handleDragStart(e, grandchild.id, grandchild.type === "folder" ? "folder" : "session")}
                          on:dragend={handleDragEnd}
                          on:dblclick={() => handleDoubleClick(grandchild)}
                          on:contextmenu={(e) => handleContextMenu(e, grandchild)}
                        >
                          <span class="session-dot" style:background={grandchild.color || "#45a049"}></span>
                          <span class="node-name">{grandchild.name}</span>
                          {#if grandchild.session}
                            <span class="node-host">{grandchild.session.host}</span>
                          {/if}
                        </button>
                      {/each}
                    </div>
                  {/if}
                </div>
              {:else}
                <button
                  class="tree-item session"
                  draggable="true"
                  on:dragstart={(e) => handleDragStart(e, child.id, "session")}
                  on:dragend={handleDragEnd}
                  on:dblclick={() => handleDoubleClick(child)}
                  on:contextmenu={(e) => handleContextMenu(e, child)}
                >
                  <span class="session-dot" style:background={child.color || "#45a049"}></span>
                  <span class="node-name">{child.name}</span>
                  {#if child.session}
                    <span class="node-host">{child.session.host}</span>
                  {/if}
                </button>
              {/if}
            {/each}
          </div>
        {/if}
      </div>
    {:else}
      <button
        class="tree-item session"
        draggable="true"
        on:dragstart={(e) => handleDragStart(e, node.id, "session")}
        on:dragend={handleDragEnd}
        on:dblclick={() => handleDoubleClick(node)}
        on:contextmenu={(e) => handleContextMenu(e, node)}
      >
        <span class="session-dot" style:background={node.color || "#45a049"}></span>
        <span class="node-name">{node.name}</span>
        {#if node.session}
          <span class="node-host">{node.session.host}</span>
        {/if}
      </button>
    {/if}
  {/each}
</div>

{#if contextMenu}
  <div class="context-menu" style="left: {contextMenu.x}px; top: {contextMenu.y}px">
    {#if contextMenu.node.type === "session"}
      <button on:click={() => { connectToSession(contextMenu.node); closeContextMenu(); }}>Connect</button>
    {/if}
    {#if contextMenu.node.type === "folder"}
      <button on:click={addSessionToFolder}>Add Session</button>
    {/if}
    <button on:click={editNode}>Edit</button>
    <button class="danger" on:click={deleteNode}>Delete</button>
  </div>
{/if}

<style>
  .tree {
    user-select: none;
  }

  .tree-item {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 6px 16px;
    background: none;
    border: none;
    color: #ccc;
    font-size: 13px;
    cursor: pointer;
    text-align: left;
    white-space: nowrap;
    overflow: hidden;
    transition: background 0.15s, outline 0.15s;
  }

  .tree-item:hover {
    background: #252640;
  }

  .tree-item.folder {
    color: #e0e0e0;
    font-weight: 500;
  }

  .tree-item.drop-target {
    outline: 1.5px dashed #4a6cf7;
    outline-offset: -1.5px;
    background: rgba(74, 108, 247, 0.08);
  }

  .tree-children {
    padding-left: 16px;
  }

  .chevron {
    transition: transform 0.15s ease;
    flex-shrink: 0;
  }

  .chevron.expanded {
    transform: rotate(90deg);
  }

  .session-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
    margin-left: 20px;
  }

  .node-name {
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .node-host {
    font-size: 11px;
    color: #666;
    margin-left: auto;
    flex-shrink: 0;
  }

  .context-menu {
    position: fixed;
    background: #2a2b3d;
    border: 1px solid #3a3b4d;
    border-radius: 6px;
    padding: 4px;
    z-index: 1000;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
    min-width: 140px;
  }

  .context-menu button {
    display: block;
    width: 100%;
    padding: 8px 12px;
    background: none;
    border: none;
    color: #ccc;
    font-size: 13px;
    cursor: pointer;
    text-align: left;
    border-radius: 4px;
  }

  .context-menu button:hover {
    background: #3a3b4d;
    color: #fff;
  }

  .context-menu button.danger:hover {
    background: #5c2020;
    color: #ff6b6b;
  }
</style>
