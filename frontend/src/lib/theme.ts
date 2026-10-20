import type { ThemePreference } from "./api";

const darkQuery = "(prefers-color-scheme: dark)";

/**
 * Applies a theme to the document by toggling the "dark" class on <html>.
 * With "system" it also follows OS changes live. Returns a cleanup function.
 */
export function applyTheme(preference: ThemePreference): () => void {
  const root = document.documentElement;
  const media = window.matchMedia(darkQuery);

  const update = () => {
    const dark = preference === "dark" || (preference === "system" && media.matches);
    root.classList.toggle("dark", dark);
    root.style.colorScheme = dark ? "dark" : "light";
  };

  update();
  if (preference !== "system") return () => {};
  media.addEventListener("change", update);
  return () => media.removeEventListener("change", update);
}
