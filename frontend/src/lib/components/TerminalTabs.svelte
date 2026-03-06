<script lang="ts">
  import { tabs, activeTabId, closeTab } from "../stores/terminals";
  import { DisconnectTab, IsSSHSession } from "../../../wailsjs/go/main/App";
  import TerminalView from "./TerminalView.svelte";
  import ScpPanel from "./ScpPanel.svelte";

  let scpOpen = false;
  let scpWidth = 320;
  let isResizingScp = false;
  let isCurrentSSH = false;

  function openLocalTerminal() {
    window.dispatchEvent(new CustomEvent("open-local-terminal", { detail: {} }));
  }

  function selectTab(id: string) {
    activeTabId.set(id);
  }

  async function handleClose(e: MouseEvent, id: string) {
    e.stopPropagation();
    await DisconnectTab(id);
    closeTab(id);
  }

  function handleMouseDown(e: MouseEvent, id: string) {
    if (e.button === 1) {
      e.preventDefault();
      DisconnectTab(id).then(() => closeTab(id));
    }
  }

  function toggleScp() {
    scpOpen = !scpOpen;
  }

  function startScpResize(e: MouseEvent) {
    isResizingScp = true;
    e.preventDefault();
    const startX = e.clientX;
    const startWidth = scpWidth;
    const onMove = (ev: MouseEvent) => {
      scpWidth = Math.max(200, Math.min(600, startWidth + (ev.clientX - startX)));
    };
    const onUp = () => {
      isResizingScp = false;
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    };
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
  }

  // Check if current tab is SSH
  $: {
    const currentId = $activeTabId;
    if (currentId) {
      IsSSHSession(currentId).then((v) => { isCurrentSSH = v; });
    } else {
      isCurrentSSH = false;
    }
  }
</script>

