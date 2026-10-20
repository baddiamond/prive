import { useEffect, useId, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { toBackendError, translateCode, type BackendError } from "../../lib/errors";
import {
  confirmRecoveryPhrase,
  getRecoveryPhrase,
  pickPositions,
} from "../../lib/recovery";
import { useRecovery, type RecoveryDialogMode } from "../../state/recovery";
import Dialog from "../common/Dialog";
import { CheckIcon, CloseIcon, LockIcon } from "../icons";

type Step = "intro" | "words" | "confirm" | "done";

const primary = "h-10 rounded-control bg-accent px-5 font-bold text-on-accent disabled:cursor-not-allowed disabled:opacity-50";
const secondary = "h-10 rounded-control border-[1.5px] border-border px-4 font-semibold hover:bg-raised";

/** The recovery phrase dialog (ADR 0007), opened from the banner or from Settings. */
export default function RecoveryDialog() {
  const mode = useRecovery((s) => s.dialog);
  const close = useRecovery((s) => s.closeDialog);
  if (!mode) return null;
  return <RecoveryFlow mode={mode} onClose={close} />;
}

function RecoveryFlow({ mode, onClose }: { mode: RecoveryDialogMode; onClose: () => void }) {
  const { t, i18n } = useTranslation();
  const titleId = useId();
  const confirmed = useRecovery((s) => s.confirmed);
  const [step, setStep] = useState<Step>(mode === "save" ? "intro" : "words");
  const [words, setWords] = useState<string[]>([]);
  const [error, setError] = useState<BackendError | null>(null);

  useEffect(() => {
    let current = true;
    getRecoveryPhrase(i18n.resolvedLanguage ?? "en")
      .then((w) => current && setWords(w))
      .catch((err: unknown) => current && setError(toBackendError(err)));
    return () => {
      current = false;
    };
  }, [i18n.resolvedLanguage]);

  // Settings → "Show words" after the phrase was saved only shows the words.
  const steps: Step[] = mode === "save" ? ["intro", "words", "confirm"] : confirmed ? ["words"] : ["words", "confirm"];
  const index = steps.indexOf(step);

  return (
    <Dialog labelledBy={titleId} onClose={onClose} width="max-w-[560px]">
      <header className="flex items-center gap-3 border-b border-divider px-5 py-4">
        <span className="flex h-9 w-9 items-center justify-center rounded-control bg-raised text-accent">
          <LockIcon size={18} />
        </span>
        <div className="flex-1">
          <h1 id={titleId} className="text-heading font-semibold">
            {t("recovery.dialog.title")}
          </h1>
          {index >= 0 && steps.length > 1 && (
            <p className="text-caption text-text-secondary">
              {t("recovery.dialog.step", { current: index + 1, total: steps.length })}
            </p>
          )}
        </div>
        <button
          type="button"
          onClick={onClose}
          aria-label={t("common.close")}
          className="flex h-9 w-9 items-center justify-center rounded-control text-text-secondary hover:bg-raised"
        >
          <CloseIcon size={18} />
        </button>
      </header>

      {error ? (
        <div className="flex flex-col gap-5 p-5">
          <p role="alert" className="text-status-danger">
            {translateCode(t, error.code, error.params)}
          </p>
          <div className="flex justify-end">
            <button type="button" onClick={onClose} className={secondary}>
              {t("common.close")}
            </button>
          </div>
        </div>
      ) : step === "intro" ? (
        <Intro onNext={() => setStep("words")} />
      ) : step === "words" ? (
        <Words
          words={words}
          finalStep={steps[steps.length - 1] === "words"}
          onNext={() => setStep("confirm")}
          onClose={onClose}
        />
      ) : step === "confirm" ? (
        <Confirm onBack={() => setStep("words")} onDone={() => setStep("done")} />
      ) : (
        <Done onClose={onClose} />
      )}
    </Dialog>
  );
}

function Intro({ onNext }: { onNext: () => void }) {
  const { t } = useTranslation();
  const points = [t("recovery.dialog.intro.paper"), t("recovery.dialog.intro.noCopies"), t("recovery.dialog.intro.neverAsked")];
  return (
    <div className="flex flex-col gap-4 p-5">
      <h2 className="text-body-strong font-semibold">{t("recovery.dialog.intro.title")}</h2>
      <p className="leading-relaxed text-text-secondary">{t("recovery.dialog.intro.body")}</p>
      <ul className="flex flex-col gap-2">
        {points.map((p) => (
          <li key={p} className="flex items-start gap-2 leading-relaxed text-text-secondary">
            <CheckIcon size={16} className="mt-0.5 shrink-0 text-status-success" />
            {p}
          </li>
        ))}
      </ul>
      <div className="flex justify-end">
        <button type="button" onClick={onNext} className={primary} data-autofocus autoFocus>
          {t("recovery.dialog.intro.action")}
        </button>
      </div>
    </div>
  );
}

function Words(props: { words: string[]; finalStep: boolean; onNext: () => void; onClose: () => void }) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col gap-4 p-5">
      <h2 className="text-body-strong font-semibold">{t("recovery.dialog.words.title")}</h2>
      <p className="leading-relaxed text-text-secondary">{t("recovery.dialog.words.body")}</p>
      <ol
        aria-label={t("recovery.dialog.words.list")}
        className="grid grid-cols-3 gap-2 rounded-card border border-divider bg-raised p-3"
      >
        {props.words.map((w, i) => (
          <li key={i} className="flex items-baseline gap-2 rounded-control bg-surface px-3 py-2">
            <span className="w-5 shrink-0 text-right font-mono text-caption text-text-faint">{i + 1}</span>
            <span className="font-mono font-semibold text-text">{w}</span>
          </li>
        ))}
      </ol>
      <div className="flex justify-end">
        {props.finalStep ? (
          <button type="button" onClick={props.onClose} className={primary} data-autofocus autoFocus>
            {t("common.done")}
          </button>
        ) : (
          <button
            type="button"
            onClick={props.onNext}
            disabled={props.words.length === 0}
            className={primary}
            data-autofocus
            autoFocus
          >
            {t("recovery.dialog.words.next")}
          </button>
        )}
      </div>
    </div>
  );
}

