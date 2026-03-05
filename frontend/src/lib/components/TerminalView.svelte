<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { Terminal } from "@xterm/xterm";
  import { FitAddon } from "@xterm/addon-fit";
  import { WebLinksAddon } from "@xterm/addon-web-links";
  import { SendInput, ResizeTerminal, DisconnectTab } from "../../../wailsjs/go/main/App";
  import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime";
  import { setTabConnected } from "../stores/terminals";
  import { appConfig } from "../stores/config";

  export let tabId: string;
  export let active: boolean = false;

  let container: HTMLDivElement;
  let terminal: Terminal;
  let fitAddon: FitAddon;
  let resizeObserver: ResizeObserver;

  onMount(() => {
    terminal = new Terminal({
      cursorBlink: true,
      fontSize: $appConfig.fontSize,
      fontFamily: "'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace",
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
    fitAddon.fit();

    // Send input to backend (encode string as UTF-8 bytes, then base64)
    terminal.onData((data) => {
      const bytes = new TextEncoder().encode(data);
      let binary = "";
      for (let i = 0; i < bytes.length; i++) {
        binary += String.fromCharCode(bytes[i]);
      }
      SendInput(tabId, btoa(binary)).catch(() => {});
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
</script>

<div class="terminal-container" class:visible={active} bind:this={container}></div>

<style>
  .terminal-container {
    width: 100%;
    height: 100%;
    display: none;
  }

  .terminal-container.visible {
    display: block;
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
</style>
