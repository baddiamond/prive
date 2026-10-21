import { useTranslation } from "react-i18next";
import { linkedAccounts, useAccounts } from "../../state/accounts";
import { useChats } from "../../state/chats";
import { hasIssue } from "../accounts/accountLabel";
import { useRecovery } from "../../state/recovery";
import { LockIcon } from "../icons";

/**
 * Calm, non-blocking offer to save the recovery phrase (ADR 0007). It appears once there is
 * data to lose: chats exist and no account is still importing. It waits while an account
 * needs attention, so it never pushes those banners and the chats out of view. "Later" hides it until the
 * next start; it stops for good once the phrase is confirmed.
 */
export default function RecoveryBanner() {
  const { t } = useTranslation();
  const { loaded, confirmed, postponed, postpone, openDialog } = useRecovery();
  const hasChats = useChats((s) => s.chats.length > 0);
  const importing = useChats((s) => Object.keys(s.syncProgress).length > 0);
  // Accounts that need attention come first; the phrase can wait until they are fine.
  const issues = useAccounts((s) => linkedAccounts(s.accounts).some(hasIssue));
  if (!loaded || confirmed || postponed || !hasChats || importing || issues) return null;

  return (
    <div role="status" className="mx-3 mb-2 flex items-start gap-2.5 rounded-control border border-accent px-3 py-2.5">
      <span className="mt-0.5 shrink-0 text-accent">
        <LockIcon size={18} />
      </span>
      <span className="flex flex-1 flex-col gap-1.5">
        <span className="text-caption font-semibold text-text">{t("recovery.banner.title")}</span>
        <span className="text-caption leading-relaxed text-text-secondary">{t("recovery.banner.body")}</span>
        <span className="flex gap-2">
          <button
            type="button"
            onClick={() => openDialog("save")}
            className="h-[30px] rounded-control bg-accent px-3 text-caption font-bold text-on-accent"
          >
            {t("recovery.banner.action")}
          </button>
          <button
            type="button"
            onClick={postpone}
            className="h-[30px] rounded-control px-3 text-caption font-semibold text-text-secondary hover:bg-raised hover:text-text"
          >
            {t("recovery.banner.later")}
          </button>
        </span>
      </span>
    </div>
  );
}
