import { useId, useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { AccountStatus } from "../../lib/accounts";
import { useMessages } from "../../state/messages";
import { accountLabel } from "../accounts/accountLabel";
import { accountBg } from "../accounts/colors";
import { AttachIcon, SendIcon } from "../icons";

/** Unsent text per chat, kept while the app runs so switching chats does not lose it. */
const drafts = new Map<string, string>();

/** Lines the text field grows to before it scrolls. */
const maxLines = 6;

/**
 * Text field and send button (docs/UI.md §2.3). Enter sends, Shift+Enter adds a line.
 * The note above names the account that sends, so nobody replies from the wrong one.
 */
export default function Composer({ account, chatId }: { account: AccountStatus; chatId: string }) {
  const { t } = useTranslation();
  const fieldId = useId();
  const hintId = useId();
  const key = `${account.account.id}\u0000${chatId}`;
  const send = useMessages((s) => s.send);
  const [text, setText] = useState(() => drafts.get(key) ?? "");
  const ref = useRef<HTMLTextAreaElement>(null);
  const caps = account.manifest.capabilities;
  const online = account.state === "connected";
  const canSend = caps.sendText && online && text.trim() !== "";

  // Grow with the text up to maxLines; only then may the field scroll. The height includes
  // the borders (box-sizing: border-box), or a scrollbar would show on an empty field.
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    const style = getComputedStyle(el);
    const borders = parseFloat(style.borderTopWidth) + parseFloat(style.borderBottomWidth);
    const padding = parseFloat(style.paddingTop) + parseFloat(style.paddingBottom);
    const line = parseFloat(style.lineHeight) || 20;
    const max = line * maxLines + padding + borders;
    el.style.height = "auto";
    const needed = el.scrollHeight + borders;
    el.style.height = `${Math.min(needed, max)}px`;
    el.style.overflowY = needed > max ? "auto" : "hidden";
  }, [text]);

  const change = (value: string) => {
    setText(value);
    if (value) drafts.set(key, value);
    else drafts.delete(key);
  };

  const submit = () => {
    if (!canSend) return;
    void send(account.account.id, chatId, text.trim());
    change("");
    ref.current?.focus();
  };

  return (
    <footer className="flex flex-col gap-2 border-t border-divider px-5 pb-4 pt-2.5">
      <span className="flex items-center gap-1.5 text-caption text-text-secondary">
        <span aria-hidden className={`h-2 w-2 rounded-full ${accountBg[account.account.color]}`} />
        {online
          ? t("conversation.sendingAs", { account: accountLabel(account), network: account.manifest.name })
          : t("conversation.offline", { account: accountLabel(account) })}
      </span>
      <div className="flex items-end gap-2">
        <button
          type="button"
          disabled={!caps.media}
          aria-label={t("conversation.attach")}
          title={caps.media ? t("conversation.attach") : t("common.nextVersion")}
          className="flex h-[42px] w-10 shrink-0 items-center justify-center rounded-control text-text-faint disabled:cursor-not-allowed"
        >
          <AttachIcon size={18} />
        </button>
        <label htmlFor={fieldId} className="sr-only">
          {t("conversation.composer")}
        </label>
        <textarea
          id={fieldId}
          ref={ref}
          rows={1}
          value={text}
          onChange={(e) => change(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
              e.preventDefault();
              submit();
            }
          }}
          placeholder={t("conversation.composer")}
          aria-describedby={hintId}
          disabled={!caps.sendText}
          className="min-h-[42px] flex-1 resize-none rounded-control border-[1.5px] border-border bg-raised px-3.5 py-2.5 leading-normal text-text placeholder:text-text-faint"
        />
        <span id={hintId} className="sr-only">
          {t("conversation.sendHint")}
        </span>
        <button
          type="button"
          onClick={submit}
          disabled={!canSend}
          aria-label={t("conversation.send")}
          title={t("conversation.send")}
          className="flex h-[42px] w-[42px] shrink-0 items-center justify-center rounded-control bg-accent text-on-accent disabled:cursor-not-allowed disabled:opacity-50"
        >
          <SendIcon size={18} />
        </button>
      </div>
    </footer>
  );
}
