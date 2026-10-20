import type { SVGProps } from "react";

/**
 * The Privé mark, drawn with theme tokens so one component works in light and dark.
 * Geometry copied from assets/brand/logo/prive-mark-on-dark.svg; if the mark changes there,
 * update it here. Decorative: label the surrounding control instead.
 */
export default function LogoMark(props: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="16.5 1.75 35.75 59.75" aria-hidden focusable={false} {...props}>
      <circle cx="34" cy="36" r="14" fill="none" stroke="var(--prive-text)" strokeWidth="7" />
      <path d="M20 36V58" fill="none" stroke="var(--prive-text)" strokeWidth="7" strokeLinecap="round" />
      <path d="M39 12L49 5" fill="none" stroke="var(--prive-accent)" strokeWidth="6.5" strokeLinecap="round" />
    </svg>
  );
}
