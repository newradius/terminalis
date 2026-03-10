<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { Terminal } from "@xterm/xterm";
  import { FitAddon } from "@xterm/addon-fit";
  import { WebLinksAddon } from "@xterm/addon-web-links";
  import { SendInput, ResizeTerminal, DisconnectTab, SaveCommandToHistory, GetCommandHistory, FetchRemoteCompletions } from "../../../wailsjs/go/main/App";
  import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime";
  import { setTabConnected } from "../stores/terminals";
  import { appConfig } from "../stores/config";

  export let tabId: string;
  export let active: boolean = false;
  export let historyId: string = "";

  const TERMINAL_FONT = "'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace";
  const XTERM_PADDING = 4;

  let container: HTMLDivElement;
  let terminal: Terminal;
  let fitAddon: FitAddon;
  let resizeObserver: ResizeObserver;

  // Autosuggestion state
  let commandHistory: string[] = [];
  let remoteHistory: string[] = [];
  let remoteCommands: string[] = [];
  let currentInput = "";
  let currentSuggestion = "";

  function updateSuggestion() {
    if (!currentInput) {
      currentSuggestion = "";
      return;
    }

    const isFirstWord = !currentInput.includes(" ");

    // Search local history first (full line match)
    let match = commandHistory.find((h) =>
      h.startsWith(currentInput) && h !== currentInput
    );

    // Then remote history (full line match)
    if (!match) {
      match = remoteHistory.find((h) =>
        h.startsWith(currentInput) && h !== currentInput
      );
    }

    // Then remote commands (first word only)
    if (!match && isFirstWord) {
      const cmdMatch = remoteCommands.find((c) =>
        c.startsWith(currentInput) && c !== currentInput
      );
      if (cmdMatch) {
        match = cmdMatch;
      }
    }

    currentSuggestion = match ? match.slice(currentInput.length) : "";
  }

  let ghostLeft = 0;
  let ghostTop = 0;

  function updateGhostPosition() {
    if (!terminal || !currentSuggestion) return;
    const cursorX = terminal.buffer.active.cursorX;
    const cursorY = terminal.buffer.active.cursorY;
    const renderer = (terminal as any)._core._renderService;
    const cellWidth = renderer?.dimensions?.css?.cell?.width || ($appConfig.fontSize * 0.6);
    const cellHeight = renderer?.dimensions?.css?.cell?.height || ($appConfig.fontSize * 1.2);
    ghostLeft = cursorX * cellWidth + XTERM_PADDING;
    ghostTop = cursorY * cellHeight + XTERM_PADDING;
  }

  onMount(() => {
    terminal = new Terminal({
      cursorBlink: true,
      fontSize: $appConfig.fontSize,
      fontFamily: TERMINAL_FONT,
      theme: {
        background: $appConfig.terminalBackground,
        foreground: "#e0e0e0",
        cursor: "#4a6cf7",
        selectionBackground: "#4a6cf744",
        black: "#1a1b2e",
        red: "#ff6b6b",
        green: "#45a049",
        yellow: "#ffd93d",
        blue: "#4a6cf7",
        magenta: "#c678dd",
        cyan: "#56b6c2",
        white: "#e0e0e0",
        brightBlack: "#5c5c7a",
        brightRed: "#ff8787",
        brightGreen: "#66bb6a",
        brightYellow: "#ffe066",
        brightBlue: "#6b8cf7",
        brightMagenta: "#d19aff",
        brightCyan: "#79c8d2",
        brightWhite: "#ffffff",
      },
      scrollback: 10000,
      allowProposedApi: true,
    });

    fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.loadAddon(new WebLinksAddon());

    terminal.open(container);

    // Accept suggestion with Right Arrow or End key
    terminal.attachCustomKeyEventHandler((e: KeyboardEvent) => {
      if (!currentSuggestion) return true;
      if (e.type !== "keydown") return true;

      if (e.key === "ArrowRight" || e.key === "End") {
        e.preventDefault();
        const bytes = new TextEncoder().encode(currentSuggestion);
        let binary = "";
        for (let i = 0; i < bytes.length; i++) {
          binary += String.fromCharCode(bytes[i]);
        }
        SendInput(tabId, btoa(binary)).catch(() => {});
        currentInput += currentSuggestion;
        currentSuggestion = "";
        return false;
      }

      return true;
    });

    fitAddon.fit();

    // Send input to backend (encode string as UTF-8 bytes, then base64)
    terminal.onData((data) => {
      // Send raw data to backend (unchanged)
      const bytes = new TextEncoder().encode(data);
      let binary = "";
      for (let i = 0; i < bytes.length; i++) {
        binary += String.fromCharCode(bytes[i]);
      }
      SendInput(tabId, btoa(binary)).catch(() => {});

      // Track input buffer for autosuggestions
      if (data === "\r" || data === "\n") {
        const cmd = currentInput.trim();
        if (cmd && historyId) {
          SaveCommandToHistory(historyId, cmd).catch(() => {});
          commandHistory = commandHistory.filter((h) => h !== cmd);
          commandHistory.unshift(cmd);
        }
        currentInput = "";
        currentSuggestion = "";
      } else if (data === "\x7f" || data === "\b") {
        currentInput = currentInput.slice(0, -1);
        updateSuggestion();
      } else if (data === "\x03") {
        currentInput = "";
        currentSuggestion = "";
      } else if (data === "\x15") {
        currentInput = "";
        currentSuggestion = "";
      } else if (data === "\x17") {
        currentInput = currentInput.trimEnd().replace(/\S+$/, "").trimEnd();
        updateSuggestion();
      } else if (data.length > 1 && (data.charCodeAt(0) === 27 || data.length > 4)) {
        if (data.charCodeAt(0) === 27) {
          currentInput = "";
          currentSuggestion = "";
        } else {
          currentInput += data;
          updateSuggestion();
        }
      } else if (data >= " ") {
        currentInput += data;
        updateSuggestion();
      } else {
        currentInput = "";
        currentSuggestion = "";
      }
    });

    // Receive output from backend (decode base64 to raw bytes for proper UTF-8)
    EventsOn("terminal:data:" + tabId, (data: string) => {
      const binary = atob(data);
      const bytes = new Uint8Array(binary.length);
      for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i);
      }
      terminal.write(bytes);
    });

    // Session closed
    EventsOn("terminal:closed:" + tabId, (reason: string) => {
      terminal.write("\r\n\x1b[31m--- Session closed: " + reason + " ---\x1b[0m\r\n");
      setTabConnected(tabId, false);
    });

    // Right-click: if text selected, copy+paste it; otherwise paste from clipboard.
    // Capture selection on mousedown because xterm clears it before contextmenu fires.
    let pendingSelection = "";
    container.addEventListener("mousedown", (e) => {
      if (e.button === 2) {
        pendingSelection = terminal.hasSelection() ? terminal.getSelection() : "";
      }
    });
    container.addEventListener("contextmenu", async (e) => {
      e.preventDefault();
      if (pendingSelection) {
        await navigator.clipboard.writeText(pendingSelection);
        terminal.paste(pendingSelection);
        pendingSelection = "";
      } else {
        try {
          const text = await navigator.clipboard.readText();
          if (text) {
            terminal.paste(text);
          }
        } catch {}
      }
    });

    // Connection established
    setTabConnected(tabId, true);

    // Load command history for autosuggestions
    if (historyId) {
      GetCommandHistory(historyId).then((h) => {
        commandHistory = h || [];
      });
    }

    // Fetch remote completions for SSH sessions (non-blocking)
    FetchRemoteCompletions(tabId).then((r) => {
      if (r) {
        remoteHistory = r.history || [];
        remoteCommands = r.commands || [];
      }
    }).catch(() => {});

    // Handle resize
    resizeObserver = new ResizeObserver(() => {
      if (active) {
        fitAddon.fit();
        ResizeTerminal(tabId, terminal.cols, terminal.rows).catch(() => {});
      }
    });
    resizeObserver.observe(container);
  });

  onDestroy(() => {
    EventsOff("terminal:data:" + tabId);
    EventsOff("terminal:closed:" + tabId);
    if (resizeObserver) resizeObserver.disconnect();
    if (terminal) terminal.dispose();
    DisconnectTab(tabId);
  });

  // React to visibility changes
  $: if (active && fitAddon && terminal) {
    setTimeout(() => {
      fitAddon.fit();
      terminal.focus();
      ResizeTerminal(tabId, terminal.cols, terminal.rows).catch(() => {});
    }, 50);
  }

  // React to config changes (font size, background)
  $: if (terminal && fitAddon && $appConfig) {
    terminal.options.fontSize = $appConfig.fontSize;
    terminal.options.theme = { ...terminal.options.theme, background: $appConfig.terminalBackground };
    fitAddon.fit();
  }

  // Update ghost position when suggestion changes
  $: if (currentSuggestion && terminal) {
    setTimeout(updateGhostPosition, 0);
  }
