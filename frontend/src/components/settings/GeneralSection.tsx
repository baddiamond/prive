import { useId } from "react";
import { useTranslation } from "react-i18next";
import { languageCodes } from "../../i18n";
import type { ThemePreference } from "../../lib/api";
import { useSettings, type LanguagePreference } from "../../state/settings";

const themeOptions: readonly ThemePreference[] = ["system", "light", "dark"];

/** Language and theme. Both apply instantly and are saved on this computer. */
export default function GeneralSection() {
  const { t } = useTranslation();
  const language = useSettings((s) => s.language);
  const setLanguage = useSettings((s) => s.setLanguage);
  const theme = useSettings((s) => s.theme);
  const setTheme = useSettings((s) => s.setTheme);
  const languageId = useId();
  const themeId = useId();

  const languages: { value: LanguagePreference; label: string }[] = [
    { value: "", label: t("settings.language.system") },
    ...languageCodes.map((code) => ({ value: code, label: t(`settings.language.options.${code}`) })),
  ];

  return (
    <>
      <section aria-labelledby={languageId} className="flex flex-col gap-3">
        <div>
          <h2 id={languageId} className="text-body-strong font-semibold">
            {t("settings.language.title")}
          </h2>
          <p className="mt-1 text-text-secondary">{t("settings.language.description")}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          {languages.map((l) => {
            const selected = language === l.value;
            return (
              <button
                key={l.value || "system"}
                type="button"
                aria-pressed={selected}
                onClick={() => void setLanguage(l.value)}
                className={`h-10 rounded-control border px-4 font-medium text-text transition-colors duration-fast ease-prive ${
                  selected ? "border-accent bg-accent-tint" : "border-border hover:bg-raised"
                }`}
              >
                {l.label}
              </button>
            );
          })}
        </div>
      </section>

      <section aria-labelledby={themeId} className="flex flex-col gap-3">
        <h2 id={themeId} className="text-body-strong font-semibold">
          {t("settings.theme.title")}
        </h2>
        <div className="inline-flex gap-1 self-start rounded-control bg-raised p-1">
          {themeOptions.map((option) => {
            const selected = theme === option;
            return (
              <button
                key={option}
                type="button"
                aria-pressed={selected}
                onClick={() => void setTheme(option)}
                className={`h-[34px] rounded-control px-4 font-medium transition-colors duration-fast ease-prive ${
                  selected ? "bg-text text-bg" : "text-text-secondary hover:text-text"
                }`}
              >
                {t(`settings.theme.options.${option}`)}
              </button>
            );
          })}
        </div>
      </section>
    </>
  );
}
