<script lang="ts">
  export let host: string;
  export let onSubmit: (password: string) => void;
  export let onCancel: () => void;

  let password = "";

  function submit() {
    onSubmit(password);
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Enter") submit();
    if (e.key === "Escape") onCancel();
  }
</script>

<div class="overlay" on:click|self={onCancel}>
  <div class="dialog">
    <div class="dialog-header">
      <h2>Password Required</h2>
    </div>
    <div class="dialog-body">
      <p>Enter password for <strong>{host}</strong></p>
      <input
        type="password"
        bind:value={password}
        placeholder="Password"
        on:keydown={handleKeydown}
        autofocus
      />
    </div>
    <div class="dialog-footer">
      <button class="btn-secondary" on:click={onCancel}>Cancel</button>
      <button class="btn-primary" on:click={submit}>Connect</button>
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
    z-index: 150;
  }

  .dialog {
    background: #1e1f33;
    border: 1px solid #2a2b3d;
    border-radius: 12px;
    width: 380px;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  }

  .dialog-header {
    padding: 20px 24px 12px;
  }

  .dialog-header h2 {
    margin: 0;
    font-size: 16px;
    color: #e0e0e0;
  }

  .dialog-body {
    padding: 0 24px 16px;
  }

  .dialog-body p {
    color: #999;
    font-size: 13px;
    margin: 0 0 12px;
  }

  input {
    width: 100%;
    background: #15162a;
    border: 1px solid #2a2b3d;
    border-radius: 6px;
    padding: 10px 12px;
    color: #e0e0e0;
    font-size: 14px;
    outline: none;
  }

  input:focus {
    border-color: #4a6cf7;
  }

  .dialog-footer {
    padding: 12px 24px 20px;
    display: flex;
    justify-content: flex-end;
    gap: 10px;
  }

  .btn-secondary {
    background: #2a2b3d;
    border: none;
    border-radius: 6px;
    color: #ccc;
    padding: 8px 16px;
    font-size: 13px;
    cursor: pointer;
  }

  .btn-primary {
    background: #4a6cf7;
    border: none;
    border-radius: 6px;
    color: white;
    padding: 8px 20px;
    font-size: 13px;
    cursor: pointer;
  }

  .btn-primary:hover {
    background: #5b7bf8;
  }
</style>
