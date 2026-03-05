<script lang="ts">
  import { tabs, activeTabId, closeTab } from "../stores/terminals";
  import { DisconnectTab } from "../../../wailsjs/go/main/App";
  import TerminalView from "./TerminalView.svelte";

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
</script>

<div class="terminal-area">
  {#if $tabs.length > 0}
    <div class="tab-bar">
      {#each $tabs as tab (tab.id)}
        <button
          class="tab"
          class:active={$activeTabId === tab.id}
          on:click={() => selectTab(tab.id)}
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
    </div>
    <div class="terminal-panels">
      {#each $tabs as tab (tab.id)}
        <TerminalView tabId={tab.id} active={$activeTabId === tab.id} />
      {/each}
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
</style>
