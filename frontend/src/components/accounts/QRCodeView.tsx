import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { renderQR, type QRCode } from "../../lib/accounts";

interface QRCodeViewProps {
  /** QR payload from the engine, or null while waiting for the first one. */
  value: string | null;
  /** Shown over the code (e.g. a check mark once linked). */
  overlay?: React.ReactNode;
}

/**
 * Draws a login QR code in Privé's style: round dots, rounded finder patterns with an
 * indigo center, and the mark in the middle (the code uses error correction Q, so the
 * covered modules are recovered). It uses the qr* tokens, which are the same in both
 * themes: dark on white, because phone cameras read inverted codes unreliably.
 */
export default function QRCodeView({ value, overlay }: QRCodeViewProps) {
  const { t } = useTranslation();
  const [code, setCode] = useState<QRCode | null>(null);

  useEffect(() => {
    let current = true;
    if (value) {
      void renderQR(value).then((c) => current && setCode(c));
    }
    return () => {
      current = false;
    };
  }, [value]);

  return (
    <div
      role="img"
      aria-label={t("link.qrAlt")}
      className="relative flex aspect-square w-full max-w-[300px] items-center justify-center rounded-card bg-qr-paper p-3"
    >
      {value && code ? (
        <StyledQR code={code} />
      ) : (
        <span className="h-8 w-8 animate-spin rounded-full border-2 border-divider border-t-qr-ink motion-reduce:animate-none" />
      )}
      {overlay && (
        <div className="absolute inset-0 flex items-center justify-center rounded-card">
          {/* Tailwind can't add opacity to a CSS-variable color, so the veil is its own layer. */}
          <span aria-hidden className="absolute inset-0 rounded-card bg-qr-paper opacity-90" />
          <span className="relative">{overlay}</span>
        </div>
      )}
    </div>
  );
}

const quiet = 2; // modules of white margin around the code
const finder = 7; // finder patterns are 7×7 modules

function StyledQR({ code }: { code: QRCode }) {
  const { size, modules } = code;
  const layout = useMemo(() => qrLayout(size, modules), [size, modules]);

  return (
    <svg
      viewBox={`${-quiet} ${-quiet} ${size + quiet * 2} ${size + quiet * 2}`}
      className="h-full w-full"
      aria-hidden
    >
      <g fill="var(--prive-qr-ink)">
        {layout.dots.map(([x, y]) => (
          <circle key={`${x}-${y}`} cx={x + 0.5} cy={y + 0.5} r={0.45} />
        ))}
      </g>
      {layout.finders.map(([x, y]) => (
        <g key={`f-${x}-${y}`}>
          <rect
            x={x + 0.5}
            y={y + 0.5}
            width={finder - 1}
            height={finder - 1}
            rx={1.8}
            fill="none"
            stroke="var(--prive-qr-ink)"
            strokeWidth={1}
          />
          <rect x={x + 2} y={y + 2} width={3} height={3} rx={0.9} fill="var(--prive-qr-accent)" />
        </g>
      ))}
      <g transform={`translate(${layout.logo.x} ${layout.logo.y})`}>
        <rect width={layout.logo.size} height={layout.logo.size} rx={layout.logo.size * 0.22} fill="var(--prive-qr-paper)" />
        <svg x={layout.logo.size * 0.15} y={layout.logo.size * 0.15} width={layout.logo.size * 0.7} height={layout.logo.size * 0.7} viewBox="0 0 64 64">
          <circle cx="34" cy="36" r="14" fill="none" stroke="var(--prive-qr-ink)" strokeWidth="7" />
          <path d="M20 36v22" fill="none" stroke="var(--prive-qr-ink)" strokeWidth="7" strokeLinecap="round" />
          <path d="M39 12 49 5" fill="none" stroke="var(--prive-qr-accent)" strokeWidth="6.5" strokeLinecap="round" />
        </svg>
      </g>
    </svg>
  );
}

/** Splits the grid into data dots, the three finder patterns and the logo area. */
function qrLayout(size: number, modules: string) {
  const finders: [number, number][] = [
    [0, 0],
    [size - finder, 0],
    [0, size - finder],
  ];
  const inFinder = (x: number, y: number) =>
    finders.some(([fx, fy]) => x >= fx && x < fx + finder && y >= fy && y < fy + finder);

  // The logo covers about a sixth of the width (≈3% of the modules), centered on whole modules.
  let logoSize = Math.round(size * 0.16);
  if ((size - logoSize) % 2 !== 0) logoSize += 1;
  const logoStart = (size - logoSize) / 2;
  const inLogo = (x: number, y: number) =>
    x >= logoStart && x < logoStart + logoSize && y >= logoStart && y < logoStart + logoSize;

  const dots: [number, number][] = [];
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      if (modules[y * size + x] === "1" && !inFinder(x, y) && !inLogo(x, y)) dots.push([x, y]);
    }
  }
  return { dots, finders, logo: { x: logoStart, y: logoStart, size: logoSize } };
}
