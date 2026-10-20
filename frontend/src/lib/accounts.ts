// Typed wrappers for the account and onboarding methods bound by Wails (app.go, onboarding.go).
import {
  AcceptLegal,
  AddAccount,
  CancelLogin,
  CompleteOnboarding,
  GetOnboarding,
  ListAccounts,
  ListEngines,
  Login,
  ReconnectAccount,
  RemoveAccount,
  RenderQR,
  SetOnboardingStep,
  UpdateAccount,
} from "../../wailsjs/go/main/App";

/** Account accent colors, the "account.*" brand tokens. */
export const accountColors = ["green", "amber", "rose", "sky", "violet", "coral"] as const;
export type AccountColor = (typeof accountColors)[number];

/** Connection states sent in account:state. Translated with "account.state.<state>". */
export type AccountState = "disconnected" | "connecting" | "connected" | "reconnecting" | "loggedOut" | "error";

/** Feature flags of an engine; the UI never checks the engine id. */
export interface Capabilities {
  sendText: boolean;
  readReceipts: boolean;
  typing: boolean;
  media: boolean;
  reactions: boolean;
  edit: boolean;
  delete: boolean;
  replies: boolean;
}

/** Describes one network. */
export interface Manifest {
  id: string;
  /** Brand name, not translated. */
  name: string;
  /** 2–3 letter tag for the rail. */
  shortName: string;
  loginMethods: string[];
  capabilities: Capabilities;
}

/** One linked account. */
export interface Account {
  id: string;
  engineId: string;
  /** Network identity (e.g. a WhatsApp JID); empty until linked. Never display it unmasked. */
  remoteId: string;
  displayName: string;
  color: AccountColor;
  position: number;
}

/** An account with its live connection state. */
export interface AccountStatus {
  account: Account;
  state: AccountState;
  /** Translation key explaining the state, if any. */
  code?: string;
  manifest: Manifest;
}

/** First-run state (docs/ONBOARDING.md). */
export interface Onboarding {
  completed: boolean;
  step: number;
  legalAccepted: boolean;
  legalVersion: number;
}

/** A QR code as one SVG path over a size×size grid. */
export interface QRCode {
  size: number;
  path: string;
}

export function isAccountColor(value: string): value is AccountColor {
  return (accountColors as readonly string[]).includes(value);
}

function toAccount(a: { id: string; engineId: string; remoteId: string; displayName: string; color: string; position: number }): Account {
  return {
    id: a.id,
    engineId: a.engineId,
    remoteId: a.remoteId,
    displayName: a.displayName,
    color: isAccountColor(a.color) ? a.color : "green",
    position: a.position,
  };
}

/** Normalizes an account payload from the backend (bound method result or event). */
export function accountFromBackend(a: Parameters<typeof toAccount>[0]): Account {
  return toAccount(a);
}

export async function listAccounts(): Promise<AccountStatus[]> {
  const list = await ListAccounts();
  return list.map((s) => ({
    account: toAccount(s.account),
    state: s.state as AccountState,
    code: s.code || undefined,
    manifest: s.manifest as Manifest,
  }));
}

export async function listEngines(): Promise<Manifest[]> {
  return (await ListEngines()) as Manifest[];
}

export async function addAccount(engineId: string, displayName: string, color: AccountColor): Promise<Account> {
  return toAccount(await AddAccount(engineId, displayName, color));
}

export async function updateAccount(id: string, displayName: string, color: AccountColor): Promise<Account> {
  return toAccount(await UpdateAccount(id, displayName, color));
}

export const login = (id: string, method: string): Promise<void> => Login(id, method);
export const cancelLogin = (id: string): Promise<void> => CancelLogin(id);
export const removeAccount = (id: string): Promise<void> => RemoveAccount(id);
export const reconnectAccount = (id: string): Promise<void> => ReconnectAccount(id);
export const renderQR = (text: string): Promise<QRCode> => RenderQR(text);

export const getOnboarding = (): Promise<Onboarding> => GetOnboarding();
export const setOnboardingStep = (step: number): Promise<void> => SetOnboardingStep(step);
export const acceptLegal = (): Promise<void> => AcceptLegal();
export const completeOnboarding = (): Promise<void> => CompleteOnboarding();

/** Shows only the last two digits of a phone-based identity: "•••• ••47". */
export function maskRemoteId(remoteId: string): string {
  const user = remoteId.split("@")[0] ?? "";
  return user.length >= 2 ? `•••• ••${user.slice(-2)}` : "";
}
