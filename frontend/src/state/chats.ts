import { create } from "zustand";
import { badgeCount, chatFromBackend, compareChats, listChats, type Chat } from "../lib/chats";
import { onBackendEvent } from "../lib/events";

interface ChatsState {
  loaded: boolean;
  /** Every non-archived chat of every account, in inbox order. */
  chats: Chat[];
  /** Initial sync progress per account (0–100); absent when not syncing. */
  syncProgress: Record<string, number>;
  /** Loads the chat list and starts listening to chat events. Safe to call more than once. */
  load: () => Promise<void>;
}

let listening = false;
let reloadTimer: ReturnType<typeof setTimeout> | undefined;

/** Global store of the inbox. */
export const useChats = create<ChatsState>()((set, get) => ({
  loaded: false,
  chats: [],
  syncProgress: {},

  load: async () => {
    if (!listening) {
      listening = true;
      subscribe(set, get);
    }
    set({ chats: await listChats(), loaded: true });
  },
}));

type Set = (partial: Partial<ChatsState>) => void;
type Get = () => ChatsState;

const key = (c: { accountId: string; id: string }) => `${c.accountId}\u0000${c.id}`;

function subscribe(set: Set, get: Get) {
  onBackendEvent("chat:updated", (e) => {
    const chat = chatFromBackend(e.chat);
    const others = get().chats.filter((c) => key(c) !== key(chat));
    const next = chat.archived ? others : [...others, chat];
    set({ chats: next.sort(compareChats) });
  });
  // A history batch can touch hundreds of chats: reload once, shortly after the last one.
  onBackendEvent("chats:synced", () => {
    clearTimeout(reloadTimer);
    reloadTimer = setTimeout(() => void get().load(), 400);
  });
  onBackendEvent("sync:progress", (e) => {
    const progress = { ...get().syncProgress };
    if (e.total > 0 && e.done < e.total) progress[e.accountId] = Math.round((e.done / e.total) * 100);
    else delete progress[e.accountId];
    set({ syncProgress: progress });
  });
  onBackendEvent("account:removed", (e) =>
    set({ chats: get().chats.filter((c) => c.accountId !== e.accountId) }),
  );
}

/** Unread totals for the rail: per account and overall (muted and archived chats excluded). */
export function unreadCounts(chats: Chat[]): { total: number; perAccount: Record<string, number> } {
  const perAccount: Record<string, number> = {};
  let total = 0;
  for (const c of chats) {
    const n = badgeCount(c);
    if (n > 0) {
      perAccount[c.accountId] = (perAccount[c.accountId] ?? 0) + n;
      total += n;
    }
  }
  return { total, perAccount };
}