</script>

<div class="terminal-wrapper" class:visible={active}>
  <div class="terminal-container" bind:this={container}></div>
  {#if currentSuggestion}
    <div
      class="ghost-suggestion"
      style="left: {ghostLeft}px; top: {ghostTop}px; font-size: {$appConfig.fontSize}px; font-family: {TERMINAL_FONT};"
    >
      {currentSuggestion}
    </div>
  {/if}
</div>

<style>
  .terminal-wrapper {
    width: 100%;
    height: 100%;
    display: none;
    position: relative;
  }

  .terminal-wrapper.visible {
    display: block;
  }

  .terminal-container {
    width: 100%;
    height: 100%;
  }

  .terminal-container :global(.xterm) {
    height: 100%;
    padding: 4px;
  }

  .terminal-container :global(.xterm-viewport) {
    overflow-y: auto !important;
  }

  .terminal-container :global(.xterm-viewport::-webkit-scrollbar) {
    width: 8px;
  }

  .terminal-container :global(.xterm-viewport::-webkit-scrollbar-track) {
    background: transparent;
  }

  .terminal-container :global(.xterm-viewport::-webkit-scrollbar-thumb) {
    background: #3a3b4d;
    border-radius: 4px;
  }

  .ghost-suggestion {
    position: absolute;
    color: #555;
    pointer-events: none;
    white-space: pre;
    z-index: 1;
    opacity: 0.5;
  }
</style>
