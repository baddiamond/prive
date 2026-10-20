import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import AccountRail from "./components/AccountRail";
import ChatList from "./components/ChatList";
import SettingsScreen from "./components/settings/SettingsScreen";
import Welcome from "./components/Welcome";
import { translateCode } from "./lib/errors";
import { hasCommandKey } from "./lib/platform";
import { applyTheme } from "./lib/theme";
import { useSettings } from "./state/settings";
import { useUI } from "./state/ui";

/** Root component: loads preferences, applies the theme and lays out the window (docs/UI.md §1). */
export default function App() {
  const { t } = useTranslation();
  const status = useSettings((s) => s.status);
  const theme = useSettings((s) => s.theme);
  const errorCode = useSettings((s) => s.errorCode);
  const load = useSettings((s) => s.load);
  const screen = useUI((s) => s.screen);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => applyTheme(theme), [theme]);

  useEffect(() => {
    // ⌘, (Ctrl+, elsewhere) opens Settings; Esc goes back to the inbox.
    const onKey = (e: KeyboardEvent) => {
      const ui = useUI.getState();
      if (hasCommandKey(e) && e.key === ",") {
        e.preventDefault();
        ui.openSettings();
      } else if (e.key === "Escape" && ui.screen === "settings") {
        ui.closeSettings();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

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
      <AccountRail />
      {screen === "inbox" ? (
        <>
          <ChatList />
          <Welcome />
        </>
      ) : (
        <SettingsScreen />
      )}
    </div>
  );
}
