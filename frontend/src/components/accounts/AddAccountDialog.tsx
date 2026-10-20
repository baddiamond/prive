import { useId, useState } from "react";
import { useTranslation } from "react-i18next";
import type { AccountColor } from "../../lib/accounts";
import { errorCode, translateCode } from "../../lib/errors";
import { useAccounts } from "../../state/accounts";
import { useUI } from "../../state/ui";
import Dialog from "../common/Dialog";
import ExternalLink from "../common/ExternalLink";
import { projectLinks } from "../../lib/external";
import { BackIcon, CloseIcon } from "../icons";
import LinkPanel from "./LinkPanel";
import NetworkGrid from "./NetworkGrid";
import { nextColor, saveIdentity } from "./identity";
import { useLinkFlow } from "./useLinkFlow";

/** Add account (docs/UI.md §2.6): pick a network, then link it. Also links an account again. */
export default function AddAccountDialog() {
  const dialog = useUI((s) => s.addAccount);
  if (!dialog.open) return null;
  return <AddAccountContent relinkAccountId={dialog.relinkAccountId} initialEngine={dialog.engineId} />;
}

function AddAccountContent({ relinkAccountId, initialEngine }: { relinkAccountId?: string; initialEngine?: string }) {
  const { t } = useTranslation();
  const titleId = useId();
  const close = useUI((s) => s.closeAddAccount);
  const engines = useAccounts((s) => s.engines);
  const accounts = useAccounts((s) => s.accounts);
  const relinking = accounts.find((s) => s.account.id === relinkAccountId);

  const [engineId, setEngineId] = useState<string | null>(
    relinking?.account.engineId ?? initialEngine ?? (engines.length === 1 ? engines[0]!.id : null),
  );
  const [step, setStep] = useState<"pick" | "link">(relinking ? "link" : "pick");
  const manifest = engines.find((e) => e.id === engineId);

  const title = step === "pick" ? t("addAccount.title") : t("link.title", { network: manifest?.name ?? "" });

  return (
    <Dialog labelledBy={titleId} onClose={close} width="max-w-[760px]">
      <header className="flex items-center gap-3 border-b border-divider px-5 py-4">
        {step === "link" && !relinking && (
          <button
            type="button"
            onClick={() => setStep("pick")}
            aria-label={t("common.back")}
            title={t("common.back")}
            className="flex h-9 w-9 items-center justify-center rounded-control text-text-secondary hover:bg-raised"
          >
            <BackIcon size={18} />
          </button>
        )}
        <div className="flex-1">
          <h1 id={titleId} className="text-heading font-semibold">
            {title}
          </h1>
          <p className="text-caption text-text-secondary">
            {step === "pick" ? t("addAccount.subtitle") : t("link.body")}
          </p>
        </div>
        {!relinking && (
          <span className="text-caption text-text-faint">
            {t("addAccount.stepOf", { step: step === "pick" ? 1 : 2, total: 2 })}
          </span>
        )}
        <button
          type="button"
          onClick={close}
          aria-label={t("common.close")}
          title={t("common.close")}
          className="flex h-9 w-9 items-center justify-center rounded-control text-text-secondary hover:bg-raised"
        >
          <CloseIcon size={18} />
        </button>
      </header>

      {step === "pick" ? (
        <div className="flex flex-col gap-4 p-5">
          <NetworkGrid engines={engines} selected={engineId} onSelect={setEngineId} />
          <p className="text-caption text-text-secondary">
            {t("addAccount.buildEngine")}{" "}
            <ExternalLink href={projectLinks.addingAnEngine}>
              {t("addAccount.buildEngineLink")}
            </ExternalLink>
          </p>
          <div className="flex justify-end">
            <button
              type="button"
              disabled={!manifest}
              onClick={() => setStep("link")}
              className="h-10 rounded-control bg-accent px-5 font-bold text-on-accent disabled:cursor-not-allowed disabled:opacity-50"
            >
              {t("common.continue")}
            </button>
          </div>
        </div>
      ) : (
        manifest && (
          <LinkStep
            key={manifest.id}
            engineId={manifest.id}
            relinkAccountId={relinkAccountId}
            initialName={relinking?.account.displayName ?? ""}
            initialColor={relinking?.account.color ?? nextColor(accounts.map((s) => s.account.color))}
            onDone={close}
          />
        )
      )}
    </Dialog>
  );
}

interface LinkStepProps {
  engineId: string;
  relinkAccountId?: string;
  initialName: string;
  initialColor: AccountColor;
  onDone: () => void;
}

function LinkStep({ engineId, relinkAccountId, initialName, initialColor, onDone }: LinkStepProps) {
  const { t } = useTranslation();
  const manifest = useAccounts((s) => s.engines.find((e) => e.id === engineId))!;
  const [name, setName] = useState(initialName);
  const [color, setColor] = useState(initialColor);
  const [saveError, setSaveError] = useState<string | null>(null);
  const flow = useLinkFlow(engineId, initialColor, relinkAccountId);

  const finish = async () => {
    if (!flow.accountId) return;
    try {
      await saveIdentity(flow.accountId, name, color);
      onDone();
    } catch (err) {
      setSaveError(errorCode(err));
    }
  };

  return (
    <div className="flex flex-col gap-5 p-5">
      <LinkPanel manifest={manifest} flow={flow} name={name} color={color} onName={setName} onColor={setColor} />
      {saveError && (
        <p role="alert" className="text-status-danger">
          {translateCode(t, saveError)}
        </p>
      )}
      <div className="flex justify-end">
        <button
          type="button"
          disabled={flow.phase !== "linked"}
          onClick={() => void finish()}
          className="h-10 rounded-control bg-accent px-5 font-bold text-on-accent disabled:cursor-not-allowed disabled:opacity-50"
        >
          {t("common.done")}
        </button>
      </div>
    </div>
  );
}
