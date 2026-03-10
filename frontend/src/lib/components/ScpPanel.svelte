<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { SftpListDir, SftpDownloadFile, SftpUploadFile, SftpUploadPaths, SftpDownloadToDownloads, SftpGetPwd, SftpGetHome } from "../../../wailsjs/go/main/App";
  import { OnFileDrop, OnFileDropOff } from "../../../wailsjs/runtime/runtime";

  export let tabId: string;
  export let active: boolean = false;

  interface FileEntry {
    name: string;
    size: number;
    isDir: boolean;
    modTime: number;
    mode: string;
  }

  let currentPath = "";
  let files: FileEntry[] = [];
  let loading = false;
  let error = "";
  let followTerminal = false;
  let followInterval: ReturnType<typeof setInterval> | null = null;
  let sortColumn: "name" | "size" | "modTime" = "name";
  let sortAsc = true;
  let showHidden = true;
  let dropHover = false;
  let panelEl: HTMLDivElement;

  onMount(async () => {
    OnFileDrop(handleFileDrop, true);
    await loadHome();
  });

  onDestroy(() => {
    stopFollowing();
    OnFileDropOff();
  });

  async function handleFileDrop(_x: number, _y: number, paths: string[]) {
    if (!active || !paths || paths.length === 0) return;
    dropHover = false;
    try {
      const count = await SftpUploadPaths(tabId, currentPath, paths);
      await loadDir(currentPath);
      window.dispatchEvent(new CustomEvent("toast-success", {
        detail: { message: `Uploaded ${count} file${count !== 1 ? "s" : ""}` },
      }));
    } catch (err: any) {
      window.dispatchEvent(new CustomEvent("connection-error", {
        detail: { message: "Upload failed: " + err },
      }));
    }
  }

  function handleDragStart(e: DragEvent, entry: FileEntry) {
    if (entry.isDir) {
      e.preventDefault();
      return;
    }
    e.dataTransfer?.setData("text/plain", entry.name);
    e.dataTransfer!.effectAllowed = "copy";
  }

  async function handleDragEnd(e: DragEvent, entry: FileEntry) {
    // If dropped outside the panel, prompt for directory and download there
    if (entry.isDir) return;
    const rect = panelEl?.getBoundingClientRect();
    if (!rect) return;
    const { clientX, clientY } = e;
    const insidePanel =
      clientX >= rect.left && clientX <= rect.right &&
      clientY >= rect.top && clientY <= rect.bottom;
    if (insidePanel) return;

    const remotePath = currentPath === "/"
      ? "/" + entry.name
      : currentPath + "/" + entry.name;
    try {
      const localPath = await SftpDownloadToDownloads(tabId, remotePath);
      const fileName = localPath.split(/[/\\]/).pop() || entry.name;
      window.dispatchEvent(new CustomEvent("toast-success", {
        detail: { message: `Downloaded ${fileName} to Downloads` },
      }));
    } catch (err: any) {
      window.dispatchEvent(new CustomEvent("connection-error", {
        detail: { message: "Download failed: " + err },
      }));
    }
  }

  async function loadHome() {
    loading = true;
    error = "";
    try {
      const home = await SftpGetHome(tabId);
      currentPath = home || "/";
      await loadDir(currentPath);
    } catch (err: any) {
      error = "Failed to connect SFTP: " + err;
      loading = false;
    }
  }

  async function loadDir(path: string) {
    loading = true;
    error = "";
    try {
      const result = await SftpListDir(tabId, path);
      currentPath = result.path;
      files = result.files || [];
    } catch (err: any) {
      error = err.toString();
    } finally {
      loading = false;
    }
  }

  async function navigateTo(entry: FileEntry) {
    if (entry.isDir) {
      const newPath = currentPath === "/"
        ? "/" + entry.name
        : currentPath + "/" + entry.name;
      await loadDir(newPath);
    }
  }

  async function goUp() {
    const parts = currentPath.split("/").filter(Boolean);
    parts.pop();
    const parent = "/" + parts.join("/");
    await loadDir(parent);
  }

  async function downloadFile(entry: FileEntry) {
    if (entry.isDir) return;
    const remotePath = currentPath === "/"
      ? "/" + entry.name
      : currentPath + "/" + entry.name;
    try {
      await SftpDownloadFile(tabId, remotePath);
      window.dispatchEvent(new CustomEvent("toast-success", {
        detail: { message: `Downloaded ${entry.name}` },
      }));
    } catch (err: any) {
      window.dispatchEvent(new CustomEvent("connection-error", {
        detail: { message: "Download failed: " + err },
      }));
    }
  }

  async function uploadFile() {
    try {
      await SftpUploadFile(tabId, currentPath);
      await loadDir(currentPath);
      window.dispatchEvent(new CustomEvent("toast-success", {
        detail: { message: "File uploaded" },
      }));
    } catch (err: any) {
      if (err) {
        window.dispatchEvent(new CustomEvent("connection-error", {
          detail: { message: "Upload failed: " + err },
        }));
      }
    }
  }

  async function syncWithTerminal() {
    try {
      const pwd = await SftpGetPwd(tabId);
      if (pwd && pwd !== currentPath) {
        await loadDir(pwd);
      }
    } catch {
      // Silently fail - terminal might not have a shell running
    }
  }

  function toggleFollow() {
    followTerminal = !followTerminal;
    if (followTerminal) {
      syncWithTerminal();
      followInterval = setInterval(syncWithTerminal, 2000);
    } else {
      stopFollowing();
    }
  }

  function stopFollowing() {
    if (followInterval) {
      clearInterval(followInterval);
      followInterval = null;
    }
  }

  function toggleSort(col: "name" | "size" | "modTime") {
    if (sortColumn === col) {
      sortAsc = !sortAsc;
    } else {
      sortColumn = col;
      sortAsc = true;
    }
  }

  function formatSize(bytes: number): string {
    if (bytes < 1024) return bytes + " B";
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
    if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + " MB";
    return (bytes / (1024 * 1024 * 1024)).toFixed(1) + " GB";
  }

  function formatDate(ts: number): string {
    if (!ts) return "";
    const d = new Date(ts * 1000);
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, "0");
    const day = String(d.getDate()).padStart(2, "0");
    const h = String(d.getHours()).padStart(2, "0");
    const min = String(d.getMinutes()).padStart(2, "0");
    return `${y}-${m}-${day} ${h}:${min}`;
  }

  function handlePathKeydown(e: KeyboardEvent) {
    if (e.key === "Enter") {
      loadDir((e.target as HTMLInputElement).value);
    }
  }

  function handleDoubleClick(entry: FileEntry) {
    if (entry.isDir) {
      navigateTo(entry);
    } else {
      downloadFile(entry);
    }
  }

  $: sortedFiles = (() => {
    let filtered = showHidden ? files : files.filter(f => !f.name.startsWith("."));
    // Always show directories first
    const dirs = filtered.filter(f => f.isDir);
    const regular = filtered.filter(f => !f.isDir);

    const compare = (a: FileEntry, b: FileEntry) => {
      let result = 0;
      if (sortColumn === "name") result = a.name.localeCompare(b.name);
      else if (sortColumn === "size") result = a.size - b.size;
      else if (sortColumn === "modTime") result = a.modTime - b.modTime;
      return sortAsc ? result : -result;
    };

    dirs.sort(compare);
    regular.sort(compare);
    return [...dirs, ...regular];
  })();

  // Breadcrumb parts
  $: breadcrumbs = (() => {
    const parts = currentPath.split("/").filter(Boolean);
    const crumbs = [{ name: "/", path: "/" }];
    let accum = "";
    for (const p of parts) {
      accum += "/" + p;
      crumbs.push({ name: p, path: accum });
    }
    return crumbs;
  })();
