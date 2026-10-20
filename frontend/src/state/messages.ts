import { create } from "zustand";
import { errorCode } from "../lib/errors";
import { onBackendEvent } from "../lib/events";
import {
  getMessages,
  mergeMessages,
  messageFromBackend,
  messagePageSize,
  requestOlderHistory,
  type Message,
} from "../lib/messages";

/** Where older messages of a chat stand once the stored ones are all shown. */
export type OlderState =
  /** More stored messages may exist on this computer. */
  | "local"
  /** Every stored message is shown; the network can be asked for more. */
  | "askable"
  /** Waiting for the network (e.g. the phone) to answer. */
  | "asking"
  /** The network has nothing older, or cannot be asked. */
  | "complete";

/** One chat's loaded messages. */
export interface ChatMessages {
  messages: Message[];
  /** True while the first page loads. */
  loading: boolean;
  /** True while an older page loads from this computer. */
  loadingOlder: boolean;
  older: OlderState;
  /** The network said it has nothing older: once the stored pages end, so does the chat. */
  remoteComplete: boolean;
  /** Translation key of the last failure (loading, or asking the network). */
  errorCode: string | null;
}

interface MessagesState {
  /** Loaded chats by "accountId\u0000chatId". */
  chats: Record<string, ChatMessages>;
  /**
   * Shows a chat: loads its newest page unless it is already loaded (reopening is
   * instant). canAsk says whether the engine can fetch older history on demand.
   */
  open: (accountId: string, chatId: string, canAsk: boolean) => Promise<void>;
  /** Loads the page before the oldest shown message, or asks the network when none is left. */
  loadOlder: (accountId: string, chatId: string, canAsk: boolean) => Promise<void>;
}

/** Chats kept in memory; the least recently opened are dropped beyond this. */
const maxCachedChats = 20;

let listening = false;
const recent: string[] = [];

const chatKey = (accountId: string, chatId: string) => `${accountId}\u0000${chatId}`;

const empty: ChatMessages = {
  messages: [],
  loading: true,
  loadingOlder: false,
  older: "local",
  remoteComplete: false,
  errorCode: null,
};

/** Messages of the chats the user opened, cached in memory on top of the local database. */
export const useMessages = create<MessagesState>()((set, get) => {
  const update = (key: string, patch: Partial<ChatMessages>) =>
    set((s) => ({ chats: { ...s.chats, [key]: { ...(s.chats[key] ?? empty), ...patch } } }));

  /** What is left once the stored messages before the oldest one are all shown. */
  const exhausted = (key: string, canAsk: boolean): OlderState =>
    canAsk && !get().chats[key]?.remoteComplete ? "askable" : "complete";

  return {
    chats: {},

    open: async (accountId, chatId, canAsk) => {
      if (!listening) {
        listening = true;
        subscribe();
      }
      const key = chatKey(accountId, chatId);
      touch(key);
      if (get().chats[key]) return;
      update(key, { ...empty });
      try {
        const page = await getMessages(accountId, chatId);
        update(key, {
          messages: mergeMessages(get().chats[key]?.messages ?? [], page),
          loading: false,
          older: page.length < messagePageSize ? exhausted(key, canAsk) : "local",
        });
      } catch (err) {
        update(key, { loading: false, errorCode: errorCode(err) });
      }
    },

    loadOlder: async (accountId, chatId, canAsk) => {
      const key = chatKey(accountId, chatId);
      const chat = get().chats[key];
      if (!chat || chat.loading || chat.loadingOlder || chat.older === "asking" || chat.older === "complete") return;

      if (chat.older === "askable") {
        update(key, { older: "asking", errorCode: null });
        try {
          await requestOlderHistory(accountId, chatId);
        } catch (err) {
          update(key, { older: "askable", errorCode: errorCode(err) });
        }
        return;
      }

      update(key, { loadingOlder: true, errorCode: null });
      try {
        const page = await getMessages(accountId, chatId, chat.messages[0]);
        const current = get().chats[key]?.messages ?? [];
        update(key, {
          messages: mergeMessages(current, page),
          loadingOlder: false,
          older: page.length < messagePageSize ? exhausted(key, canAsk) : "local",
        });
      } catch (err) {
        update(key, { loadingOlder: false, errorCode: errorCode(err) });
      }
    },
  };

  function subscribe() {
    // A live message lands in its chat if that chat is loaded.
    onBackendEvent("message:new", (e) => {
      const m = messageFromBackend(e.message);
      const key = chatKey(e.accountId, m.chatId);
      const chat = get().chats[key];
      if (chat && !chat.loading) update(key, { messages: mergeMessages(chat.messages, [m]) });
    });

    // The phone answered: show what it sent (now stored) and whether more remain.
    onBackendEvent("history:result", (e) => {
      const key = chatKey(e.accountId, e.chatId);
      const chat = get().chats[key];
      if (!chat) return;
      if (e.code) {
        update(key, { older: "askable", errorCode: e.code });
        return;
      }
      void getMessages(e.accountId, e.chatId, chat.messages[0])
        .then((page) => {
          const current = get().chats[key]?.messages ?? [];
          const remoteComplete = e.complete || e.messages === 0;
          update(key, {
            messages: mergeMessages(current, page),
            remoteComplete,
            older: page.length >= messagePageSize ? "local" : remoteComplete ? "complete" : "askable",
            errorCode: null,
          });
        })
        .catch((err: unknown) => update(key, { older: "askable", errorCode: errorCode(err) }));
    });

    // A history batch may hold newer messages of an open chat (e.g. right after linking).
    onBackendEvent("chats:synced", (e) => {
      for (const [key, chat] of Object.entries(get().chats)) {
        if (!key.startsWith(`${e.accountId}\u0000`) || chat.loading) continue;
        const chatId = key.slice(e.accountId.length + 1);
        void getMessages(e.accountId, chatId)
          .then((page) => update(key, { messages: mergeMessages(get().chats[key]?.messages ?? [], page) }))
          .catch(() => undefined);
      }
    });
  }

  /** Marks a chat as recently opened and forgets the oldest ones beyond maxCachedChats. */
  function touch(key: string) {
    const i = recent.indexOf(key);
    if (i >= 0) recent.splice(i, 1);
    recent.push(key);
    while (recent.length > maxCachedChats) {
      const dropped = recent.shift()!;
      set((s) => {
        const chats = { ...s.chats };
        delete chats[dropped];
        return { chats };
      });
    }
  }
});
