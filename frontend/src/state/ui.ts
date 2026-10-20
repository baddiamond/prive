import { create } from "zustand";

/** Top-level screens of the window. */
export type Screen = "inbox" | "settings";

/** Sections of the Settings screen (docs/UI.md §2.7). */
export type SettingsSection = "general" | "accounts" | "privacy" | "about";

/** Chat list filters (docs/UI.md §2.2). */
export type ChatFilter = "all" | "unread" | "groups";

interface UIState {
  screen: Screen;
  settingsSection: SettingsSection;
  chatFilter: ChatFilter;
  /** Opens Settings, optionally on a given section. */
  openSettings: (section?: SettingsSection) => void;
  /** Returns to the inbox. */
  closeSettings: () => void;
  setSettingsSection: (section: SettingsSection) => void;
  setChatFilter: (filter: ChatFilter) => void;
}

/** Navigation state that is not persisted. */
export const useUI = create<UIState>()((set) => ({
  screen: "inbox",
  settingsSection: "general",
  chatFilter: "all",
  openSettings: (section) =>
    set((s) => ({ screen: "settings", settingsSection: section ?? s.settingsSection })),
  closeSettings: () => set({ screen: "inbox" }),
  setSettingsSection: (settingsSection) => set({ settingsSection }),
  setChatFilter: (chatFilter) => set({ chatFilter }),
}));