</script>

{#if active}
<div
  class="scp-panel"
  class:drop-hover={dropHover}
  bind:this={panelEl}
  style="--wails-drop-target: drop"
  on:wails-drop-target-active={() => dropHover = true}
  on:wails-drop-target-inactive={() => dropHover = false}
>
  {#if dropHover}
    <div class="drop-overlay">
      <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
        <polyline points="17 8 12 3 7 8"/>
        <line x1="12" y1="3" x2="12" y2="15"/>
      </svg>
      <span>Drop files to upload</span>
    </div>
  {/if}
  <div class="scp-toolbar">
    <button class="scp-btn" on:click={goUp} title="Go up">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="15 18 9 12 15 6"/>
      </svg>
    </button>
    <button class="scp-btn" on:click={() => loadDir(currentPath)} title="Refresh">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="23 4 23 10 17 10"/>
        <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
      </svg>
    </button>
    <button class="scp-btn" on:click={loadHome} title="Home">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
        <polyline points="9 22 9 12 15 12 15 22"/>
      </svg>
    </button>
    <button class="scp-btn" on:click={uploadFile} title="Upload file">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
        <polyline points="17 8 12 3 7 8"/>
        <line x1="12" y1="3" x2="12" y2="15"/>
      </svg>
    </button>
    <button
      class="scp-btn"
      on:click={() => showHidden = !showHidden}
      title={showHidden ? "Hide hidden files" : "Show hidden files"}
      class:active={showHidden}
    >
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
        <circle cx="12" cy="12" r="3"/>
      </svg>
    </button>
  </div>

  <div class="scp-path">
    <div class="breadcrumbs">
      {#each breadcrumbs as crumb, i}
        {#if i > 0}<span class="crumb-sep">/</span>{/if}
        <button class="crumb" on:click={() => loadDir(crumb.path)}>{crumb.name}</button>
      {/each}
    </div>
  </div>

  <div class="scp-table-wrapper">
    {#if loading && files.length === 0}
      <div class="scp-loading">Loading...</div>
    {:else if error}
      <div class="scp-error">{error}</div>
    {:else}
      <table class="scp-table">
        <thead>
          <tr>
            <th class="col-name" on:click={() => toggleSort("name")}>
              Name {sortColumn === "name" ? (sortAsc ? "▲" : "▼") : ""}
            </th>
            <th class="col-size" on:click={() => toggleSort("size")}>
              Size {sortColumn === "size" ? (sortAsc ? "▲" : "▼") : ""}
            </th>
            <th class="col-date" on:click={() => toggleSort("modTime")}>
              Modified {sortColumn === "modTime" ? (sortAsc ? "▲" : "▼") : ""}
            </th>
            <th class="col-actions"></th>
          </tr>
        </thead>
        <tbody>
          {#if currentPath !== "/"}
            <tr class="file-row" on:dblclick={goUp}>
              <td class="col-name">
                <span class="file-icon dir-icon">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="#6c7086" stroke="none">
                    <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
                  </svg>
                </span>
                <span class="file-name">..</span>
              </td>
              <td class="col-size"></td>
              <td class="col-date"></td>
              <td class="col-actions"></td>
            </tr>
          {/if}
          {#each sortedFiles as entry}
            <tr
              class="file-row"
              draggable={!entry.isDir}
              on:dragstart={(e) => handleDragStart(e, entry)}
              on:dragend={(e) => handleDragEnd(e, entry)}
              on:dblclick={() => handleDoubleClick(entry)}
            >
              <td class="col-name">
                <span class="file-icon" class:dir-icon={entry.isDir}>
                  {#if entry.isDir}
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="#6c7086" stroke="none">
                      <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
                    </svg>
                  {:else}
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#666" stroke-width="2">
                      <path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/>
                      <polyline points="13 2 13 9 20 9"/>
                    </svg>
                  {/if}
                </span>
                <span class="file-name" class:dir-name={entry.isDir}>{entry.name}</span>
              </td>
              <td class="col-size">{entry.isDir ? "" : formatSize(entry.size)}</td>
              <td class="col-date">{formatDate(entry.modTime)}</td>
              <td class="col-actions">
                {#if !entry.isDir}
                  <button class="dl-btn" on:click|stopPropagation={() => downloadFile(entry)} title="Download">
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                      <polyline points="7 10 12 15 17 10"/>
                      <line x1="12" y1="15" x2="12" y2="3"/>
                    </svg>
                  </button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>

  <div class="scp-footer">
    <label class="follow-label">
      <input type="checkbox" checked={followTerminal} on:change={toggleFollow} />
      <span>Follow terminal folder</span>
    </label>
    <span class="file-count">{sortedFiles.length} items</span>
  </div>
</div>
{/if}

<style>
  .scp-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #1a1b2e;
    width: 100%;
    overflow: hidden;
    position: relative;
  }

  .scp-panel.drop-hover {
    outline: 2px dashed #4a6cf7;
    outline-offset: -2px;
  }

  .drop-overlay {
    position: absolute;
    inset: 0;
    background: rgba(74, 108, 247, 0.1);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    z-index: 10;
    color: #4a6cf7;
    font-size: 13px;
    font-weight: 500;
    pointer-events: none;
  }

  .file-row[draggable="true"] {
    cursor: grab;
  }

  .file-row[draggable="true"]:active {
    cursor: grabbing;
  }

  .scp-toolbar {
    display: flex;
    align-items: center;
    gap: 2px;
    padding: 6px 8px;
    border-bottom: 1px solid #2a2b3d;
    flex-shrink: 0;
  }

  .scp-btn {
    background: none;
    border: none;
    color: #888;
    cursor: pointer;
    padding: 4px 6px;
    border-radius: 4px;
    display: flex;
    align-items: center;
  }

  .scp-btn:hover {
    color: #fff;
    background: #2a2b3d;
  }

  .scp-btn.active {
    color: #4a6cf7;
  }

  .scp-path {
    padding: 4px 8px;
    border-bottom: 1px solid #2a2b3d;
    flex-shrink: 0;
    overflow-x: auto;
  }

  .breadcrumbs {
    display: flex;
    align-items: center;
    gap: 0;
    font-size: 12px;
    white-space: nowrap;
  }

  .crumb {
    background: none;
    border: none;
    color: #4a6cf7;
    cursor: pointer;
    padding: 2px 4px;
    border-radius: 3px;
    font-size: 12px;
    font-family: 'Cascadia Code', 'Fira Code', monospace;
  }

  .crumb:hover {
    background: #2a2b3d;
    color: #6b8cf7;
  }

  .crumb:last-child {
    color: #e0e0e0;
  }

  .crumb-sep {
    color: #555;
    font-size: 11px;
  }

  .scp-table-wrapper {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
  }

  .scp-table-wrapper::-webkit-scrollbar {
    width: 6px;
  }

  .scp-table-wrapper::-webkit-scrollbar-track {
    background: transparent;
  }

  .scp-table-wrapper::-webkit-scrollbar-thumb {
    background: #3a3b4d;
    border-radius: 3px;
  }

  .scp-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 12px;
  }

  .scp-table thead {
    position: sticky;
    top: 0;
    z-index: 1;
  }

  .scp-table th {
    background: #1e1f33;
    color: #888;
    font-weight: 500;
    text-align: left;
    padding: 6px 8px;
    border-bottom: 1px solid #2a2b3d;
    cursor: pointer;
    user-select: none;
    white-space: nowrap;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }

  .scp-table th:hover {
    color: #ccc;
  }

  .col-name {
    min-width: 120px;
  }

  .col-size {
    width: 80px;
    text-align: right;
  }

  .col-date {
    width: 120px;
  }

  .col-actions {
    width: 30px;
  }

  .file-row {
    cursor: default;
    transition: background 0.1s;
  }

  .file-row:hover {
    background: #252640;
  }

  .file-row td {
    padding: 4px 8px;
    color: #ccc;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    border-bottom: 1px solid rgba(42, 43, 61, 0.5);
  }

  .col-name {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  td.col-size {
    text-align: right;
    color: #888;
    font-variant-numeric: tabular-nums;
  }

  td.col-date {
    color: #777;
    font-variant-numeric: tabular-nums;
  }

  .file-icon {
    display: flex;
    flex-shrink: 0;
  }

  .file-name {
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .dir-name {
    color: #6b8cf7;
  }

  .dl-btn {
    background: none;
    border: none;
    color: #666;
    cursor: pointer;
    padding: 2px;
    border-radius: 3px;
    display: flex;
    opacity: 0;
    transition: opacity 0.1s;
  }

  .file-row:hover .dl-btn {
    opacity: 1;
  }

  .dl-btn:hover {
    color: #4a6cf7;
    background: #2a2b3d;
  }

  .scp-loading, .scp-error {
    padding: 20px;
    text-align: center;
    color: #666;
    font-size: 13px;
  }

  .scp-error {
    color: #ff6b6b;
  }

  .scp-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 10px;
    border-top: 1px solid #2a2b3d;
    flex-shrink: 0;
  }

  .follow-label {
    display: flex;
    align-items: center;
    gap: 6px;
    cursor: pointer;
    font-size: 11px;
    color: #888;
  }

  .follow-label input[type="checkbox"] {
    width: 14px;
    height: 14px;
    accent-color: #4a6cf7;
  }

  .follow-label span {
    text-transform: none;
    font-weight: normal;
    letter-spacing: normal;
  }

  .file-count {
    font-size: 11px;
    color: #666;
    font-variant-numeric: tabular-nums;
  }
</style>
