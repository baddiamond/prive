import { useTranslation } from "react-i18next";
import { shortcutLabel } from "../lib/platform";
import LogoMark from "./LogoMark";

/** Conversation area while there are no accounts (docs/UI.md, "empty" state). */
export default function Welcome() {
  const { t } = useTranslation();
  const tips = [
    { key: shortcutLabel("N"), label: t("inbox.welcome.tips.addAccount") },
    { key: shortcutLabel("K"), label: t("inbox.welcome.tips.search") },
    { key: shortcutLabel(","), label: t("inbox.welcome.tips.settings") },
  ];

  return (
    <main className="flex min-w-0 flex-1 flex-col items-center justify-center gap-5 bg-bg p-6 text-center">
      <LogoMark width={72} height={72} tone="muted" />
      <h2 className="text-title font-bold tracking-[-0.01em]">{t("inbox.welcome.title")}</h2>
      <p className="max-w-[380px] leading-relaxed text-text-secondary">{t("inbox.welcome.body")}</p>
      <ul className="flex flex-wrap justify-center gap-2">
        {tips.map((tip) => (
          <li
            key={tip.key}
            className="flex items-center gap-2 rounded-control border border-divider px-3 py-2 text-caption text-text-secondary"
          >
            <kbd className="rounded-tag border border-border px-1.5 py-0.5 font-mono text-micro font-semibold text-text">
              {tip.key}
            </kbd>
            {tip.label}
          </li>
        ))}
      </ul>
    </main>
  );
}
