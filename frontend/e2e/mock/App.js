// In-memory stand-in for the Go bindings (wailsjs/go/main/App) used by the UI regression
// harness. The scenario comes from the page URL: ?scenario=inbox&lang=en&theme=dark.
import { engines, makeAccounts, makeChats, makeHistory } from "./data.js";
import { emit } from "./runtime.js";

const params = new URLSearchParams(window.location.search);
const scenario = params.get("scenario") ?? "inbox";
const lang = params.get("lang") ?? "en";
const theme = params.get("theme") ?? "dark";

const setups = {
  inbox: { accounts: 2, chats: 30 },
  "many-accounts": { accounts: 16, chats: 2 },
  issues: { accounts: 4, chats: 5, states: ["reconnecting", "loggedOut", "error", "connected"] },
  empty: { accounts: 0, chats: 0 },
  onboarding: { accounts: 0, chats: 0, onboarding: false },
  locked: { accounts: 0, chats: 0, locked: true },
};
const setup = setups[scenario] ?? setups.inbox;

let accounts = makeAccounts(setup.accounts, setup.states);
const chats = makeChats(accounts, setup.chats);
const histories = new Map();
let confirmed = params.get("recovery") === "saved";
let sent = 0;

/** Rejects like a bound Go method: an Error whose message is the JSON of {code, params}. */
const fail = (code, params) => Promise.reject(new Error(JSON.stringify({ code, params })));
const ok = (value) => Promise.resolve(value);
const guard = () => (setup.locked ? fail("errors.encryption.locked") : null);

function history(accountId, chatId) {
  const key = `${accountId}/${chatId}`;
  if (!histories.has(key)) {
    const chat = chats.find((c) => c.accountId === accountId && c.id === chatId);
    histories.set(key, chat ? makeHistory(chat, chat.id.endsWith("-chat-1") || chat.id.endsWith("-chat-2") ? 400 : 25) : []);
  }
  return histories.get(key);
}

export const GetAppInfo = () => ok({ version: "0.0.0-e2e", dataDir: "/Users/someone/Library/Application Support/Prive" });
export const GetSystemLocale = () => ok(lang);
export const GetSettings = () => guard() ?? ok({ language: lang, systemLocale: lang, theme });
export const SetLanguage = () => ok();
export const SetTheme = () => ok();
export const GetOnboarding = () =>
  guard() ?? ok({ completed: setup.onboarding !== false, step: 1, legalAccepted: setup.onboarding !== false, legalVersion: 1 });
export const SetOnboardingStep = () => ok();
export const AcceptLegal = () => ok();
export const CompleteOnboarding = () => ok();
export const ListEngines = () => ok(engines);
export const ListAccounts = () => guard() ?? ok(accounts);
export const AddAccount = (engineId, displayName, color) => {
  const acc = makeAccounts(accounts.length + 1)[accounts.length];
  acc.account.displayName = displayName;
  acc.account.color = color;
  acc.account.remoteId = "";
  accounts = [...accounts, acc];
  return ok(acc.account);
};
export const UpdateAccount = (id, displayName, color) => {
  const s = accounts.find((a) => a.account.id === id);
  if (s) Object.assign(s.account, { displayName, color });
  return ok(s?.account);
};
export const Login = () => ok();
export const CancelLogin = () => ok();
export const RemoveAccount = (id) => {
  accounts = accounts.filter((a) => a.account.id !== id);
  return ok();
};
export const ReconnectAccount = () => ok();
export const RenderQR = () => ok({ size: 25, modules: "10".repeat(312) + "1" });
export const ListChats = (accountId) => guard() ?? ok(chats.filter((c) => !accountId || c.accountId === accountId));
export const UnreadSummary = () => ok({ total: 0, perAccount: {} });
export const GetMessages = (accountId, chatId, beforeMs, beforeId, limit) => {
  const all = history(accountId, chatId);
  const end = beforeMs
    ? all.findIndex((m) => Date.parse(m.timestamp) > beforeMs || (Date.parse(m.timestamp) === beforeMs && m.id >= beforeId))
    : all.length;
  const stop = end < 0 ? all.length : end;
  return ok(all.slice(Math.max(0, stop - limit), stop));
};
export const RequestOlderHistory = (accountId, chatId) => {
  setTimeout(() => emit("history:result", { accountId, chatId, messages: 0, complete: true }), 300);
  return ok();
};
export const SendMessage = (accountId, chatId, text) => {
  const m = {
    accountId,
    chatId,
    id: `sent-${++sent}`,
    senderId: "",
    senderName: "",
    fromMe: true,
    timestamp: new Date().toISOString(),
    kind: "text",
    text,
    status: "sent",
  };
  history(accountId, chatId).push(m);
  return ok(m);
};
export const MarkChatRead = (accountId, chatId) => {
  const c = chats.find((x) => x.accountId === accountId && x.id === chatId);
  if (c && c.unreadCount) {
    c.unreadCount = 0;
    setTimeout(() => emit("chat:updated", { accountId, chat: c }), 0);
  }
  return ok();
};
export const GetRecoveryStatus = () => ok({ confirmed, confirmedAt: confirmed ? new Date().toISOString() : "" });
export const GetRecoveryPhrase = () =>
  ok(["ocean", "candle", "maple", "river", "pilot", "garden", "violet", "harbor", "tiger", "lemon", "canyon", "orbit"]);
export const ConfirmRecoveryPhrase = () => {
  confirmed = true;
  return ok();
};
export const UnlockWithPhrase = () => fail("errors.recovery.noMatch");
