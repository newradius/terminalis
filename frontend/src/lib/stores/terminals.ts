import { writable, derived } from "svelte/store";
import type { Tab } from "../types";

export const tabs = writable<Tab[]>([]);
export const activeTabId = writable<string | null>(null);

export const activeTab = derived([tabs, activeTabId], ([$tabs, $activeTabId]) => {
  return $tabs.find((t) => t.id === $activeTabId) || null;
});

let tabCounter = 0;

export function createTab(title: string, sessionId?: string): string {
  const id = `tab-${Date.now()}-${tabCounter++}`;
  const tab: Tab = {
    id,
    title,
    sessionId,
    connected: false,
  };
  tabs.update((t) => [...t, tab]);
  activeTabId.set(id);
  return id;
}

export function closeTab(tabId: string) {
  tabs.update((t) => {
    const filtered = t.filter((tab) => tab.id !== tabId);
    return filtered;
  });
  activeTabId.update((current) => {
    if (current === tabId) {
      let currentTabs: Tab[] = [];
      tabs.subscribe((t) => (currentTabs = t))();
      return currentTabs.length > 0 ? currentTabs[currentTabs.length - 1].id : null;
    }
    return current;
  });
}

export function setTabConnected(tabId: string, connected: boolean) {
  tabs.update((t) =>
    t.map((tab) => (tab.id === tabId ? { ...tab, connected } : tab))
  );
}
