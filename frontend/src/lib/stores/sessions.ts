import { writable } from "svelte/store";
import type { TreeNode } from "../types";

export const sessionTree = writable<TreeNode[]>([]);
export const showSessionForm = writable(false);
export const editingSession = writable<string | null>(null);
export const showFolderForm = writable(false);
export const editingFolder = writable<string | null>(null);
export const selectedFolderId = writable<string>("");
export const showSettings = writable(false);
