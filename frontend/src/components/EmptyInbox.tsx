import { useTranslation } from "react-i18next";
import { InboxIcon } from "./icons";

/** Shown in the main area while there are no accounts or chats yet. */
export default function EmptyInbox() {
  const { t } = useTranslation();

  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
      <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-indigo-50 text-indigo-600 dark:bg-indigo-950 dark:text-indigo-300">
        <InboxIcon width={28} height={28} />
      </div>
      <h1 className="text-lg font-semibold">{t("inbox.empty.title")}</h1>
      <p className="max-w-sm text-sm text-neutral-500 dark:text-neutral-400">{t("inbox.empty.body")}</p>
    </div>
  );
}
