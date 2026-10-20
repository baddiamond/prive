import type { AccountColor } from "../../lib/accounts";

// Full class names, so Tailwind sees them at build time (dynamic class names are not generated).

/** Background class of each account color token. */
export const accountBg: Record<AccountColor, string> = {
  green: "bg-account-green",
  amber: "bg-account-amber",
  rose: "bg-account-rose",
  sky: "bg-account-sky",
  violet: "bg-account-violet",
  coral: "bg-account-coral",
};
