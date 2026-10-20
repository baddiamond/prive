import type { AccountColor } from "../../lib/accounts";
import { accountBg } from "./colors";

interface AccountTileProps {
  label: string;
  color: AccountColor;
  /** Edge length in pixels (rail items are 48). */
  size?: number;
}

/** The colored square with the account's initial (docs/UI.md §2.1). Decorative. */
export default function AccountTile({ label, color, size = 48 }: AccountTileProps) {
  const initial = Array.from(label.trim())[0]?.toUpperCase() ?? "";
  return (
    <span
      aria-hidden
      style={{ width: size, height: size }}
      className={`flex shrink-0 items-center justify-center rounded-card font-bold text-on-account-color ${accountBg[color]}`}
    >
      {initial}
    </span>
  );
}