function Confirm({ onBack, onDone }: { onBack: () => void; onDone: () => void }) {
  const { t } = useTranslation();
  const baseId = useId();
  const markConfirmed = useRecovery((s) => s.markConfirmed);
  const positions = useMemo(() => pickPositions(), []);
  const [typed, setTyped] = useState<Record<number, string>>({});
  const [error, setError] = useState<BackendError | null>(null);
  const [busy, setBusy] = useState(false);
  const complete = positions.every((p) => (typed[p] ?? "").trim() !== "");

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!complete || busy) return;
    setBusy(true);
    setError(null);
    try {
      await confirmRecoveryPhrase(positions.map((position) => ({ position, word: typed[position] ?? "" })));
      markConfirmed();
      onDone();
    } catch (err) {
      setError(toBackendError(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <form onSubmit={(e) => void onSubmit(e)} className="flex flex-col gap-4 p-5">
      <h2 className="text-body-strong font-semibold">{t("recovery.dialog.confirm.title")}</h2>
      <p className="leading-relaxed text-text-secondary">{t("recovery.dialog.confirm.body")}</p>
      <div className="grid grid-cols-3 gap-3">
        {positions.map((p, i) => (
          <div key={p} className="flex flex-col gap-1.5">
            <label htmlFor={`${baseId}-${p}`} className="text-caption font-semibold text-text-secondary">
              {t("recovery.dialog.confirm.word", { position: p })}
            </label>
            <input
              id={`${baseId}-${p}`}
              value={typed[p] ?? ""}
              onChange={(e) => setTyped({ ...typed, [p]: e.target.value })}
              autoComplete="off"
              autoCapitalize="off"
              autoCorrect="off"
              spellCheck={false}
              autoFocus={i === 0}
              data-autofocus={i === 0 ? true : undefined}
              className="h-10 rounded-control border-[1.5px] border-border bg-raised px-3 font-mono text-text"
            />
          </div>
        ))}
      </div>
      {error && (
        <p role="alert" className="text-status-danger">
          {translateCode(t, error.code, error.params)}
        </p>
      )}
      <div className="flex justify-between gap-2">
        <button type="button" onClick={onBack} className={secondary}>
          {t("recovery.dialog.confirm.back")}
        </button>
        <button type="submit" disabled={!complete || busy} className={primary}>
          {t("recovery.dialog.confirm.action")}
        </button>
      </div>
    </form>
  );
}

function Done({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col items-center gap-3 p-6 text-center">
      <span className="flex h-12 w-12 items-center justify-center rounded-full bg-raised text-status-success">
        <CheckIcon size={24} />
      </span>
      <h2 role="status" className="text-body-strong font-semibold">
        {t("recovery.dialog.done.title")}
      </h2>
      <p className="max-w-sm leading-relaxed text-text-secondary">{t("recovery.dialog.done.body")}</p>
      <button type="button" onClick={onClose} className={`${primary} mt-2`} data-autofocus autoFocus>
        {t("common.done")}
      </button>
    </div>
  );
}
