import { create } from "zustand";
import i18n, { isLanguageCode, resolveLanguage, type LanguageCode } from "../i18n";
import {
  getSettings,
  getSystemLocale,
  saveLanguage,
  saveTheme,
  type ThemePreference,
} from "../lib/api";
import { errorCode } from "../lib/errors";

/** Saved language preference: a supported code, or "" to follow the OS. */
export type LanguagePreference = LanguageCode | "";

interface SettingsState {
  status: "loading" | "ready" | "error";
  language: LanguagePreference;
  systemLocale: string;
  theme: ThemePreference;
  /** Translation key of the last failed operation, if any. */
  errorCode: string | null;

  /** Loads preferences from the backend and applies the language. */
  load: () => Promise<void>;
  /** Changes and persists the UI language. */
  setLanguage: (language: LanguagePreference) => Promise<void>;
  /** Changes and persists the color theme. */
  setTheme: (theme: ThemePreference) => Promise<void>;
}

/** Global store for user preferences (language and theme). */
export const useSettings = create<SettingsState>()((set, get) => ({
  status: "loading",
  language: "",
  systemLocale: "",
  theme: "system",
  errorCode: null,

  load: async () => {
    try {
      const s = await getSettings();
      // A saved language that is no longer shipped falls back to following the OS.
      const language: LanguagePreference = isLanguageCode(s.language) ? s.language : "";
      set({ language, systemLocale: s.systemLocale, theme: s.theme });
      await applyLanguage(language, s.systemLocale);
      set({ status: "ready" });
    } catch (err) {
      // Explain the problem in the user's language even without settings.
      const systemLocale = await getSystemLocale().catch(() => "");
      await applyLanguage("", systemLocale);
      set({ status: "error", errorCode: errorCode(err), systemLocale });
    }
  },

  setLanguage: async (language) => {
    const previous = get().language;
    set({ language, errorCode: null });
    await applyLanguage(language, get().systemLocale);
    try {
      await saveLanguage(language);
    } catch (err) {
      set({ language: previous, errorCode: errorCode(err) });
      await applyLanguage(previous, get().systemLocale);
    }
  },

  setTheme: async (theme) => {
    const previous = get().theme;
    set({ theme, errorCode: null });
    try {
      await saveTheme(theme);
    } catch (err) {
      set({ theme: previous, errorCode: errorCode(err) });
    }
  },
}));

async function applyLanguage(saved: string, systemLocale: string): Promise<void> {
  const lng = resolveLanguage(saved, systemLocale);
  document.documentElement.lang = lng;
  await i18n.changeLanguage(lng);
}