<div class="terminal-area">
  {#if $tabs.length > 0}
    <div class="tab-bar">
      {#each $tabs as tab (tab.id)}
        <button
          class="tab"
          class:active={$activeTabId === tab.id}
          on:click={() => selectTab(tab.id)}
          on:mousedown={(e) => handleMouseDown(e, tab.id)}
        >
          <span class="tab-status" class:connected={tab.connected}></span>
          <span class="tab-title">{tab.title}</span>
          <button class="tab-close" on:click={(e) => handleClose(e, tab.id)}>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"/>
              <line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </button>
      {/each}
      <button class="tab-new" on:click={openLocalTerminal} title="New Terminal (Ctrl+T)">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"/>
          <line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
      </button>
      <div class="tab-bar-spacer"></div>
      {#if isCurrentSSH}
        <button class="scp-toggle" class:active={scpOpen} on:click={toggleScp} title="Toggle SCP file browser">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          </svg>
          <span>SCP</span>
        </button>
      {/if}
    </div>
    <div class="terminal-content">
      {#if scpOpen && isCurrentSSH && $activeTabId}
        <div class="scp-container" style="width: {scpWidth}px; min-width: {scpWidth}px">
          {#each $tabs as tab (tab.id)}
            {#if tab.connected}
              <ScpPanel tabId={tab.id} active={$activeTabId === tab.id} />
            {/if}
          {/each}
          <div class="scp-resize-handle" on:mousedown={startScpResize}></div>
        </div>
      {/if}
      <div class="terminal-panels">
        {#each $tabs as tab (tab.id)}
          <TerminalView tabId={tab.id} active={$activeTabId === tab.id} />
        {/each}
      </div>
    </div>
  {:else}
    <div class="empty-state">
      <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="#3a3b4d" stroke-width="1.5">
        <polyline points="4 17 10 11 4 5"/>
        <line x1="12" y1="19" x2="20" y2="19"/>
      </svg>
      <p>Double-click a session to connect</p>
      <p class="hint">or use the quick connect bar above</p>
      <button class="open-terminal-btn" on:click={openLocalTerminal}>
        Open Local Terminal
      </button>
      <div class="shortcuts-hint">
        <span><kbd>Ctrl</kbd>+<kbd>T</kbd> New terminal</span>
        <span><kbd>Ctrl</kbd>+<kbd>W</kbd> Close tab</span>
        <span><kbd>Ctrl</kbd>+<kbd>Tab</kbd> Next tab</span>
        <span><kbd>Ctrl</kbd>+<kbd>K</kbd> Search</span>
      </div>
    </div>
  {/if}
</div>

<style>
  .terminal-area {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background: #14142a;
  }

  .tab-bar {
    display: flex;
    background: #1a1b2e;
    border-bottom: 1px solid #2a2b3d;
    overflow-x: auto;
    flex-shrink: 0;
  }

  .tab-bar::-webkit-scrollbar {
    height: 0;
  }

  .tab {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    background: none;
    border: none;
    border-right: 1px solid #2a2b3d;
    color: #888;
    font-size: 13px;
    cursor: pointer;
    white-space: nowrap;
    min-width: 120px;
    max-width: 200px;
    transition: all 0.15s;
  }

  .tab:hover {
    background: #252640;
    color: #ccc;
  }

  .tab.active {
    background: #14142a;
    color: #e0e0e0;
    border-bottom: 2px solid #4a6cf7;
  }

  .tab-status {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: #555;
    flex-shrink: 0;
  }

  .tab-status.connected {
    background: #45a049;
  }

  .tab-title {
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .tab-close {
    background: none;
    border: none;
    color: #666;
    cursor: pointer;
    padding: 2px;
    display: flex;
    border-radius: 4px;
    margin-left: auto;
  }

  .tab-close:hover {
    color: #ff6b6b;
    background: rgba(255, 107, 107, 0.1);
  }

  .tab-new {
    background: none;
    border: none;
    color: #555;
    cursor: pointer;
    padding: 8px 12px;
    display: flex;
    align-items: center;
    flex-shrink: 0;
  }

  .tab-new:hover {
    color: #ccc;
    background: #252640;
  }

  .tab-bar-spacer {
    flex: 1;
  }

  .scp-toggle {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 14px;
    margin: 3px 6px;
    background: #2a2b3d;
    border: 1px solid #3a3b4d;
    border-radius: 5px;
    color: #888;
    font-size: 12px;
    cursor: pointer;
    white-space: nowrap;
    flex-shrink: 0;
    transition: all 0.15s;
  }

  .scp-toggle:hover {
    color: #ccc;
    border-color: #4a6cf7;
  }

  .scp-toggle.active {
    background: #4a6cf7;
    border-color: #4a6cf7;
    color: #fff;
  }

  .terminal-content {
    flex: 1;
    display: flex;
    overflow: hidden;
  }

  .scp-container {
    position: relative;
    flex-shrink: 0;
    overflow: hidden;
  }

  .scp-resize-handle {
    position: absolute;
    top: 0;
    right: -3px;
    width: 6px;
    height: 100%;
    cursor: col-resize;
    z-index: 10;
  }

  .scp-resize-handle:hover {
    background: #4a6cf7;
    opacity: 0.5;
  }

  .terminal-panels {
    flex: 1;
    overflow: hidden;
    position: relative;
  }

  .empty-state {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    color: #555;
    gap: 12px;
  }

  .empty-state p {
    margin: 0;
    font-size: 16px;
  }

  .empty-state .hint {
    font-size: 13px;
    color: #444;
  }

  .open-terminal-btn {
    margin-top: 8px;
    padding: 8px 20px;
    background: #4a6cf7;
    color: #fff;
    border: none;
    border-radius: 6px;
    font-size: 13px;
    cursor: pointer;
    transition: background 0.15s;
  }

  .open-terminal-btn:hover {
    background: #5b7bf8;
  }

  .shortcuts-hint {
    display: flex;
    gap: 16px;
    margin-top: 20px;
    font-size: 11px;
    color: #444;
  }

  .shortcuts-hint span {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  :global(kbd) {
    background: #2a2b3d;
    color: #888;
    padding: 1px 5px;
    border-radius: 3px;
    font-size: 10px;
    font-family: inherit;
    border: 1px solid #3a3b4d;
  }
</style>
