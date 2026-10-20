import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import AccountRail from "./components/AccountRail";
import AddAccountDialog from "./components/accounts/AddAccountDialog";
import ChatList from "./components/ChatList";
import ConversationPlaceholder from "./components/chats/ConversationPlaceholder";
import LogoMark from "./components/LogoMark";
import Onboarding from "./components/onboarding/Onboarding";
import RecoveryDialog from "./components/recovery/RecoveryDialog";
import UnlockScreen from "./components/recovery/UnlockScreen";
import SettingsScreen from "./components/settings/SettingsScreen";
import Welcome from "./components/Welcome";
import { errorCode, translateCode } from "./lib/errors";
import { hasCommandKey } from "./lib/platform";
import { applyTheme } from "./lib/theme";
import { linkedAccounts, useAccounts } from "./state/accounts";
import { useChats } from "./state/chats";
import { needsOnboarding, useOnboarding } from "./state/onboarding";
import { useRecovery } from "./state/recovery";
import { useSettings } from "./state/settings";
import { useUI } from "./state/ui";

/** Root component: loads local state, applies the theme and shows onboarding or the inbox. */
export default function App() {
  const { t } = useTranslation();
  const status = useSettings((s) => s.status);
  const theme = useSettings((s) => s.theme);
  const settingsError = useSettings((s) => s.errorCode);
  const loadSettings = useSettings((s) => s.load);
  const onboarding = useOnboarding((s) => s.state);
  const [dataError, setDataError] = useState<string | null>(null);
  const [dataReady, setDataReady] = useState(false);

  useEffect(() => {
    void loadSettings();
  }, [loadSettings]);

  useEffect(() => applyTheme(theme), [theme]);

  useEffect(() => {
    if (status !== "ready") return;
    Promise.all([
      useOnboarding.getState().load(),
      useAccounts.getState().load(),
      useChats.getState().load(),
      useRecovery.getState().load(),
    ])
      .then(() => setDataReady(true))
      .catch((err: unknown) => setDataError(errorCode(err)));
  }, [status]);

  useShortcuts(dataReady && !needsOnboarding(onboarding));

  const failure = status === "error" ? (settingsError ?? "errors.unknown") : dataError;
  if (failure === lockedCode) {
    return <UnlockScreen />;
  }
  if (failure) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-5 p-8 text-center">
        <LogoMark width={56} height={56} />
        <p role="alert" className="max-w-md leading-relaxed text-text">
          {translateCode(t, failure)}
        </p>
      </div>
    );
  }

  if (status === "loading" || !dataReady) {
    return (
      <div className="flex h-full items-center justify-center text-text-secondary" role="status">
        {t("app.loading")}
      </div>
    );
  }

  if (needsOnboarding(onboarding)) {
    return <Onboarding />;
  }

  return <Shell />;
}

/** The backend's code for data whose key is not in the keychain (internal/keys). */
const lockedCode = "errors.encryption.locked";

function Shell() {
  const screen = useUI((s) => s.screen);
  const hasSelection = useUI((s) => s.selectedChat !== null);
  return (
    <div className="flex h-full">
      <AccountRail />
      {screen === "inbox" ? (
        <>
          <ChatList />
          {hasSelection ? <ConversationPlaceholder /> : <Welcome />}
        </>
      ) : (
        <SettingsScreen />
      )}
      <AddAccountDialog />
      <RecoveryDialog />
    </div>
  );
}

/**
 * Window shortcuts (docs/UI.md §4): ⌘, settings, ⌘N add account, ⌘0 all inboxes,
 * ⌘1–9 an account, Esc back from settings. Ctrl replaces ⌘ outside macOS.
 */
function useShortcuts(enabled: boolean) {
  useEffect(() => {
    if (!enabled) return;
    const onKey = (e: KeyboardEvent) => {
      const ui = useUI.getState();
      if (ui.addAccount.open || useRecovery.getState().dialog) return; // dialogs handle their own keys
      if (hasCommandKey(e)) {
        if (e.key === ",") {
          e.preventDefault();
          ui.openSettings();
        } else if (e.key.toLowerCase() === "n") {
          e.preventDefault();
          ui.openAddAccount();
        } else if (/^[0-9]$/.test(e.key)) {
          e.preventDefault();
          const n = Number(e.key);
          const accounts = linkedAccounts(useAccounts.getState().accounts);
          if (n === 0) ui.setScope("all");
          else if (accounts[n - 1]) ui.setScope(accounts[n - 1]!.account.id);
        }
      } else if (e.key === "Escape" && ui.screen === "settings") {
        ui.closeSettings();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [enabled]);
}
