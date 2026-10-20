import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import AccountRail, { type View } from "./components/AccountRail";
import EmptyInbox from "./components/EmptyInbox";
import Settings from "./components/Settings";
import { translateCode } from "./lib/errors";
import { applyTheme } from "./lib/theme";
import { useSettings } from "./state/settings";

/** Root component: loads preferences, applies the theme and lays out the window. */
export default function App() {
  const { t } = useTranslation();
  const status = useSettings((s) => s.status);
  const theme = useSettings((s) => s.theme);
  const errorCode = useSettings((s) => s.errorCode);
  const load = useSettings((s) => s.load);
  const [view, setView] = useState<View>("inbox");

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => applyTheme(theme), [theme]);

  if (status === "loading") {
    return (
      <div className="flex h-full items-center justify-center text-text-secondary" role="status">
        {t("app.loading")}
      </div>
    );
  }

  if (status === "error") {
    return (
      <div className="flex h-full items-center justify-center p-8 text-status-danger" role="alert">
        {translateCode(t, errorCode ?? "errors.unknown")}
      </div>
    );
  }

  return (
    <div className="flex h-full">
      <AccountRail view={view} onSelect={setView} />
      <main className="flex min-w-0 flex-1 flex-col">
        {view === "inbox" ? <EmptyInbox /> : <Settings />}
      </main>
    </div>
  );
}
