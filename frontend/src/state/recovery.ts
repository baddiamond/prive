import { create } from "zustand";
import { getRecoveryStatus } from "../lib/recovery";

/** How the recovery dialog opens: "save" starts with the explanation, "show" with the words. */
export type RecoveryDialogMode = "save" | "show";

interface RecoveryState {
  loaded: boolean;
  /** True once the user typed back some of the words. */
  confirmed: boolean;
  /** "Later" hides the banner until the next start (ADR 0007). */
  postponed: boolean;
  dialog: RecoveryDialogMode | null;
  /** Loads the status from the backend. */
  load: () => Promise<void>;
  postpone: () => void;
  openDialog: (mode: RecoveryDialogMode) => void;
  closeDialog: () => void;
  markConfirmed: () => void;
}

/** Recovery phrase state: banner, dialog and whether the phrase was saved. */
export const useRecovery = create<RecoveryState>()((set) => ({
  loaded: false,
  confirmed: false,
  postponed: false,
  dialog: null,

  load: async () => {
    const s = await getRecoveryStatus();
    set({ loaded: true, confirmed: s.confirmed });
  },
  postpone: () => set({ postponed: true }),
  openDialog: (mode) => set({ dialog: mode }),
  closeDialog: () => set({ dialog: null }),
  markConfirmed: () => set({ confirmed: true }),
}));
