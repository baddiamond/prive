import type { TFunction } from "i18next";
import i18n from "../i18n";

/** An error as reported by the backend: a translation key plus interpolation values (ADR 0004). */
export interface BackendError {
  code: string;
  params?: Record<string, string>;
}

const unknownError: BackendError = { code: "errors.unknown" };

/**
 * Normalizes a rejected backend call. Bound Go methods reject with an Error whose message is
 * the JSON text of {code, params} (see formatError in app.go); anything else becomes errors.unknown.
 */
export function toBackendError(err: unknown): BackendError {
  const text = err instanceof Error ? err.message : typeof err === "string" ? err : "";
  try {
    const parsed: unknown = JSON.parse(text);
    if (typeof parsed === "object" && parsed !== null && "code" in parsed && typeof parsed.code === "string") {
      const params =
        "params" in parsed && typeof parsed.params === "object" && parsed.params !== null
          ? (parsed.params as Record<string, string>)
          : undefined;
      return { code: parsed.code, params };
    }
  } catch {
    // not JSON: fall through
  }
  return unknownError;
}

/** Returns just the translation key of a rejected backend call. */
export function errorCode(err: unknown): string {
  return toBackendError(err).code;
}

/** Translates a backend code, falling back to errors.unknown when the key does not exist. */
export function translateCode(t: TFunction, code: string, params?: Record<string, string>): string {
  if (code.startsWith("errors.") && i18n.exists(code)) {
    // The key comes from the backend at runtime, so it cannot be type-checked like
    // literal keys; i18n.exists above guarantees it is real.
    const translateDynamic = t as unknown as (key: string, options?: Record<string, string>) => string;
    return translateDynamic(code, params);
  }
  return t("errors.unknown");
}
