import { useTranslation } from "react-i18next";
import { shortcutLabel } from "../lib/platform";
import { useUI, type ChatFilter } from "../state/ui";
import { ChatIcon, PlusIcon, SearchIcon } from "./icons";

const filters: readonly ChatFilter[] = ["all", "unread", "groups"];

/**
 * Middle column (docs/UI.md §2.2). Until accounts can be linked (phase 2) it only shows
 * its header and the empty state.
 */
export default function ChatList() {
  const { t } = useTranslation();
  const chatFilter = useUI((s) => s.chatFilter);
  const setChatFilter = useUI((s) => s.setChatFilter);

  return (
    <section
      aria-label={t("chatList.label")}
      className="flex w-[340px] shrink-0 flex-col border-r border-divider bg-surface"
    >
      <div className="flex flex-col gap-3 px-4 pb-3 pt-[18px]">
        <div className="flex items-baseline justify-between">
          <h1 className="text-heading font-semibold tracking-[-0.01em]">{t("chatList.allInboxes")}</h1>
          <span className="text-caption text-text-secondary">{t("chatList.noAccounts")}</span>
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

      <div className="flex flex-1 flex-col items-center justify-center gap-3 p-6 text-center">
        <span className="flex h-14 w-14 items-center justify-center rounded-bubble bg-raised text-text-secondary">
          <ChatIcon size={26} />
        </span>
        <strong className="text-body-strong font-semibold">{t("chatList.empty.title")}</strong>
        <span className="max-w-[240px] text-text-secondary">{t("chatList.empty.body")}</span>
        <button
          type="button"
          disabled
          title={t("common.nextVersion")}
          className="flex h-[38px] items-center gap-2 rounded-control bg-accent px-4 font-bold text-on-accent disabled:cursor-not-allowed disabled:opacity-60"
        >
          <PlusIcon size={16} />
          {t("chatList.empty.action")}
        </button>
      </div>
    </section>
  );
}
