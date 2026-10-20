import { useEffect, useRef, type ReactNode } from "react";

interface DialogProps {
  /** id of the element that names the dialog. */
  labelledBy: string;
  onClose: () => void;
  children: ReactNode;
  /** Maximum width class, e.g. "max-w-[720px]". */
  width?: string;
}

/**
 * Modal dialog on a scrim (docs/UI.md §1). Esc closes it, focus moves inside on open (to the
 * element marked data-autofocus, else the first focusable one) and
 * returns to the previously focused element on close, and Tab stays inside.
 */
export default function Dialog({ labelledBy, onClose, children, width = "max-w-[520px]" }: DialogProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    // An element marked data-autofocus (the step's main action) wins over the first one.
    const first =
      ref.current?.querySelector<HTMLElement>("[data-autofocus]") ?? ref.current?.querySelector<HTMLElement>(focusable);
    (first ?? ref.current)?.focus();
    return () => previous?.focus();
  }, []);

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      e.stopPropagation();
      onClose();
      return;
    }
    if (e.key !== "Tab" || !ref.current) return;
    const items = Array.from(ref.current.querySelectorAll<HTMLElement>(focusable));
    if (items.length === 0) return;
    const first = items[0]!;
    const last = items[items.length - 1]!;
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-scrim px-4 py-16">
      <div
        ref={ref}
        role="dialog"
        aria-modal="true"
        aria-labelledby={labelledBy}
        tabIndex={-1}
        onKeyDown={onKeyDown}
        className={`w-full ${width} rounded-dialog border border-divider bg-surface shadow-2xl`}
      >
        {children}
      </div>
    </div>
  );
}

const focusable =
  'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';
