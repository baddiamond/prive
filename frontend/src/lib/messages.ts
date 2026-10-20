// Messages of one chat: backend wrappers and the grouping the conversation view needs.
import { GetMessages, MarkChatRead, RequestOlderHistory, SendMessage } from "../../wailsjs/go/main/App";
import type { MessageKind } from "./chats";

/** Page size when opening a chat and when scrolling back. */
export const messagePageSize = 50;

/** Delivery status of a message the user sent; "" when unknown. */
export type MessageStatus = "" | "pending" | "sent" | "delivered" | "read" | "failed";

/** One message, as the UI uses it. */
export interface Message {
  accountId: string;
  chatId: string;
  id: string;
  senderId: string;
  senderName: string;
  fromMe: boolean;
  /** Unix time in milliseconds. */
  at: number;
  kind: MessageKind;
  text: string;
  status: MessageStatus;
  /** Set on a message typed here that the network has not accepted yet ("local:<n>" id). */
  local?: boolean;
  /** Translation key of why a local message was not sent. */
  errorCode?: string;
}

/** A message as the backend sends it (bindings and events). */
export interface MessagePayload {
  accountId: string;
  chatId: string;
  id: string;
  senderId: string;
  senderName: string;
  fromMe: boolean;
  timestamp: string;
  kind: string;
  text: string;
  status: string;
}

const kinds = new Set(["text", "image", "video", "audio", "voice", "document", "sticker", "location", "contact", "poll"]);
const statuses = new Set(["pending", "sent", "delivered", "read", "failed"]);

/** Converts a backend message. */
export function messageFromBackend(m: MessagePayload): Message {
  return {
    accountId: m.accountId,
    chatId: m.chatId,
    id: m.id,
    senderId: m.senderId,
    senderName: m.senderName,
    fromMe: m.fromMe,
    at: Date.parse(m.timestamp) || 0,
    kind: (kinds.has(m.kind) ? m.kind : "unsupported") as MessageKind,
    text: m.text,
    status: (statuses.has(m.status) ? m.status : "") as MessageStatus,
  };
}

/**
 * Loads a page of stored messages, oldest first: the newest ones without a cursor, or the
 * ones just before `before` (the oldest message already shown).
 */
export async function getMessages(accountId: string, chatId: string, before?: Message): Promise<Message[]> {
  const page = await GetMessages(accountId, chatId, before?.at ?? 0, before?.id ?? "", messagePageSize);
  return (page ?? []).map((m) => messageFromBackend(m as unknown as MessagePayload));
}

/** Asks the network (e.g. the phone) for older messages; the answer is a history:result event. */
export function requestOlderHistory(accountId: string, chatId: string): Promise<void> {
  return RequestOlderHistory(accountId, chatId);
}

/** Sends a text message; resolves with the message as the network accepted it. */
export async function sendMessage(accountId: string, chatId: string, text: string): Promise<Message> {
  const m = await SendMessage(accountId, chatId, text);
  return messageFromBackend(m as unknown as MessagePayload);
}

/** Clears a chat's unread count and sends read receipts where the network supports them. */
export function markChatRead(accountId: string, chatId: string): Promise<void> {
  return MarkChatRead(accountId, chatId);
}

const statusOrder: Record<MessageStatus, number> = { "": 0, failed: 0, pending: 1, sent: 2, delivered: 3, read: 4 };

/**
 * The status to keep when a message in state current receives next: delivery only moves
 * forward, and "failed" only replaces a message that never left (like core.MergeStatus).
 */
export function mergeStatus(current: MessageStatus, next: MessageStatus): MessageStatus {
  if (next === "failed") return current === "" || current === "pending" ? "failed" : current;
  return statusOrder[next] > statusOrder[current] ? next : current;
}

/** Orders messages like the store does: by time, then id. */
export function compareMessages(a: Message, b: Message): number {
  return a.at - b.at || (a.id < b.id ? -1 : a.id > b.id ? 1 : 0);
}

/** Merges two sorted lists without duplicates (a message seen twice keeps its newest copy). */
export function mergeMessages(current: Message[], incoming: Message[]): Message[] {
  const byId = new Map(current.map((m) => [m.id, m]));
  for (const m of incoming) byId.set(m.id, m);
  return [...byId.values()].sort(compareMessages);
}

/** A row of the message log: a day separator or a message. */
export type LogItem =
  | { type: "day"; key: string; at: number }
  | { type: "message"; key: string; message: Message; firstOfRun: boolean; lastOfRun: boolean };

/** Messages closer than this, from the same author, form one run (one sender name shown). */
const runGapMs = 5 * 60_000;

/** Splits messages into days and runs of consecutive messages from the same author. */
export function buildLog(messages: Message[]): LogItem[] {
  const items: LogItem[] = [];
  let previous: Message | undefined;
  messages.forEach((m, i) => {
    const next = messages[i + 1];
    if (!previous || !sameDay(previous.at, m.at)) {
      items.push({ type: "day", key: `day-${dayKey(m.at)}`, at: m.at });
      previous = undefined;
    }
    items.push({
      type: "message",
      key: m.id,
      message: m,
      firstOfRun: !previous || !sameRun(previous, m),
      lastOfRun: !next || !sameDay(m.at, next.at) || !sameRun(m, next),
    });
    previous = m;
  });
  return items;
}

function sameRun(a: Message, b: Message): boolean {
  return a.fromMe === b.fromMe && a.senderId === b.senderId && Math.abs(b.at - a.at) < runGapMs;
}

function dayKey(at: number): string {
  const d = new Date(at);
  return `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`;
}

function sameDay(a: number, b: number): boolean {
  return dayKey(a) === dayKey(b);
}

/** Label of a day separator: "Today", "Yesterday", the weekday within a week, then the date. */
export function formatDay(at: number, locale: string, today: string, yesterday: string, now = new Date()): string {
  const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  const date = new Date(at);
  const days = Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000);
  if (days <= 0) return today;
  if (days === 1) return yesterday;
  if (days < 7) return new Intl.DateTimeFormat(locale, { weekday: "long" }).format(date);
  return new Intl.DateTimeFormat(locale, {
    day: "numeric",
    month: "long",
    year: date.getFullYear() === now.getFullYear() ? undefined : "numeric",
  }).format(date);
}

/** Time of a message, e.g. "09:30" or "9:30 AM" depending on the language. */
export function formatMessageTime(at: number, locale: string): string {
  return new Intl.DateTimeFormat(locale, { hour: "2-digit", minute: "2-digit" }).format(new Date(at));
}

/** Sender name colors (brand tokens color.sender.*), as static class names for Tailwind. */
const senderColors = [
  "text-sender-green",
  "text-sender-amber",
  "text-sender-rose",
  "text-sender-sky",
  "text-sender-violet",
  "text-sender-coral",
] as const;

/** A stable color class for a sender, so each person keeps their color in a group. */
export function senderColor(senderId: string): string {
  let h = 0;
  for (let i = 0; i < senderId.length; i++) h = (h * 31 + senderId.charCodeAt(i)) >>> 0;
  return senderColors[h % senderColors.length]!;
}
