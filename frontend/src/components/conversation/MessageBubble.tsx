import { useTranslation } from "react-i18next";
import { translateCode } from "../../lib/errors";
import { formatMessageTime, senderColor, type Message } from "../../lib/messages";
import { useMessages } from "../../state/messages";
import { AlertIcon } from "../icons";

interface MessageBubbleProps {
  message: Message;
  /** Group chats show the sender's name on the first bubble of a run. */
  group: boolean;
  /** The chat's name: who wrote incoming messages of a direct chat. */
  chatName: string;
  firstOfRun: boolean;
  lastOfRun: boolean;
  /** The engine reports delivery and read status (Capabilities.readReceipts). */
  showStatus: boolean;
}

/** One message (docs/UI.md §2.3): incoming on `raised`, outgoing on `accentTint`. */
export default function MessageBubble(props: MessageBubbleProps) {
  const { message: m, group, chatName, firstOfRun, lastOfRun, showStatus } = props;
  const { t, i18n } = useTranslation();
  const retry = useMessages((s) => s.retry);
  const failed = m.status === "failed";
  const sender = m.fromMe ? t("conversation.you") : m.senderName || (group ? t("conversation.unknownSender") : chatName);
  const showSender = group && !m.fromMe && firstOfRun;
  // A local message always shows its state; others only where the network reports it.
  const status = m.fromMe && m.status && (showStatus || m.local) ? t(`conversation.status.${m.status}`) : "";
  const meta = [formatMessageTime(m.at, i18n.resolvedLanguage ?? "en"), status].filter(Boolean).join(" · ");
  // The corner next to the author's side is squared on the last bubble of a run.
  const shape = m.fromMe
    ? `bg-accent-tint ${lastOfRun ? "rounded-bubble rounded-br-tag" : "rounded-bubble"}`
    : `bg-raised ${lastOfRun ? "rounded-bubble rounded-bl-tag" : "rounded-bubble"}`;

  return (
    <div className={`flex max-w-[62%] flex-col gap-1.5 ${m.fromMe ? "items-end self-end" : "items-start self-start"} ${firstOfRun ? "mt-2" : ""}`}>
      <div
        className={`flex min-w-0 max-w-full flex-col gap-0.5 border-[1.5px] px-3 pb-1.5 pt-2 ${shape} ${failed ? "border-status-danger" : "border-transparent"}`}
      >
        {showSender ? (
          <span className={`text-caption font-bold ${senderColor(m.senderId || m.senderName)}`}>{sender}</span>
        ) : (
          <span className="sr-only">{sender}</span>
        )}
        {m.kind !== "text" && <span className="italic text-text-secondary">{t(`message.kind.${m.kind}`)}</span>}
        {m.text && <span className="select-text whitespace-pre-wrap leading-relaxed text-text [overflow-wrap:anywhere]">{m.text}</span>}
        <span className={`self-end text-micro font-medium ${failed ? "text-status-danger" : "text-text-secondary"}`}>
          <time dateTime={new Date(m.at).toISOString()}>{meta}</time>
        </span>
      </div>
      {failed && (
        <span role="alert" className="flex flex-wrap items-center gap-2 text-caption text-status-danger">
          <AlertIcon size={14} />
          {m.errorCode && m.errorCode !== "errors.message.sendFailed" && m.errorCode !== "errors.unknown"
            ? translateCode(t, m.errorCode)
            : t("conversation.sendFailed")}
          <button
            type="button"
            onClick={() => void retry(m.accountId, m.chatId, m.id)}
            className="h-[26px] rounded-control border-[1.5px] border-border px-2.5 font-semibold text-text hover:bg-raised"
          >
            {t("conversation.retry")}
          </button>
        </span>
      )}
    </div>
  );
}
