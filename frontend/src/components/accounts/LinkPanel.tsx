import { useId } from "react";
import { useTranslation } from "react-i18next";
import i18n from "../../i18n";
import type { AccountColor, Manifest } from "../../lib/accounts";
import { translateCode } from "../../lib/errors";
import { CheckIcon } from "../icons";
import AccountTile from "./AccountTile";
import ColorPicker from "./ColorPicker";
import QRCodeView from "./QRCodeView";
import type { LinkFlow } from "./useLinkFlow";

interface LinkPanelProps {
  manifest: Manifest;
  flow: LinkFlow;
  name: string;
  color: AccountColor;
  onName: (name: string) => void;
  onColor: (color: AccountColor) => void;
}

/** Live QR code, the phone steps, and the account name and color with a rail preview. */
export default function LinkPanel({ manifest, flow, name, color, onName, onColor }: LinkPanelProps) {
  const { t } = useTranslation();
  const nameId = useId();
  const steps = engineSteps(manifest.id);
  const label = name.trim() || manifest.name;

  const status =
    flow.phase === "linked"
      ? t("link.status.linked")
      : flow.phase === "failed"
        ? translateCode(t, flow.failure ?? "errors.unknown")
        : flow.phase === "starting"
          ? t("link.status.starting")
          : t("link.status.waiting");

  return (
    <div className="grid gap-8 md:grid-cols-[260px_1fr]">
      <div className="flex flex-col items-center gap-3">
        <QRCodeView
          value={flow.phase === "failed" ? null : flow.qr}
          overlay={
            flow.phase === "linked" ? (
              <span className="flex h-16 w-16 items-center justify-center rounded-full bg-accent text-on-accent">
                <CheckIcon size={34} />
              </span>
            ) : undefined
          }
        />
        <p
          role="status"
          className={`text-center text-caption ${flow.phase === "failed" ? "text-status-danger" : "text-text-secondary"}`}
        >
          {status}
        </p>
        {flow.phase === "failed" && (
          <button
            type="button"
            onClick={flow.retry}
            className="h-9 rounded-control border-[1.5px] border-border px-4 font-semibold hover:bg-raised"
          >
            {t("link.tryAgain")}
          </button>
        )}
      </div>

      <div className="flex flex-col gap-6">
        {steps.length > 0 && (
          <ol className="flex list-decimal flex-col gap-2 pl-5 leading-relaxed">
            {steps.map((s) => (
              <li key={s}>{s}</li>
            ))}
          </ol>
        )}
        <div className="flex flex-col gap-2">
          <label htmlFor={nameId} className="text-caption font-semibold text-text-secondary">
            {t("link.accountName")}
          </label>
          <input
            id={nameId}
            type="text"
            value={name}
            maxLength={40}
            placeholder={manifest.name}
            onChange={(e) => onName(e.target.value)}
            className="h-10 rounded-control border-[1.5px] border-border bg-raised px-3 text-text placeholder:text-text-faint"
          />
        </div>
        <ColorPicker value={color} onChange={onColor} />
        <div className="flex items-center gap-3 rounded-card border border-divider p-3">
          <AccountTile label={label} color={color} size={40} />
          <span className="flex flex-col">
            <span className="font-semibold">{label}</span>
            <span className="text-caption text-text-secondary">{t("link.preview", { network: manifest.name })}</span>
          </span>
        </div>
      </div>
    </div>
  );
}

/** Network-specific phone steps from engines.<id>.loginSteps (step1, step2, …). */
function engineSteps(engineId: string): string[] {
  const out: string[] = [];
  for (let i = 1; i <= 5; i++) {
    const key = `engines.${engineId}.loginSteps.step${i}`;
    if (!i18n.exists(key)) break;
    out.push(i18n.t(key as "app.name"));
  }
  return out;
}
