import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { InboxIcon, SettingsIcon } from "./icons";
import LogoMark from "./LogoMark";

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
      className="flex w-[76px] shrink-0 flex-col items-center gap-2 border-r border-divider bg-rail py-3"
    >
      <div className="mb-2 flex h-11 w-11 items-center justify-center">
        <LogoMark height={30} />
      </div>
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
      className={`flex h-11 w-11 items-center justify-center rounded-card transition-colors duration-fast ease-prive ${
        active ? "bg-accent text-on-accent" : "text-text-secondary hover:bg-raised hover:text-text"
      }`}
    >
      {children}
    </button>
  );
}
