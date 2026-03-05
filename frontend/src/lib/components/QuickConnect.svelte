<script lang="ts">
  import { createTab } from "../stores/terminals";

  let connString = "";
  let showPassword = false;
  let password = "";

  function connect() {
    if (!connString.trim()) return;
    const tabId = createTab(connString.trim());

    window.dispatchEvent(
      new CustomEvent("quick-connect", {
        detail: { tabId, connString: connString.trim(), password },
      })
    );

    connString = "";
    password = "";
    showPassword = false;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Enter") {
      connect();
    }
  }
</script>

<div class="quick-connect">
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#666" stroke-width="2">
    <polyline points="4 17 10 11 4 5" />
    <line x1="12" y1="19" x2="20" y2="19" />
  </svg>
  <input
    type="text"
    bind:value={connString}
    placeholder="Quick connect: user@host:port"
    on:keydown={handleKeydown}
  />
  {#if showPassword}
    <input
      type="password"
      bind:value={password}
      placeholder="Password"
      on:keydown={handleKeydown}
      class="password-input"
    />
  {/if}
  <button class="toggle-pwd" on:click={() => (showPassword = !showPassword)} title="Toggle password field">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      {#if showPassword}
        <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/>
        <line x1="1" y1="1" x2="23" y2="23"/>
      {:else}
        <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
        <circle cx="12" cy="12" r="3"/>
      {/if}
    </svg>
  </button>
  <button class="connect-btn" on:click={connect} disabled={!connString.trim()}>
    Connect
  </button>
</div>

<style>
  .quick-connect {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    background: #1a1b2e;
    border-bottom: 1px solid #2a2b3d;
  }

  input[type="text"],
  input[type="password"] {
    background: #15162a;
    border: 1px solid #2a2b3d;
    border-radius: 6px;
    padding: 6px 12px;
    color: #e0e0e0;
    font-size: 13px;
    outline: none;
    font-family: 'Cascadia Code', 'Fira Code', monospace;
  }

  input[type="text"] {
    flex: 1;
    min-width: 200px;
  }

  input[type="text"]:focus,
  input[type="password"]:focus {
    border-color: #4a6cf7;
  }

  .password-input {
    width: 150px;
  }

  .toggle-pwd {
    background: none;
    border: none;
    color: #666;
    cursor: pointer;
    padding: 4px;
    display: flex;
    border-radius: 4px;
  }

  .toggle-pwd:hover {
    color: #ccc;
  }

  .connect-btn {
    background: #4a6cf7;
    border: none;
    border-radius: 6px;
    color: white;
    padding: 6px 16px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    white-space: nowrap;
  }

  .connect-btn:hover {
    background: #5b7bf8;
  }

  .connect-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
