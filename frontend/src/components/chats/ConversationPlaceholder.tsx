import { useTranslation } from "react-i18next";
import { useAccounts } from "../../state/accounts";
import { useChats } from "../../state/chats";
import { useUI } from "../../state/ui";
import { accountLabel } from "../accounts/accountLabel";
import { ChatIcon } from "../icons";
import ChatAvatar from "./ChatAvatar";

/** The open chat's header. Its messages arrive in phase 4. */
export default function ConversationPlaceholder() {
  const { t } = useTranslation();
  const selected = useUI((s) => s.selectedChat);
  const chat = useChats((s) => s.chats.find((c) => `${c.accountId}\u0000${c.id}` === selected));
  const account = useAccounts((s) => s.accounts.find((a) => a.account.id === chat?.accountId));
  if (!chat) return null;
  const name = chat.name || t("chatList.unknownChat");

  return (
    <main className="flex min-w-0 flex-1 flex-col bg-bg">
      <header className="flex h-16 shrink-0 items-center gap-3 border-b border-divider px-5">
        <ChatAvatar name={name} size={38} />
        <div className="flex min-w-0 flex-col">
          <h2 className="truncate text-body-strong font-semibold">{name}</h2>
          {account && (
            <span className="truncate text-caption text-text-secondary">
              {accountLabel(account)} · {account.manifest.name} · {t(`account.state.${account.state}`)}
            </span>
          )}
        </div>
      </header>
      <div className="flex flex-1 flex-col items-center justify-center gap-3 p-6 text-center text-text-secondary">
        <span className="flex h-14 w-14 items-center justify-center rounded-bubble bg-raised">
          <ChatIcon size={26} />
        </span>
        <p className="max-w-[340px]">{t("conversation.comingSoon")}</p>
      </div>
    </main>
  );
}
