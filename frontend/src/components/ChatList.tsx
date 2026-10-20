import { useTranslation } from "react-i18next";
import { reconnectAccount, type AccountStatus } from "../lib/accounts";
import { translateCode } from "../lib/errors";
import { shortcutLabel } from "../lib/platform";
import { linkedAccounts, useAccounts } from "../state/accounts";
import { useUI, type ChatFilter } from "../state/ui";
import { accountLabel, hasIssue } from "./accounts/accountLabel";
import { ChatIcon, PlusIcon, RefreshIcon, SearchIcon } from "./icons";

const filters: readonly ChatFilter[] = ["all", "unread", "groups"];
const maxBanners = 2;

/**
 * Middle column (docs/UI.md §2.2): scope title, search, filters, account status banners
 * and the chats. Chats arrive with phase 3; until then it shows the empty states.
 */
export default function ChatList() {
  const { t } = useTranslation();
  const chatFilter = useUI((s) => s.chatFilter);
  const setChatFilter = useUI((s) => s.setChatFilter);
  const scope = useUI((s) => s.scope);
  const openAddAccount = useUI((s) => s.openAddAccount);
  const accounts = linkedAccounts(useAccounts((s) => s.accounts));
  const scoped = scope === "all" ? accounts : accounts.filter((s) => s.account.id === scope);
  const scopeAccount = scope === "all" ? undefined : scoped[0];
  const issues = scoped.filter(hasIssue);

  const title = scopeAccount ? accountLabel(scopeAccount) : t("chatList.allInboxes");
  const subtitle = scopeAccount
    ? scopeAccount.manifest.name
    : accounts.length === 0
      ? t("chatList.noAccounts")
      : t("chatList.accountsCount", { count: accounts.length });

  return (
    <section aria-label={t("chatList.label")} className="flex w-[340px] shrink-0 flex-col border-r border-divider bg-surface">
      <div className="flex flex-col gap-3 px-4 pb-3 pt-[18px]">
        <div className="flex items-baseline justify-between gap-3">
          <h1 className="truncate text-heading font-semibold tracking-[-0.01em]">{title}</h1>
          <span className="shrink-0 text-caption text-text-secondary">{subtitle}</span>
        </div>

        <button
          type="button"
          disabled
          title={t("common.nextVersion")}
          className="flex h-[38px] items-center gap-2 rounded-control border-[1.5px] border-border bg-raised px-3 text-left text-text-secondary disabled:cursor-not-allowed"
        >
          <SearchIcon size={16} />
          <span className="flex-1">{t("chatList.search")}</span>
          <kbd className="rounded-tag border border-border px-1.5 font-mono text-micro font-semibold">{shortcutLabel("K")}</kbd>
        </button>

        <div role="group" aria-label={t("chatList.filters.label")} className="flex gap-1.5">
          {filters.map((f) => {
            const selected = chatFilter === f;
            return (
              <button
                key={f}
                type="button"
                aria-pressed={selected}
                onClick={() => setChatFilter(f)}
                className={`h-[30px] rounded-full border-[1.5px] px-3 text-caption font-semibold transition-colors duration-fast ease-prive ${
                  selected ? "border-text bg-text text-bg" : "border-border text-text-secondary hover:text-text"
                }`}
              >
                {t(`chatList.filters.${f}`)}
              </button>
            );
          })}
        </div>
      </div>

      {issues.slice(0, maxBanners).map((s) => (
        <StatusBanner key={s.account.id} status={s} />
      ))}
      {issues.length > maxBanners && (
        <p role="status" className="mx-3 mb-2 text-caption text-text-secondary">
          {t("chatList.banner.more", { count: issues.length - maxBanners })}
        </p>
      )}

      <div className="flex flex-1 flex-col items-center justify-center gap-3 p-6 text-center">
        <span className="flex h-14 w-14 items-center justify-center rounded-bubble bg-raised text-text-secondary">
          <ChatIcon size={26} />
        </span>
        <strong className="text-body-strong font-semibold">{t("chatList.empty.title")}</strong>
        {accounts.length === 0 ? (
          <>
            <span className="max-w-[240px] text-text-secondary">{t("chatList.empty.body")}</span>
            <button
              type="button"
              onClick={() => openAddAccount()}
              className="flex h-[38px] items-center gap-2 rounded-control bg-accent px-4 font-bold text-on-accent"
            >
              <PlusIcon size={16} />
              {t("chatList.empty.action")}
            </button>
          </>
        ) : (
          <span className="max-w-[260px] text-text-secondary">{t("chatList.empty.syncSoon")}</span>
        )}
      </div>
    </section>
  );
}

/** Banner for an account that is reconnecting, logged out or failing (docs/UI.md §3). */
function StatusBanner({ status }: { status: AccountStatus }) {
  const { t } = useTranslation();
  const openAddAccount = useUI((s) => s.openAddAccount);
  const account = accountLabel(status);
  const warning = status.state === "reconnecting";

  let title: string;
  let body: string;
  let action: { label: string; run: () => void } | null = null;
  if (status.state === "reconnecting") {
    title = t("chatList.banner.reconnecting.title", { account });
    body = t("chatList.banner.reconnecting.body");
  } else if (status.state === "loggedOut") {
    title = t("chatList.banner.loggedOut.title", { account });
    body = status.code ? translateCode(t, status.code) : t("chatList.banner.loggedOut.body");
    action = { label: t("chatList.banner.loggedOut.action"), run: () => openAddAccount(status.account.id) };
  } else {
    title = t("chatList.banner.error.title", { account });
    body = translateCode(t, status.code ?? "errors.unknown");
    action = { label: t("chatList.banner.error.action"), run: () => void reconnectAccount(status.account.id) };
  }

  return (
    <div
      role="status"
      className={`mx-3 mb-2 flex items-start gap-2.5 rounded-control border px-3 py-2.5 ${
        warning ? "border-status-warning" : "border-status-danger"
      }`}
    >
      <span className={`mt-0.5 shrink-0 ${warning ? "text-status-warning" : "text-status-danger"}`}>
        <RefreshIcon size={18} />
      </span>
      <span className="flex flex-1 flex-col gap-1.5">
        <span className="text-caption font-semibold text-text">{title}</span>
        <span className="text-caption leading-relaxed text-text-secondary">{body}</span>
        {action && (
          <button
            type="button"
            onClick={action.run}
            className="h-[30px] self-start rounded-control bg-accent px-3 text-caption font-bold text-on-accent"
          >
            {action.label}
          </button>
        )}
      </span>
    </div>
  );
}
