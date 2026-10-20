import type { ThemePreference } from "./api";

/**
 * Applies a theme preference. The brand tokens (assets/brand/tokens/tokens.css) do the rest:
 * without a data-theme attribute they follow the OS (dark by default), and
 * data-theme="light" | "dark" forces a theme.
 */
export function applyTheme(preference: ThemePreference): void {
  const root = document.documentElement;
  if (preference === "system") {
    delete root.dataset.theme;
  } else {
    root.dataset.theme = preference;
  }
}
