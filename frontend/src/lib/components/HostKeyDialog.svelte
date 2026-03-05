<script lang="ts">
  import type { HostKeyInfo } from "../types";
  import { AcceptHostKey } from "../../../wailsjs/go/main/App";

  export let info: HostKeyInfo;
  export let onClose: () => void;

  async function accept() {
    await AcceptHostKey(true);
    onClose();
  }

  async function reject() {
    await AcceptHostKey(false);
    onClose();
  }
</script>

<div class="overlay">
  <div class="dialog">
    <div class="dialog-header">
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#FF9800" stroke-width="2">
        <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
        <line x1="12" y1="9" x2="12" y2="13"/>
        <line x1="12" y1="17" x2="12.01" y2="17"/>
      </svg>
      <h2>{info.isNew ? "Unknown Host" : "Host Key Changed!"}</h2>
    </div>

    <div class="dialog-body">
      {#if info.isMismatch}
        <p class="warning">
          WARNING: The host key for this server has changed. This could indicate a
          man-in-the-middle attack or the server was reinstalled.
        </p>
      {:else}
        <p>
          The authenticity of host '<strong>{info.host}</strong>' can't be established.
        </p>
      {/if}

      <div class="key-info">
        <div class="key-row">
          <span class="label">Host:</span>
          <span class="value">{info.host}</span>
        </div>
        <div class="key-row">
          <span class="label">Key Type:</span>
          <span class="value">{info.keyType}</span>
        </div>
        <div class="key-row">
          <span class="label">Fingerprint:</span>
          <span class="value fingerprint">{info.fingerprint}</span>
        </div>
      </div>

      <p>Are you sure you want to continue connecting?</p>
    </div>

    <div class="dialog-footer">
      <button class="btn-danger" on:click={reject}>Reject</button>
      <button class="btn-primary" on:click={accept}>Accept & Connect</button>
    </div>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 200;
    backdrop-filter: blur(3px);
  }

  .dialog {
    background: #1e1f33;
    border: 1px solid #2a2b3d;
    border-radius: 12px;
    width: 480px;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.6);
  }

  .dialog-header {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 20px 24px 16px;
    border-bottom: 1px solid #2a2b3d;
  }

  .dialog-header h2 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: #e0e0e0;
  }

  .dialog-body {
    padding: 20px 24px;
  }

  .dialog-body p {
    color: #ccc;
    font-size: 14px;
    margin: 0 0 16px;
    line-height: 1.5;
  }

  .warning {
    color: #ff6b6b !important;
    background: rgba(255, 107, 107, 0.1);
    padding: 12px;
    border-radius: 6px;
    border-left: 3px solid #ff6b6b;
  }

  .key-info {
    background: #15162a;
    border-radius: 8px;
    padding: 12px 16px;
    margin-bottom: 16px;
  }

  .key-row {
    display: flex;
    gap: 12px;
    padding: 4px 0;
    font-size: 13px;
  }

  .label {
    color: #888;
    min-width: 90px;
  }

  .value {
    color: #e0e0e0;
    word-break: break-all;
  }

  .fingerprint {
    font-family: 'Cascadia Code', 'Fira Code', monospace;
    font-size: 12px;
  }

  .dialog-footer {
    padding: 16px 24px 20px;
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    border-top: 1px solid #2a2b3d;
  }

  .btn-danger {
    background: #5c2020;
    border: none;
    border-radius: 6px;
    color: #ff6b6b;
    padding: 10px 20px;
    font-size: 14px;
    cursor: pointer;
  }

  .btn-danger:hover {
    background: #6c2828;
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
</style>
