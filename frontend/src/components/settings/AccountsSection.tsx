import { useTranslation } from "react-i18next";
import { PlusIcon } from "../icons";

/** Linked accounts. Rows (rename, log out) arrive with WhatsApp login in phase 2. */
export default function AccountsSection() {
  const { t } = useTranslation();

  return (
    <section className="flex flex-col gap-3">
      <div className="flex items-center justify-end">
        <button
          type="button"
          disabled
          title={t("common.nextVersion")}
          className="flex h-[34px] items-center gap-2 rounded-control bg-accent px-3.5 font-semibold text-on-accent disabled:cursor-not-allowed disabled:opacity-60"
        >
          <PlusIcon size={16} />
          {t("settings.accounts.add")}
        </button>
      </div>
      <div className="rounded-card border border-divider px-4 py-6 text-center text-text-secondary">
        {t("settings.accounts.empty")}
      </div>
    </section>
  );
}
