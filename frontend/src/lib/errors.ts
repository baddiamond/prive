import type { TFunction } from "i18next";
import i18n from "../i18n";

/**
 * Extracts the error code from a rejected backend call. Wails rejects with the Go
 * error's message, which for Privé is always a translation key (see ADR 0004).
 */
export function errorCode(err: unknown): string {
  if (typeof err === "string") return err;
  if (err instanceof Error) return err.message;
  return "errors.unknown";
}

/** Translates a backend code, falling back to errors.unknown when the key does not exist. */
export function translateCode(t: TFunction, code: string): string {
  if (code.startsWith("errors.") && i18n.exists(code)) {
    // The key comes from the backend at runtime, so it cannot be checked at compile time.
    return t(code as "errors.unknown");
  }
  return t("errors.unknown");
}
