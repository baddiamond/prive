import { useCallback, useLayoutEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import type { Chat } from "../../lib/chats";
import { translateCode } from "../../lib/errors";
import { buildLog, formatDay } from "../../lib/messages";
import { useMessages, type ChatMessages } from "../../state/messages";
import MessageBubble from "./MessageBubble";

/** Distance from the top, in pixels, at which older messages start loading. */
const loadThreshold = 240;
/** Distance from the bottom within which a new message keeps the view pinned to it. */
const stickThreshold = 120;

interface MessageLogProps {
  chat: Chat;
  chatName: string;
  /** The engine can fetch older history on demand (Capabilities.historyOnDemand). */
  canAsk: boolean;
  /** The engine reports delivery and read status (Capabilities.readReceipts). */
  showStatus: boolean;
}

/**
 * The scrolling history, newest at the bottom. Scrolling near the top loads the previous
 * page from this computer, then asks the network (e.g. the phone) once nothing older is
 * stored and the engine can be asked.
 */
export default function MessageLog({ chat, chatName, canAsk, showStatus }: MessageLogProps) {
  const { t, i18n } = useTranslation();
  const key = `${chat.accountId}\u0000${chat.id}`;
  const state = useMessages((s) => s.chats[key]);
  const loadOlder = useMessages((s) => s.loadOlder);
  const ref = useRef<HTMLDivElement>(null);
  // Scroll geometry before the last render, to keep the view still when older messages
  // are added above and to follow new messages at the bottom.
  const before = useRef<{ height: number; top: number; atBottom: boolean; first?: string; last?: string } | null>(null);

  const messages = state?.messages;
  const items = useMemo(() => buildLog(messages ?? []), [messages]);
  const locale = i18n.resolvedLanguage ?? "en";

  const older = useCallback(() => void loadOlder(chat.accountId, chat.id, canAsk), [loadOlder, chat.accountId, chat.id, canAsk]);

  useLayoutEffect(() => {
    const el = ref.current;
    if (!el || !messages) return;
    const first = messages[0]?.id;
    const last = messages[messages.length - 1]?.id;
    const prev = before.current;
    if (!prev || prev.last === undefined) {
      el.scrollTop = el.scrollHeight; // first render: start at the newest message
    } else if (prev.first !== first && prev.last === last) {
      el.scrollTop = prev.top + (el.scrollHeight - prev.height); // older page added above
    } else if (prev.atBottom || (prev.last !== last && messages[messages.length - 1]?.fromMe)) {
      // Reading the latest messages: follow new ones and status changes (e.g. "Not sent").
      // A message the user just sent is always brought into view.
      el.scrollTop = el.scrollHeight;
    }
    before.current = { height: el.scrollHeight, top: el.scrollTop, atBottom: isAtBottom(el), first, last };
    // Short histories that do not fill the view cannot be scrolled: load more right away.
    if (el.scrollHeight <= el.clientHeight && state?.older === "local") older();
  }, [messages, state?.older, older]);

  const onScroll = () => {
    const el = ref.current;
    if (!el) return;
    if (before.current) {
      before.current = { ...before.current, top: el.scrollTop, height: el.scrollHeight, atBottom: isAtBottom(el) };
    }
    if (el.scrollTop < loadThreshold && wantsMore(state)) older();
  };

  return (
    <div
      ref={ref}
      onScroll={onScroll}
      role="log"
      aria-label={t("conversation.messages")}
      aria-busy={state?.loading || state?.loadingOlder || state?.older === "asking"}
      tabIndex={0}
      className="relative flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto overscroll-contain px-7 py-5 [&>*]:shrink-0"
    >
      <OlderStatus state={state} onLoad={older} />
      {items.map((item) =>
        item.type === "day" ? (
          <div
            key={item.key}
            className="my-2 self-center rounded-control bg-raised px-2.5 py-1 text-micro font-semibold text-text-secondary"
          >
            {formatDay(item.at, locale, t("time.today"), t("time.yesterday"))}
          </div>
        ) : (
          <MessageBubble
            key={item.key}
            message={item.message}
            group={chat.kind === "group"}
            chatName={chatName}
            firstOfRun={item.firstOfRun}
            lastOfRun={item.lastOfRun}
            showStatus={showStatus}
          />
        ),
      )}
    </div>
  );
}

/** The row above the oldest message: loading, asking the network, an error, or the start. */
function OlderStatus({ state, onLoad }: { state: ChatMessages | undefined; onLoad: () => void }) {
  const { t } = useTranslation();
  if (!state || state.loading) {
    return <Note>{t("conversation.loading")}</Note>;
  }
  if (state.errorCode) {
    return (
      <div role="alert" className="mb-2 flex flex-col items-center gap-2 text-center text-caption text-text-secondary">
        <span className="max-w-[420px]">{translateCode(t, state.errorCode)}</span>
        <button
          type="button"
          onClick={onLoad}
          className="h-[30px] rounded-control border-[1.5px] border-border px-3 font-semibold text-text hover:bg-raised"
        >
          {t("conversation.retry")}
        </button>
      </div>
    );
  }
  if (state.loadingOlder) return <Note>{t("conversation.loadingOlder")}</Note>;
  if (state.older === "asking") return <Note>{t("conversation.asking")}</Note>;
  if (state.older === "askable") {
    return (
      <button
        type="button"
        onClick={onLoad}
        className="mb-2 h-[30px] self-center rounded-control border-[1.5px] border-border px-3 text-caption font-semibold text-text-secondary hover:bg-raised hover:text-text"
      >
        {t("conversation.loadOlder")}
      </button>
    );
  }
  if (state.older === "complete") {
    return <Note>{state.messages.length > 0 ? t("conversation.start") : t("conversation.empty")}</Note>;
  }
  return null;
}

function Note({ children }: { children: string }) {
  return (
    <p role="status" className="mb-2 self-center text-caption text-text-faint">
      {children}
    </p>
  );
}

/**
 * Whether reaching the top should load more on its own: stored pages always, the network
 * too unless the last attempt failed (then the user retries with the button).
 */
function wantsMore(state: ChatMessages | undefined): boolean {
  return state?.older === "local" || (state?.older === "askable" && !state.errorCode);
}

function isAtBottom(el: HTMLElement): boolean {
  return el.scrollHeight - el.scrollTop - el.clientHeight < stickThreshold;
}
