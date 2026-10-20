import { useTranslation } from "react-i18next";
import { shortcutLabel } from "../lib/platform";
import { useUI } from "../state/ui";
import { InboxIcon, PlusIcon, SettingsIcon } from "./icons";
import LogoMark from "./LogoMark";

/**
 * Left column (docs/UI.md §2.1): logo, "All inboxes", one item per account (from phase 2),
 * add account and settings.
 */
export default function AccountRail() {
  const { t } = useTranslation();
  const screen = useUI((s) => s.screen);
  const closeSettings = useUI((s) => s.closeSettings);
  const openSettings = useUI((s) => s.openSettings);

  return (
    <nav
      aria-label={t("rail.label")}
      className="flex w-[76px] shrink-0 flex-col items-center gap-3 border-r border-divider bg-rail py-4"
    >
      <LogoMark width={30} height={30} className="mb-1" />
      <div className="h-px w-9 bg-divider" />

      <button
        type="button"
        aria-label={t("rail.allInboxes")}
        title={t("rail.allInboxes")}
        aria-pressed={screen === "inbox"}
        onClick={closeSettings}
        className={`flex h-12 w-12 items-center justify-center rounded-card text-text transition-colors duration-fast ease-prive ${
          screen === "inbox" ? "bg-raised shadow-[0_0_0_2px_var(--prive-rail),0_0_0_4px_var(--prive-accent)]" : "hover:bg-raised"
        }`}
      >
        <InboxIcon size={22} />
      </button>

      <button
        type="button"
        aria-label={t("rail.addAccount")}
        title={`${t("rail.addAccount")} · ${t("common.nextVersion")}`}
        disabled
        className="flex h-12 w-12 items-center justify-center rounded-card border-[1.5px] border-dashed border-border text-text-secondary disabled:cursor-not-allowed disabled:opacity-60"
      >
        <PlusIcon />
      </button>

      <div className="flex-1" />

      <button
        type="button"
        aria-label={t("rail.settings")}
        title={`${t("rail.settings")} (${shortcutLabel(",")})`}
        aria-pressed={screen === "settings"}
        onClick={() => openSettings()}
        className={`flex h-11 w-11 items-center justify-center rounded-control transition-colors duration-fast ease-prive ${
          screen === "settings" ? "bg-raised text-text" : "text-text-secondary hover:bg-raised hover:text-text"
        }`}
      >
        <SettingsIcon />
      </button>
    </nav>
  );
}
