import type { TFunction } from "i18next";
import i18n from "../i18n";

/** An error as reported by the backend: a translation key plus interpolation values (ADR 0004). */
export interface BackendError {
  code: string;
  params?: Record<string, string>;
}

const unknownError: BackendError = { code: "errors.unknown" };

/**
 * Normalizes a rejected backend call. Bound Go methods reject with {code, params}
 * (see formatError in app.go); anything else becomes errors.unknown.
 */
export function toBackendError(err: unknown): BackendError {
  if (typeof err === "object" && err !== null && "code" in err && typeof err.code === "string") {
    const params = "params" in err && typeof err.params === "object" && err.params !== null
      ? (err.params as Record<string, string>)
      : undefined;
    return { code: err.code, params };
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
