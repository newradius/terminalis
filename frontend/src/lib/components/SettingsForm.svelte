<script lang="ts">
  import { showSettings } from "../stores/sessions";
  import { appConfig, saveConfig } from "../stores/config";
  import type { AppConfig } from "../stores/config";
  import { onMount } from "svelte";

  let fontSize = 14;
  let terminalBackground = "#14142a";
  let saving = false;

  onMount(() => {
    const cfg = $appConfig;
    fontSize = cfg.fontSize;
    terminalBackground = cfg.terminalBackground;
  });

  async function save() {
    saving = true;
    try {
      await saveConfig({ ...$appConfig, fontSize, terminalBackground });
      close();
    } finally {
      saving = false;
    }
  }

  function close() {
    showSettings.set(false);
  }
</script>

<div class="overlay" on:click|self={close}>
  <div class="modal">
    <div class="modal-header">
      <h2>Settings</h2>
      <button class="close-btn" on:click={close}>&times;</button>
    </div>

    <div class="modal-body">
      <div class="form-row">
        <label>
          <span>Terminal Font Size</span>
          <div class="range-row">
            <input type="range" min="10" max="24" step="1" bind:value={fontSize} />
            <span class="range-value">{fontSize}px</span>
          </div>
        </label>
      </div>

      <div class="form-row">
        <label>
          <span>Terminal Background</span>
          <div class="color-row">
            <input type="color" bind:value={terminalBackground} class="color-input" />
            <input type="text" bind:value={terminalBackground} class="hex-input" maxlength="7" placeholder="#14142a" />
          </div>
        </label>
      </div>
    </div>

    <div class="modal-footer">
      <button class="btn-secondary" on:click={close}>Cancel</button>
      <button class="btn-primary" on:click={save} disabled={saving}>
        {saving ? "Saving..." : "Save"}
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
    width: 420px;
    max-height: 90vh;
    overflow-y: auto;
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

  .range-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .range-row input[type="range"] {
    flex: 1;
    accent-color: #4a6cf7;
    height: 6px;
  }

  .range-value {
    font-size: 14px;
    color: #e0e0e0;
    min-width: 40px;
    text-align: right;
    font-variant-numeric: tabular-nums;
  }

  .color-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .color-input {
    width: 40px;
    height: 36px;
    border: 1px solid #2a2b3d;
    border-radius: 6px;
    background: #15162a;
    cursor: pointer;
    padding: 2px;
  }

  .color-input::-webkit-color-swatch-wrapper {
    padding: 2px;
  }

  .color-input::-webkit-color-swatch {
    border: none;
    border-radius: 3px;
  }

  .hex-input {
    flex: 1;
    background: #15162a;
    border: 1px solid #2a2b3d;
    border-radius: 6px;
    padding: 8px 12px;
    color: #e0e0e0;
    font-size: 14px;
    font-family: 'Cascadia Code', 'Fira Code', monospace;
    outline: none;
    transition: border-color 0.2s;
  }

  .hex-input:focus {
    border-color: #4a6cf7;
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
