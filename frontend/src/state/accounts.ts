import { create } from "zustand";
import {
  accountFromBackend,
  listAccounts,
  listEngines,
  type AccountStatus,
  type Manifest,
} from "../lib/accounts";
import { onBackendEvent } from "../lib/events";
import { useUI } from "./ui";

interface AccountsState {
  loaded: boolean;
  /** Every account in rail order. Accounts still being linked (no remote id) are included. */
  accounts: AccountStatus[];
  /** Bundled networks. */
  engines: Manifest[];
  /** Loads accounts and engines and starts listening to account events. Safe to call once. */
  load: () => Promise<void>;
  /** Reloads the account list from the backend. */
  refresh: () => Promise<void>;
}

let listening = false;

/** Global store of accounts and their live state. */
export const useAccounts = create<AccountsState>()((set, get) => ({
  loaded: false,
  accounts: [],
  engines: [],

  load: async () => {
    if (!listening) {
      listening = true;
      subscribe(set, get);
    }
    const [accounts, engines] = await Promise.all([listAccounts(), listEngines()]);
    set({ accounts, engines, loaded: true });
  },

  refresh: async () => {
    set({ accounts: await listAccounts() });
  },
}));

type Set = (partial: Partial<AccountsState>) => void;
type Get = () => AccountsState;

function patch(get: Get, set: Set, id: string, fn: (s: AccountStatus) => AccountStatus) {
  set({ accounts: get().accounts.map((s) => (s.account.id === id ? fn(s) : s)) });
}

function subscribe(set: Set, get: Get) {
  onBackendEvent("account:state", (e) =>
    patch(get, set, e.accountId, (s) => ({ ...s, state: e.state, code: e.code || undefined })),
  );
  onBackendEvent("account:updated", (e) => {
    if (!get().accounts.some((s) => s.account.id === e.accountId)) {
      void get().refresh(); // an account created elsewhere (e.g. during onboarding)
      return;
    }
    patch(get, set, e.accountId, (s) => ({ ...s, account: accountFromBackend(e.account) }));
  });
  onBackendEvent("account:removed", (e) => {
    set({ accounts: get().accounts.filter((s) => s.account.id !== e.accountId) });
    if (useUI.getState().scope === e.accountId) useUI.getState().setScope("all");
  });
}

/** Accounts that finished linking: the ones the rail, inbox and settings show. */
export function linkedAccounts(accounts: AccountStatus[]): AccountStatus[] {
  return accounts.filter((s) => s.account.remoteId !== "");
}
