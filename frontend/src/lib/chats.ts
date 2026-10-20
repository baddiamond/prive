// Typed wrappers for the chat list methods bound by Wails (app.go).
import { ListChats } from "../../wailsjs/go/main/App";

/** Kinds of message content; translated with "message.kind.<kind>". */
export type MessageKind =
  | "text"
  | "image"
  | "video"
  | "audio"
  | "voice"
  | "document"
  | "sticker"
  | "location"
  | "contact"
  | "poll"
  | "unsupported";

/** A conversation in the inbox. */
export interface Chat {
  accountId: string;
  id: string;
  name: string;
  kind: "direct" | "group";
  unreadCount: number;
  lastMessagePreview: string;
  /** Null when the chat has no messages. */
  lastMessageAt: Date | null;
  lastMessageKind: MessageKind | "";
  lastMessageFromMe: boolean;
  muted: boolean;
  archived: boolean;
  pinned: boolean;
}

/** Backend payload of a chat (bound result or chat:updated event). */
export interface ChatPayload {
  accountId: string;
  id: string;
  name: string;
  kind: string;
  unreadCount: number;
  lastMessagePreview: string;
  lastMessageAt: string;
  lastMessageKind: string;
  lastMessageFromMe: boolean;
  muted: boolean;
  archived: boolean;
  pinned: boolean;
}

/** Converts a backend chat; Go's zero time becomes null. */
export function chatFromBackend(c: ChatPayload): Chat {
  const at = c.lastMessageAt ? new Date(c.lastMessageAt) : null;
  return {
    accountId: c.accountId,
    id: c.id,
    name: c.name,
    kind: c.kind === "group" ? "group" : "direct",
    unreadCount: c.unreadCount,
    lastMessagePreview: c.lastMessagePreview,
    lastMessageAt: at && at.getUTCFullYear() > 1 ? at : null,
    lastMessageKind: c.lastMessageKind as MessageKind | "",
    lastMessageFromMe: c.lastMessageFromMe,
    muted: c.muted,
    archived: c.archived,
    pinned: c.pinned,
  };
}

/** Loads the inbox of every account (empty accountId) or of one account. */
export async function listChats(accountId = ""): Promise<Chat[]> {
  return (((await ListChats(accountId)) ?? []) as ChatPayload[]).map(chatFromBackend);
}

/** Inbox order: pinned first, then the most recent message. */
export function compareChats(a: Chat, b: Chat): number {
  if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;
  return (b.lastMessageAt?.getTime() ?? 0) - (a.lastMessageAt?.getTime() ?? 0);
}

/** Unread messages that count for badges: muted and archived chats are left out. */
export function badgeCount(chat: Chat): number {
  return chat.muted || chat.archived ? 0 : chat.unreadCount;
}
