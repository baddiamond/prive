import { useTranslation } from "react-i18next";
import { useAppInfo } from "../../state/appInfo";
import { useRecovery } from "../../state/recovery";
import { CheckIcon, LockIcon } from "../icons";

/** The local-first promises, encryption and recovery phrase, and where the data lives. */
export default function PrivacySection() {
  const { t } = useTranslation();
  const info = useAppInfo();
  const statements = [
    t("settings.privacy.local"),
    t("settings.privacy.noServers"),
    t("settings.privacy.networkOnly"),
  ];

  return (
    <div className="flex flex-col gap-4">
      <section className="flex flex-col gap-2.5 rounded-card border border-divider bg-surface p-4">
        <ul className="flex flex-col gap-2.5">
          {statements.map((text) => (
            <li key={text} className="flex items-start gap-2 leading-relaxed text-text-secondary">
              <CheckIcon size={16} className="mt-0.5 shrink-0 text-status-success" />
              {text}
            </li>
          ))}
        </ul>
        {info && (
          <p className="text-caption text-text-faint">
            {t("settings.privacy.dataPath")}{" "}
            <code className="select-text break-all font-mono text-text-secondary">{info.dataDir}</code>
          </p>
        )}
      </section>
      <EncryptionRows />
    </div>
  );
}

/** Encryption status and the recovery phrase (ADR 0007). */
function EncryptionRows() {
  const { t } = useTranslation();
  const confirmed = useRecovery((s) => s.confirmed);
  const openDialog = useRecovery((s) => s.openDialog);

  return (
    <ul className="overflow-hidden rounded-card border border-divider bg-surface">
      <li className="flex items-start gap-3 px-4 py-3">
        <span className="mt-0.5 shrink-0 text-status-success">
          <LockIcon size={18} />
        </span>
        <span className="flex flex-col gap-1">
          <span className="font-semibold">{t("settings.privacy.encryption")}</span>
          <span className="leading-relaxed text-text-secondary">{t("settings.privacy.encryptionOn")}</span>
          <span className="text-caption leading-relaxed text-text-faint">{t("settings.privacy.encryptionScope")}</span>
        </span>
      </li>
      <li className="flex flex-wrap items-center gap-3 border-t border-divider px-4 py-3">
        <span className="flex min-w-[160px] flex-1 flex-col gap-0.5">
          <span className="font-semibold">{t("settings.privacy.recoveryPhrase")}</span>
          <span className="flex items-center gap-1.5 text-caption text-text-secondary">
            <span aria-hidden className={`h-2 w-2 rounded-full ${confirmed ? "bg-status-success" : "bg-status-warning"}`} />
            {confirmed ? t("settings.privacy.recoverySaved") : t("settings.privacy.recoveryNotSaved")}
          </span>
        </span>
        <button
          type="button"
          onClick={() => openDialog(confirmed ? "show" : "save")}
          className="h-[34px] rounded-control border-[1.5px] border-border px-3.5 font-semibold hover:bg-raised"
        >
          {confirmed ? t("settings.privacy.recoveryShow") : t("settings.privacy.recoverySave")}
        </button>
      </li>
    </ul>
  );
}
