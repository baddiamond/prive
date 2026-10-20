import type { AccountColor } from "../../lib/accounts";
import { accountBg } from "../accounts/colors";

interface ChatAvatarProps {
  name: string;
  /** Color of the account the chat belongs to, shown as a dot. */
  accountColor?: AccountColor;
  size?: number;
}

/** Initials in a circle with the account's color dot (docs/UI.md §2.2). Decorative. */
export default function ChatAvatar({ name, accountColor, size = 44 }: ChatAvatarProps) {
  return (
    <span
      aria-hidden
      style={{ width: size, height: size }}
      className="relative flex shrink-0 items-center justify-center rounded-full bg-divider text-body-strong font-semibold text-text"
    >
      {initials(name)}
      {accountColor && (
        <span
          className={`absolute -bottom-px -right-px h-3 w-3 rounded-full border-2 border-surface ${accountBg[accountColor]}`}
        />
      )}
    </span>
  );
}

/** Up to two initials from a name; "#" for names without letters (e.g. a phone number). */
function initials(name: string): string {
  const words = name.trim().split(/\s+/).filter((w) => /\p{L}/u.test(w));
  if (words.length === 0) return "#";
  return words
    .slice(0, 2)
    .map((w) => Array.from(w.replace(/[^\p{L}\p{N}]/gu, ""))[0]?.toUpperCase() ?? "")
    .join("");
}
