/** True on macOS, where shortcuts use ⌘ instead of Ctrl. */
export const isMac = /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent);

/** Formats a shortcut for display: "⌘," on macOS, "Ctrl+," elsewhere. */
export function shortcutLabel(key: string): string {
  return isMac ? `⌘${key}` : `Ctrl+${key}`;
}

/** Reports whether a keyboard event uses the platform's command modifier. */
export function hasCommandKey(e: KeyboardEvent): boolean {
  return isMac ? e.metaKey : e.ctrlKey;
}
