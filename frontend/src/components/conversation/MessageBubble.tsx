import { useTranslation } from "react-i18next";
import { formatMessageTime, senderColor, type Message } from "../../lib/messages";

interface MessageBubbleProps {
  message: Message;
  /** Group chats show the sender's name on the first bubble of a run. */
  group: boolean;
  /** The chat's name: who wrote incoming messages of a direct chat. */
  chatName: string;
  firstOfRun: boolean;
  lastOfRun: boolean;
}

/** One message (docs/UI.md §2.3): incoming on `raised`, outgoing on `accentTint`. */
export default function MessageBubble({ message: m, group, chatName, firstOfRun, lastOfRun }: MessageBubbleProps) {
  const { t, i18n } = useTranslation();
  const time = formatMessageTime(m.at, i18n.resolvedLanguage ?? "en");
  const sender = m.fromMe ? t("conversation.you") : m.senderName || (group ? t("conversation.unknownSender") : chatName);
  const showSender = group && !m.fromMe && firstOfRun;
  // The corner next to the author's side is squared on the last bubble of a run.
  const shape = m.fromMe
    ? `self-end bg-accent-tint ${lastOfRun ? "rounded-bubble rounded-br-tag" : "rounded-bubble"}`
    : `self-start bg-raised ${lastOfRun ? "rounded-bubble rounded-bl-tag" : "rounded-bubble"}`;

  return (
    <div className={`flex max-w-[62%] flex-col gap-0.5 px-3 pb-1.5 pt-2 ${shape} ${firstOfRun ? "mt-2" : ""}`}>
      {showSender ? (
        <span className={`text-caption font-bold ${senderColor(m.senderId || m.senderName)}`}>{sender}</span>
      ) : (
        <span className="sr-only">{sender}</span>
      )}
      {m.kind !== "text" && (
        <span className="italic text-text-secondary">{t(`message.kind.${m.kind}`)}</span>
      )}
      {m.text && <span className="select-text whitespace-pre-wrap break-words leading-relaxed text-text">{m.text}</span>}
      <time dateTime={new Date(m.at).toISOString()} className="self-end text-micro font-medium text-text-secondary">
        {time}
      </time>
    </div>
  );
}
