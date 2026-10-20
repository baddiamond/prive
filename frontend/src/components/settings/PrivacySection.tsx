import { useTranslation } from "react-i18next";
import { useAppInfo } from "../../state/appInfo";
import { CheckIcon } from "../icons";

/** The local-first promises and where the data lives. */
export default function PrivacySection() {
  const { t } = useTranslation();
  const info = useAppInfo();
  const statements = [
    t("settings.privacy.local"),
    t("settings.privacy.noServers"),
    t("settings.privacy.networkOnly"),
  ];

  return (
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
  );
}
