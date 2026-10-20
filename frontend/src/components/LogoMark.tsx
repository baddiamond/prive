import type { SVGProps } from "react";

interface LogoMarkProps extends SVGProps<SVGSVGElement> {
  /** Stroke of the "p": the text color by default, a quieter tone for large decorative uses. */
  tone?: "text" | "muted";
}

/**
 * The Privé mark, drawn with theme tokens so one component works in light and dark.
 * Geometry from assets/brand/logo/prive-mark-on-dark.svg; if the mark changes there,
 * update it here. Decorative: label the surrounding element instead.
 */
export default function LogoMark({ tone = "text", ...props }: LogoMarkProps) {
  const stroke = tone === "text" ? "var(--prive-text)" : "var(--prive-divider)";
  return (
    <svg viewBox="0 0 64 64" aria-hidden focusable={false} {...props}>
      <circle cx="34" cy="36" r="14" fill="none" stroke={stroke} strokeWidth="7" />
      <path d="M20 36v22" fill="none" stroke={stroke} strokeWidth="7" strokeLinecap="round" />
      <path d="M39 12 49 5" fill="none" stroke="var(--prive-accent)" strokeWidth="6.5" strokeLinecap="round" />
    </svg>
  );
}
