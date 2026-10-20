import { create } from "zustand";

/** Top-level screens of the window. */
export type Screen = "inbox" | "settings";

/** Sections of the Settings screen (docs/UI.md §2.7). */
export type SettingsSection = "general" | "accounts" | "privacy" | "about";

/** Chat list filters (docs/UI.md §2.2). */
export type ChatFilter = "all" | "unread" | "groups";

/** The add-account dialog: closed, or open to add a new account or link an existing one again. */
export type AddAccountDialog = { open: false } | { open: true; relinkAccountId?: string; engineId?: string };

interface UIState {
  screen: Screen;
  settingsSection: SettingsSection;
  chatFilter: ChatFilter;
  /** "all" for the unified inbox, or an account id. */
  scope: string;
  addAccount: AddAccountDialog;
  /** Opens Settings, optionally on a given section. */
  openSettings: (section?: SettingsSection) => void;
  /** Returns to the inbox. */
  closeSettings: () => void;
  setSettingsSection: (section: SettingsSection) => void;
  setChatFilter: (filter: ChatFilter) => void;
  /** Shows the unified inbox or one account's inbox. */
  setScope: (scope: string) => void;
  /** Opens the add-account dialog; with an account id it links that account again. */
  openAddAccount: (relinkAccountId?: string, engineId?: string) => void;
  closeAddAccount: () => void;
}

/** Navigation state that is not persisted. */
export const useUI = create<UIState>()((set) => ({
  screen: "inbox",
  settingsSection: "general",
  chatFilter: "all",
  scope: "all",
  addAccount: { open: false },
  openSettings: (section) =>
    set((s) => ({ screen: "settings", settingsSection: section ?? s.settingsSection })),
  closeSettings: () => set({ screen: "inbox" }),
  setSettingsSection: (settingsSection) => set({ settingsSection }),
  setChatFilter: (chatFilter) => set({ chatFilter }),
  setScope: (scope) => set({ scope, screen: "inbox" }),
  openAddAccount: (relinkAccountId, engineId) => set({ addAccount: { open: true, relinkAccountId, engineId } }),
  closeAddAccount: () => set({ addAccount: { open: false } }),
}));
