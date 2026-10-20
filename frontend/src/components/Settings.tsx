import type { ReactNode } from "react";
import { useId } from "react";
import { useTranslation } from "react-i18next";
import { languageCodes, resolveLanguage, isLanguageCode } from "../i18n";
import type { ThemePreference } from "../lib/api";
import { translateCode } from "../lib/errors";
import { useSettings } from "../state/settings";

const themeOptions: readonly ThemePreference[] = ["system", "light", "dark"];

/** Settings screen: language and appearance. */
export default function Settings() {
  const { t } = useTranslation();
  const errorCode = useSettings((s) => s.errorCode);

  return (
    <div className="flex-1 overflow-y-auto">
      <div className="mx-auto flex max-w-xl flex-col gap-8 px-8 py-10">
        <h1 className="text-2xl font-semibold">{t("settings.title")}</h1>
        {errorCode && (
          <p
            role="alert"
            className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-950 dark:text-red-300"
          >
            {translateCode(t, errorCode)}
          </p>
        )}
        <LanguageSection />
        <ThemeSection />
      </div>
    </div>
  );
}

function LanguageSection() {
  const { t } = useTranslation();
  const language = useSettings((s) => s.language);
  const systemLocale = useSettings((s) => s.systemLocale);
  const setLanguage = useSettings((s) => s.setLanguage);
  const selectId = useId();

  const systemLanguage = resolveLanguage("", systemLocale);

  return (
    <Section
      title={t("settings.language.title")}
      description={t("settings.language.description")}
      labelFor={selectId}
    >
      <select
        id={selectId}
        value={language}
        onChange={(e) => {
          const value = e.target.value;
          void setLanguage(isLanguageCode(value) ? value : "");
        }}
        className="w-full rounded-lg border border-neutral-300 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900"
      >
        <option value="">
          {t("settings.language.system", {
            language: t(`settings.language.options.${systemLanguage}`),
          })}
        </option>
        {languageCodes.map((code) => (
          <option key={code} value={code}>
            {t(`settings.language.options.${code}`)}
          </option>
        ))}
      </select>
    </Section>
  );
}

function ThemeSection() {
  const { t } = useTranslation();
  const theme = useSettings((s) => s.theme);
  const setTheme = useSettings((s) => s.setTheme);
  const titleId = useId();

  return (
    <Section
      title={t("settings.theme.title")}
      description={t("settings.theme.description")}
      titleId={titleId}
    >
      <div
        role="radiogroup"
        aria-labelledby={titleId}
        className="inline-flex self-start rounded-lg border border-neutral-300 p-1 dark:border-neutral-700"
      >
        {themeOptions.map((option) => (
          <button
            key={option}
            type="button"
            role="radio"
            aria-checked={theme === option}
            onClick={() => void setTheme(option)}
            className={`rounded-md px-4 py-1.5 text-sm transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-indigo-500 ${
              theme === option
                ? "bg-indigo-600 text-white"
                : "text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
            }`}
          >
            {t(`settings.theme.options.${option}`)}
          </button>
        ))}
      </div>
    </Section>
  );
}

interface SectionProps {
  title: string;
  description: string;
  /** id of the form control the title labels, if any. */
  labelFor?: string;
  /** id given to the title, so a group can reference it with aria-labelledby. */
  titleId?: string;
  children: ReactNode;
}

function Section({ title, description, labelFor, titleId, children }: SectionProps) {
  return (
    <section className="flex flex-col gap-3">
      <div>
        {labelFor ? (
          <label htmlFor={labelFor} className="text-base font-medium">
            {title}
          </label>
        ) : (
          <h2 id={titleId} className="text-base font-medium">
            {title}
          </h2>
        )}
        <p className="text-sm text-neutral-500 dark:text-neutral-400">{description}</p>
      </div>
      {children}
    </section>
  );
}
