import { useTranslation } from "react-i18next";
import { translateCode } from "../../lib/errors";
import { useSettings } from "../../state/settings";
import { useUI, type SettingsSection } from "../../state/ui";
import { BackIcon } from "../icons";
import AboutSection from "./AboutSection";
import AccountsSection from "./AccountsSection";
import GeneralSection from "./GeneralSection";
import PrivacySection from "./PrivacySection";

const sections: readonly SettingsSection[] = ["general", "accounts", "privacy", "about"];

/** Full-window Settings with a section list (docs/UI.md §2.7, design/prototypes/ui/Settings.dc.html). */
export default function SettingsScreen() {
  const { t } = useTranslation();
  const section = useUI((s) => s.settingsSection);
  const setSection = useUI((s) => s.setSettingsSection);
  const closeSettings = useUI((s) => s.closeSettings);
  const errorCode = useSettings((s) => s.errorCode);

  return (
    <div className="flex min-w-0 flex-1">
      <nav
        aria-label={t("settings.sections")}
        className="flex w-[240px] shrink-0 flex-col gap-1 border-r border-divider bg-surface px-3 py-6"
      >
        <button
          type="button"
          onClick={closeSettings}
          className="mb-3 flex items-center gap-2 rounded-control px-3 py-2 text-left text-caption text-text-secondary hover:text-text"
        >
          <BackIcon size={16} />
          {t("settings.back")}
        </button>
        {sections.map((s) => {
          const current = s === section;
          return (
            <button
              key={s}
              type="button"
              aria-current={current ? "page" : undefined}
              onClick={() => setSection(s)}
              className={`rounded-control px-3 py-2.5 text-left transition-colors duration-fast ease-prive ${
                current ? "bg-raised font-semibold text-text" : "font-medium text-text-secondary hover:text-text"
              }`}
            >
              {t(`settings.nav.${s}`)}
            </button>
          );
        })}
      </nav>

      <main className="min-w-0 flex-1 overflow-y-auto bg-bg">
        <div className="flex max-w-[880px] flex-col gap-7 px-10 py-8">
          <h1 className="text-title font-semibold tracking-[-0.01em]">{t(`settings.nav.${section}`)}</h1>
          {errorCode && (
            <p role="alert" className="rounded-control border border-status-danger px-3 py-2 text-status-danger">
              {translateCode(t, errorCode)}
            </p>
          )}
          {section === "general" && <GeneralSection />}
          {section === "accounts" && <AccountsSection />}
          {section === "privacy" && <PrivacySection />}
          {section === "about" && <AboutSection />}
        </div>
      </main>
    </div>
  );
}
