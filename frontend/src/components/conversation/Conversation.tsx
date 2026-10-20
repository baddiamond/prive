import { useEffect } from "react";
import { markChatRead } from "../../lib/messages";
import { useTranslation } from "react-i18next";
import { useAccounts } from "../../state/accounts";
import { useChats } from "../../state/chats";
import { useMessages } from "../../state/messages";
import { useUI } from "../../state/ui";
import { accountLabel } from "../accounts/accountLabel";
import { accountBg } from "../accounts/colors";
import ChatAvatar from "../chats/ChatAvatar";
import Composer from "./Composer";
import MessageLog from "./MessageLog";

/** The open chat (docs/UI.md §2.3): header, message history and the composer area. */
export default function Conversation() {
  const { t } = useTranslation();
  const selected = useUI((s) => s.selectedChat);
  const chat = useChats((s) => s.chats.find((c) => `${c.accountId}\u0000${c.id}` === selected));
  const account = useAccounts((s) => s.accounts.find((a) => a.account.id === chat?.accountId));
  const open = useMessages((s) => s.open);
  const canAsk = account?.manifest.capabilities.historyOnDemand ?? false;

  useEffect(() => {
    if (chat) void open(chat.accountId, chat.id, canAsk);
  }, [chat?.accountId, chat?.id, canAsk, open]); // eslint-disable-line react-hooks/exhaustive-deps

  useMarkRead(chat?.accountId, chat?.id, chat?.unreadCount ?? 0);

  if (!chat) return null;
  const name = chat.name || t("chatList.unknownChat");
  const dot = account ? accountBg[account.account.color] : "bg-text-faint";

  return (
    <main className="flex min-w-0 flex-1 flex-col bg-bg">
      <header className="flex h-16 shrink-0 items-center gap-3 border-b border-divider px-5">
        <ChatAvatar name={name} size={38} />
        <div className="flex min-w-0 flex-col">
          <h2 className="truncate text-body-strong font-semibold">{name}</h2>
          {account && (
            <span className="flex items-center gap-1.5 truncate text-caption text-text-secondary">
              <span aria-hidden className={`h-2 w-2 shrink-0 rounded-full ${dot}`} />
              {accountLabel(account)} · {account.manifest.name} · {t(`account.state.${account.state}`)}
            </span>
          )}
        </div>
      </header>

      <MessageLog
        key={selected}
        chat={chat}
        chatName={name}
        canAsk={canAsk}
        showStatus={account?.manifest.capabilities.readReceipts ?? false}
      />

      {account && <Composer key={selected} account={account} chatId={chat.id} />}
    </main>
  );
}

/**
 * Marks the open chat as read while the window has focus: when it opens, when messages
 * arrive, and when the user comes back to the window.
 */
function useMarkRead(accountId: string | undefined, chatId: string | undefined, unread: number) {
  useEffect(() => {
    if (!accountId || !chatId || unread === 0) return;
    const mark = () => {
      if (document.hasFocus()) void markChatRead(accountId, chatId).catch(() => undefined);
    };
    mark();
    window.addEventListener("focus", mark);
    return () => window.removeEventListener("focus", mark);
  }, [accountId, chatId, unread]);
}
