import { useTranslation } from "react-i18next";
import { useAppInfo } from "../../state/appInfo";
import LogoMark from "../LogoMark";

/** App name, version and license. */
export default function AboutSection() {
  const { t } = useTranslation();
  const info = useAppInfo();

  return (
    <section className="flex items-center gap-4 rounded-card border border-divider bg-surface p-5">
      <LogoMark width={48} height={48} />
      <div className="flex flex-col gap-1">
        <strong className="text-heading font-semibold">{t("app.name")}</strong>
        <span className="text-text-secondary">{t("settings.about.tagline")}</span>
        <span className="text-caption text-text-faint">
          {info && t("settings.about.version", { version: info.version })} · {t("settings.about.license")}
        </span>
      </div>
    </section>
  );
}
