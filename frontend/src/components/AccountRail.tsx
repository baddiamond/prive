import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { InboxIcon, SettingsIcon } from "./icons";

/** The main areas of the window. */
export type View = "inbox" | "settings";

interface AccountRailProps {
  view: View;
  onSelect: (view: View) => void;
}

/**
 * Narrow left column. It holds the unified inbox and settings today;
 * one entry per connected account joins it in later phases.
 */
export default function AccountRail({ view, onSelect }: AccountRailProps) {
  const { t } = useTranslation();

  return (
    <nav
      aria-label={t("nav.label")}
      className="flex w-16 shrink-0 flex-col items-center gap-2 border-r border-neutral-200 bg-neutral-50 py-3 dark:border-neutral-800 dark:bg-neutral-900"
    >
      <RailButton label={t("nav.inbox")} active={view === "inbox"} onClick={() => onSelect("inbox")}>
        <InboxIcon />
      </RailButton>
      <div className="flex-1" />
      <RailButton
        label={t("nav.settings")}
        active={view === "settings"}
        onClick={() => onSelect("settings")}
      >
        <SettingsIcon />
      </RailButton>
    </nav>
  );
}

interface RailButtonProps {
  label: string;
  active: boolean;
  onClick: () => void;
  children: ReactNode;
}

function RailButton({ label, active, onClick, children }: RailButtonProps) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      aria-current={active ? "page" : undefined}
      onClick={onClick}
      className={`flex h-10 w-10 items-center justify-center rounded-xl transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-500 ${
        active
          ? "bg-indigo-600 text-white"
          : "text-neutral-500 hover:bg-neutral-200 hover:text-neutral-900 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-100"
      }`}
    >
      {children}
    </button>
  );
}
