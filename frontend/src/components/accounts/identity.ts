import { accountColors, updateAccount, type AccountColor } from "../../lib/accounts";
import { useAccounts } from "../../state/accounts";

/** Saves the name and color chosen while linking. An empty name keeps the network's name. */
export async function saveIdentity(accountId: string, name: string, color: AccountColor): Promise<void> {
  const current = useAccounts.getState().accounts.find((s) => s.account.id === accountId)?.account;
  const finalName = name.trim() || current?.displayName || "";
  if (current && current.displayName === finalName && current.color === color) return;
  await updateAccount(accountId, finalName, color);
}

/** The first account color not used yet, cycling when all are taken. */
export function nextColor(used: AccountColor[]): AccountColor {
  return accountColors.find((c) => !used.includes(c)) ?? accountColors[used.length % accountColors.length]!;
}
