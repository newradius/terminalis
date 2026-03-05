import { writable } from "svelte/store";
import { GetConfig, SaveConfig } from "../../../wailsjs/go/main/App";

export interface AppConfig {
  fontSize: number;
  theme: string;
  defaultPort: number;
  defaultUsername: string;
  connectTimeout: number;
  scrollbackLines: number;
  terminalBackground: string;
}

const defaults: AppConfig = {
  fontSize: 14,
  theme: "dark",
  defaultPort: 22,
  defaultUsername: "root",
  connectTimeout: 30,
  scrollbackLines: 10000,
  terminalBackground: "#14142a",
};

export const appConfig = writable<AppConfig>(defaults);

export async function loadConfig() {
  try {
    const cfg = await GetConfig();
    appConfig.set({
      ...defaults,
      ...cfg,
      terminalBackground: cfg.terminalBackground || defaults.terminalBackground,
    });
  } catch {
    appConfig.set(defaults);
  }
}

export async function saveConfig(cfg: AppConfig) {
  await SaveConfig(cfg as any);
  appConfig.set(cfg);
}
