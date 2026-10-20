import { useTranslation } from "react-i18next";
import { shortcutLabel } from "../lib/platform";
import { linkedAccounts, useAccounts } from "../state/accounts";
import { useUI } from "../state/ui";
import type { AccountStatus } from "../lib/accounts";
import AccountTile from "./accounts/AccountTile";
import { accountLabel, hasIssue } from "./accounts/accountLabel";
import { InboxIcon, PlusIcon, SettingsIcon } from "./icons";
import LogoMark from "./LogoMark";

const selectedRing = "shadow-[0_0_0_2px_var(--prive-rail),0_0_0_4px_var(--prive-accent)]";

/** Left column (docs/UI.md §2.1): logo, "All inboxes", one item per account, add account, settings. */
export default function AccountRail() {
  const { t } = useTranslation();
  const screen = useUI((s) => s.screen);
  const scope = useUI((s) => s.scope);
  const setScope = useUI((s) => s.setScope);
  const openSettings = useUI((s) => s.openSettings);
  const openAddAccount = useUI((s) => s.openAddAccount);
  const accounts = linkedAccounts(useAccounts((s) => s.accounts));
  const inbox = screen === "inbox";

  return (
    <nav
      aria-label={t("rail.label")}
      className="flex w-[76px] shrink-0 flex-col items-center gap-3 overflow-y-auto border-r border-divider bg-rail py-4"
    >
      <LogoMark width={30} height={30} className="mb-1 shrink-0" />
      <div className="h-px w-9 shrink-0 bg-divider" />

      <button
        type="button"
        aria-label={t("rail.allInboxes")}
        title={`${t("rail.allInboxes")} (${shortcutLabel("0")})`}
        aria-pressed={inbox && scope === "all"}
        onClick={() => setScope("all")}
        className={`flex h-12 w-12 shrink-0 items-center justify-center rounded-card text-text transition-colors duration-fast ease-prive ${
          inbox && scope === "all" ? `bg-raised ${selectedRing}` : "hover:bg-raised"
        }`}
      >
        <InboxIcon size={22} />
      </button>

      {accounts.map((s, i) => (
        <RailAccount
          key={s.account.id}
          status={s}
          index={i}
          selected={inbox && scope === s.account.id}
          onSelect={() => setScope(s.account.id)}
        />
      ))}

      <button
        type="button"
        aria-label={t("rail.addAccount")}
        title={`${t("rail.addAccount")} (${shortcutLabel("N")})`}
        onClick={() => openAddAccount()}
        className="flex h-12 w-12 shrink-0 items-center justify-center rounded-card border-[1.5px] border-dashed border-border text-text-secondary hover:bg-raised hover:text-text"
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
        className={`flex h-11 w-11 shrink-0 items-center justify-center rounded-control transition-colors duration-fast ease-prive ${
          screen === "settings" ? "bg-raised text-text" : "text-text-secondary hover:bg-raised hover:text-text"
        }`}
      >
        <SettingsIcon />
      </button>
    </nav>
  );
}

interface RailAccountProps {
  status: AccountStatus;
  index: number;
  selected: boolean;
  onSelect: () => void;
}

function RailAccount({ status, index, selected, onSelect }: RailAccountProps) {
  const { t } = useTranslation();
  const label = accountLabel(status);
  const issue = hasIssue(status);
  // Accessible name: "{label} · {network}" plus the state when it needs attention.
  const name = [label, status.manifest.name, issue ? t(`account.state.${status.state}`) : null]
    .filter(Boolean)
    .join(" · ");
  const shortcut = index < 9 ? ` (${shortcutLabel(String(index + 1))})` : "";

  return (
    <button
      type="button"
      aria-label={name}
      title={name + shortcut}
      aria-pressed={selected}
      onClick={onSelect}
      className={`relative shrink-0 rounded-card ${selected ? selectedRing : ""}`}
    >
      <AccountTile label={label} color={status.account.color} />
      <span
        aria-hidden
        className="absolute -bottom-1 -right-1 flex h-4 min-w-[22px] items-center justify-center rounded-tag border border-border bg-rail px-0.5 text-micro font-bold leading-none text-text-secondary"
      >
        {status.manifest.shortName}
      </span>
      {issue && (
        <span
          aria-hidden
          className={`absolute -bottom-[5px] -left-[5px] flex h-[18px] w-[18px] items-center justify-center rounded-full border-2 border-rail text-micro font-extrabold leading-none text-rail ${
            status.state === "reconnecting" ? "bg-status-warning" : "bg-status-danger"
          }`}
        >
          !
        </span>
      )}
    </button>
  );
}
