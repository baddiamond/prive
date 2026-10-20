import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import en from "./locales/en.json";
import es from "./locales/es.json";

/** Every language Privé ships. To add one, see docs/I18N.md. */
export const supportedLanguages = {
  en: { translation: en },
  es: { translation: es },
} as const;

/** A language code Privé has translations for. */
export type LanguageCode = keyof typeof supportedLanguages;

/** Language used when neither the saved setting nor the OS language is supported. */
export const fallbackLanguage: LanguageCode = "en";

/** All supported language codes, in display order. */
export const languageCodes = Object.keys(supportedLanguages) as LanguageCode[];

/** Reports whether value is a supported language code. */
export function isLanguageCode(value: string): value is LanguageCode {
  return Object.hasOwn(supportedLanguages, value);
}

/**
 * Picks the UI language: the saved setting, else the OS locale, else English.
 * A regional tag ("es-MX") matches its base language ("es").
 */
export function resolveLanguage(saved: string, systemLocale: string): LanguageCode {
  for (const candidate of [saved, systemLocale]) {
    const match = matchLanguage(candidate);
    if (match) return match;
  }
  return fallbackLanguage;
}

function matchLanguage(tag: string): LanguageCode | undefined {
  if (!tag) return undefined;
  if (isLanguageCode(tag)) return tag;
  const base = tag.split("-")[0]?.toLowerCase() ?? "";
  return isLanguageCode(base) ? base : undefined;
}

// --- Compile-time check: every locale must have exactly the keys of en.json. ---
// `npm run typecheck` fails and names the missing or extra keys when they drift.

type DeepKeys<T> = T extends object
  ? {
      [K in keyof T & string]: T[K] extends object ? `${K}.${DeepKeys<T[K]>}` : K;
    }[keyof T & string]
  : never;

type KeyDiff<A, B> = Exclude<DeepKeys<A>, DeepKeys<B>> | Exclude<DeepKeys<B>, DeepKeys<A>>;

type SameKeysAsEnglish<L> = [KeyDiff<typeof en, L>] extends [never]
  ? true
  : `key missing or extra: ${KeyDiff<typeof en, L>}`;

export const localeKeysMatch: {
  [L in LanguageCode]: SameKeysAsEnglish<(typeof supportedLanguages)[L]["translation"]>;
} = { en: true, es: true };

// Until the backend reports the saved language, use the webview's language.
void i18n.use(initReactI18next).init({
  resources: supportedLanguages,
  lng: resolveLanguage("", navigator.language),
  fallbackLng: fallbackLanguage,
  supportedLngs: languageCodes,
  interpolation: { escapeValue: false }, // React already escapes output.
  returnNull: false,
});

export default i18n;
