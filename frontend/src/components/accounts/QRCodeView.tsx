import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { renderQR, type QRCode } from "../../lib/accounts";

interface QRCodeViewProps {
  /** QR payload from the engine, or null while waiting for the first one. */
  value: string | null;
  /** Shown over the code (e.g. a check mark once linked). */
  overlay?: React.ReactNode;
}

/**
 * Draws a login QR code. It is always dark on white, whatever the theme: phone cameras
 * read inverted codes unreliably, so this is the one place that uses fixed white and black.
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

  const quiet = 2;
  return (
    <div
      role="img"
      aria-label={t("link.qrAlt")}
      className="relative flex aspect-square w-full max-w-[260px] items-center justify-center rounded-card bg-white p-3"
    >
      {value && code ? (
        <svg
          viewBox={`${-quiet} ${-quiet} ${code.size + quiet * 2} ${code.size + quiet * 2}`}
          shapeRendering="crispEdges"
          className="h-full w-full"
          aria-hidden
        >
          <path d={code.path} fill="black" />
        </svg>
      ) : (
        <span className="h-8 w-8 animate-spin rounded-full border-2 border-black/20 border-t-black/70 motion-reduce:animate-none" />
      )}
      {overlay && (
        <div className="absolute inset-0 flex items-center justify-center rounded-card bg-white/90">{overlay}</div>
      )}
    </div>
  );
}
