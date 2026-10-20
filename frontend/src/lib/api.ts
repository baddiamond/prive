// Typed wrappers around the Go methods bound by Wails (see app.go).
// The generated bindings in ../../wailsjs are created by `wails dev` or `wails generate module`.
import {
  GetAppInfo,
  GetSettings,
  GetSystemLocale,
  SetLanguage,
  SetTheme,
} from "../../wailsjs/go/main/App";

/** Color theme preference. "system" follows the OS. */
export type ThemePreference = "system" | "light" | "dark";

/** User preferences as returned by the backend. */
export interface Settings {
  /** Saved UI language, or "" to follow the OS. */
  language: string;
  /** OS language as a BCP 47 tag, or "" if unknown. */
  systemLocale: string;
  theme: ThemePreference;
}

const themes: readonly ThemePreference[] = ["system", "light", "dark"];

/** Reports whether value is a valid theme preference. */
export function isThemePreference(value: string): value is ThemePreference {
  return (themes as readonly string[]).includes(value);
}

/** Loads the saved preferences and the detected OS language. */
export async function getSettings(): Promise<Settings> {
  const s = await GetSettings();
  return {
    language: s.language,
    systemLocale: s.systemLocale,
    theme: isThemePreference(s.theme) ? s.theme : "system",
  };
}

/** Saves the UI language; "" means follow the OS. */
export function saveLanguage(language: string): Promise<void> {
  return SetLanguage(language);
}

/** Saves the color theme. */
export function saveTheme(theme: ThemePreference): Promise<void> {
  return SetTheme(theme);
}

/** Static information shown in Settings. */
export interface AppInfo {
  version: string;
  /** Folder that holds every local file Privé writes. */
  dataDir: string;
}

/** Loads the app version and data folder. */
export function getAppInfo(): Promise<AppInfo> {
  return GetAppInfo();
}

/** Returns the OS language; works even when local data could not be opened. */
export function getSystemLocale(): Promise<string> {
  return GetSystemLocale();
}
