import { useId, useState } from "react";
import { useTranslation } from "react-i18next";
import { toBackendError, translateCode, type BackendError } from "../../lib/errors";
import { unlockWithPhrase } from "../../lib/recovery";
import { useAppInfo } from "../../state/appInfo";
import LogoMark from "../LogoMark";

/**
 * Shown when the data key is not in the keychain (new computer, reinstalled system): the
 * recovery phrase unlocks the data and Privé starts normally (ADR 0007).
 */
export default function UnlockScreen() {
  const { t } = useTranslation();
  const info = useAppInfo();
  const fieldId = useId();
  const hintId = useId();
  const [phrase, setPhrase] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<BackendError | null>(null);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (busy || phrase.trim() === "") return;
    setBusy(true);
    setError(null);
    try {
      await unlockWithPhrase(phrase);
      // Every store loads from scratch against the unlocked data.
      window.location.reload();
    } catch (err) {
      setError(toBackendError(err));
      setBusy(false);
    }
  };

  return (
    <main className="relative flex h-full items-center justify-center overflow-y-auto p-8">
      <form onSubmit={(e) => void onSubmit(e)} className="flex w-full max-w-[520px] flex-col gap-5">
        <LogoMark width={48} height={48} />
        <h1 className="text-title font-semibold tracking-[-0.01em]">{t("unlock.title")}</h1>
        <p className="leading-relaxed text-text-secondary">{t("unlock.body")}</p>

        <div className="flex flex-col gap-2">
          <label htmlFor={fieldId} className="text-caption font-semibold text-text-secondary">
            {t("unlock.label")}
          </label>
          <textarea
            id={fieldId}
            value={phrase}
            onChange={(e) => setPhrase(e.target.value)}
            onKeyDown={(e) => {
              // The words fit on one line of thought: Enter unlocks, Shift+Enter adds a line.
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                e.currentTarget.form?.requestSubmit();
              }
            }}
            aria-describedby={hintId}
            aria-invalid={error !== null}
            rows={3}
            autoFocus
            autoComplete="off"
            autoCapitalize="off"
            autoCorrect="off"
            spellCheck={false}
            className="resize-none rounded-control border-[1.5px] border-border bg-raised px-3 py-2.5 font-mono leading-relaxed text-text"
          />
          <p id={hintId} className="text-caption text-text-faint">
            {t("unlock.hint")}
          </p>
        </div>

        {error && (
          <p role="alert" className="text-status-danger">
            {translateCode(t, error.code, error.params)}
          </p>
        )}

        <button
          type="submit"
          disabled={busy || phrase.trim() === ""}
          className="h-10 self-start rounded-control bg-accent px-5 font-bold text-on-accent disabled:cursor-not-allowed disabled:opacity-50"
        >
          {busy ? t("unlock.working") : t("unlock.action")}
        </button>

        <p className="border-t border-divider pt-4 text-caption leading-relaxed text-text-faint">
          {t("unlock.lost")}{" "}
          {info && <code className="select-text break-all font-mono text-text-secondary">{info.dataDir}</code>}
        </p>
      </form>
    </main>
  );
}
