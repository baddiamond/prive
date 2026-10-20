import { memo } from "react";
import { useTranslation } from "react-i18next";
import type { AccountStatus } from "../../lib/accounts";
import type { Chat } from "../../lib/chats";
import { formatChatTime } from "../../lib/time";
import { accountLabel } from "../accounts/accountLabel";
import ChatAvatar from "./ChatAvatar";

interface ChatRowProps {
  chat: Chat;
  account: AccountStatus | undefined;
  selected: boolean;
  onSelect: (chat: Chat) => void;
}

/** One chat in the list (docs/UI.md §2.2): avatar, name, time, preview, unread count, account. */
function ChatRow({ chat, account, selected, onSelect }: ChatRowProps) {
  const { t, i18n } = useTranslation();
  const unread = chat.unreadCount > 0;
  const name = chat.name || t("chatList.unknownChat");
  const time = chat.lastMessageAt ? formatChatTime(chat.lastMessageAt, i18n.language, t("time.yesterday")) : "";

  return (
    <button
      type="button"
      aria-current={selected ? "true" : undefined}
      onClick={() => onSelect(chat)}
      className={`flex w-full items-center gap-3 rounded-card p-2.5 text-left [contain-intrinsic-size:auto_64px] [content-visibility:auto] ${
        selected ? "bg-raised" : "hover:bg-raised"
      }`}
    >
      <ChatAvatar name={name} accountColor={account?.account.color} />
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="flex items-center gap-2">
          <span className={`min-w-0 flex-1 truncate ${unread ? "font-bold" : "font-medium"}`}>{name}</span>
          {time && (
            <span className={`shrink-0 text-micro ${unread ? "font-bold text-accent" : "font-medium text-text-faint"}`}>{time}</span>
          )}
        </span>
        <span className="flex items-center gap-2">
          <span className={`min-w-0 flex-1 truncate ${unread ? "text-text" : "text-text-secondary"}`}>
            <Preview chat={chat} />
          </span>
          {unread && (
            <span
              aria-label={t("chatList.unreadCount", { count: chat.unreadCount })}
              className={`flex h-5 min-w-5 shrink-0 items-center justify-center rounded-full px-1.5 text-micro font-bold ${
                chat.muted ? "bg-raised text-text-secondary" : "bg-accent text-on-accent"
              }`}
            >
              {chat.unreadCount > 99 ? "99+" : chat.unreadCount}
            </span>
          )}
        </span>
        {account && (
          <span className="truncate text-micro text-text-faint">
            {accountLabel(account)} · {account.manifest.name}
          </span>
        )}
      </span>
    </button>
  );
}

/** Last message preview: the text, or a translated label for media ("Photo"), with "You: ". */
function Preview({ chat }: { chat: Chat }) {
  const { t } = useTranslation();
  if (!chat.lastMessageKind) return null;
  let text = chat.lastMessagePreview;
  if (chat.lastMessageKind !== "text") {
    const label = t(`message.kind.${chat.lastMessageKind}`);
    text = text ? `${label}: ${text}` : label;
  }
  return <>{chat.lastMessageFromMe ? t("chatList.you", { text }) : text}</>;
}

export default memo(ChatRow);
