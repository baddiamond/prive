import type { ReactNode } from "react";
import { openExternal } from "../../lib/external";

/** A link that opens in the system browser. */
export default function ExternalLink({ href, children }: { href: string; children: ReactNode }) {
  return (
    <a
      href={href}
      onClick={(e) => {
        e.preventDefault();
        openExternal(href);
      }}
      className="font-semibold text-accent underline-offset-2 hover:underline"
    >
      {children}
    </a>
  );
}
