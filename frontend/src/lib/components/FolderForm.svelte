<script lang="ts">
  import { showFolderForm, editingFolder, sessionTree } from "../stores/sessions";
  import { SaveFolder, GetSessionTree, GetFolderByID } from "../../../wailsjs/go/main/App";
  import { onMount } from "svelte";

  let name = "";
  let color = "#6c7086";
  let loading = false;

  const colors = ["#6c7086", "#45a049", "#2196F3", "#FF9800", "#E91E63", "#9C27B0", "#00BCD4", "#FF5722"];

  onMount(async () => {
    if ($editingFolder) {
      const folder = await GetFolderByID($editingFolder);
      if (folder) {
        name = folder.name;
        color = folder.color || "#6c7086";
      }
    }
  });

  async function save() {
    if (!name.trim()) return;
    loading = true;

    try {
      await SaveFolder({
        id: $editingFolder || "",
        name: name.trim(),
        parentId: "",
        color,
        expanded: true,
      });
      const tree = await GetSessionTree();
      sessionTree.set(tree || []);
      close();
    } finally {
      loading = false;
    }
  }

  function close() {
    showFolderForm.set(false);
    editingFolder.set(null);
  }
</script>

<div class="overlay" on:click|self={close}>
  <div class="modal">
    <div class="modal-header">
      <h2>{$editingFolder ? "Edit Folder" : "New Folder"}</h2>
      <button class="close-btn" on:click={close}>&times;</button>
    </div>

    <div class="modal-body">
      <div class="form-row">
        <label>
          <span>Folder Name</span>
          <input type="text" bind:value={name} placeholder="Production Servers" autofocus />
        </label>
      </div>

      <div class="form-row">
        <label>
          <span>Color</span>
          <div class="color-picker">
            {#each colors as c}
              <button
                class="color-swatch"
                class:selected={color === c}
                style:background={c}
                on:click={() => (color = c)}
              ></button>
            {/each}
          </div>
        </label>
      </div>
    </div>

    <div class="modal-footer">
      <button class="btn-secondary" on:click={close}>Cancel</button>
      <button class="btn-primary" on:click={save} disabled={loading || !name.trim()}>
        {loading ? "Saving..." : "Save"}
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
    width: 400px;
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

  input[type="text"] {
    background: #15162a;
    border: 1px solid #2a2b3d;
    border-radius: 6px;
    padding: 8px 12px;
    color: #e0e0e0;
    font-size: 14px;
    outline: none;
  }

  input:focus {
    border-color: #4a6cf7;
  }

  .color-picker {
    display: flex;
    gap: 8px;
  }

  .color-swatch {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    border: 2px solid transparent;
    cursor: pointer;
  }

  .color-swatch.selected {
    border-color: white;
    transform: scale(1.15);
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
