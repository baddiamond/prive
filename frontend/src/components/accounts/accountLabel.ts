import type { AccountStatus } from "../../lib/accounts";

/** The name shown for an account: its label, or the network's name when it has none. */
export function accountLabel(s: AccountStatus): string {
  return s.account.displayName.trim() || s.manifest.name;
}

/** States that need the user's attention (docs/UI.md §3). */
export function hasIssue(s: AccountStatus): boolean {
  return s.state === "reconnecting" || s.state === "loggedOut" || s.state === "error";
}
