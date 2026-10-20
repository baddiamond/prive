// Recovery phrase API (ADR 0007): status, words, confirmation and unlocking locked data.
import {
  ConfirmRecoveryPhrase,
  GetRecoveryPhrase,
  GetRecoveryStatus,
  UnlockWithPhrase,
} from "../../wailsjs/go/main/App";

/** Number of words in a recovery phrase. */
export const phraseLength = 12;

/** How many words the user types back to confirm the phrase. */
export const confirmCount = 3;

/** Whether the user saved the recovery phrase. */
export interface RecoveryStatus {
  confirmed: boolean;
  /** When they did (RFC 3339), or "". */
  confirmedAt: string;
}

/** One word typed back by the user; position is 1–12. */
export interface RecoveryAnswer {
  position: number;
  word: string;
}

/** Loads whether the recovery phrase was saved. */
export async function getRecoveryStatus(): Promise<RecoveryStatus> {
  const s = await GetRecoveryStatus();
  return { confirmed: s.confirmed, confirmedAt: s.confirmedAt };
}

/** Returns the 12 words in the list for language (English when there is none). */
export async function getRecoveryPhrase(language: string): Promise<string[]> {
  return (await GetRecoveryPhrase(language)) ?? [];
}

/** Checks the typed words; rejects with errors.recovery.wrongWord when one is wrong. */
export function confirmRecoveryPhrase(answers: RecoveryAnswer[]): Promise<void> {
  return ConfirmRecoveryPhrase(answers);
}

/** Opens locked data with the phrase. The page reloads afterwards. */
export function unlockWithPhrase(phrase: string): Promise<void> {
  return UnlockWithPhrase(phrase);
}

/** Picks count distinct positions (1–12) at random, in ascending order. */
export function pickPositions(count = confirmCount): number[] {
  const all = Array.from({ length: phraseLength }, (_, i) => i + 1);
  const random = new Uint32Array(phraseLength);
  crypto.getRandomValues(random);
  for (let i = all.length - 1; i > 0; i--) {
    const j = random[i]! % (i + 1);
    [all[i], all[j]] = [all[j]!, all[i]!];
  }
  return all.slice(0, count).sort((a, b) => a - b);
}
