import { useId, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  maskRemoteId,
  reconnectAccount,
  removeAccount,
  updateAccount,
  type AccountColor,
  type AccountStatus,
} from "../../lib/accounts";
import { errorCode, translateCode } from "../../lib/errors";
import { linkedAccounts, useAccounts } from "../../state/accounts";
import { useUI } from "../../state/ui";
import AccountTile from "../accounts/AccountTile";
import { accountLabel } from "../accounts/accountLabel";
import ColorPicker from "../accounts/ColorPicker";
import Dialog from "../common/Dialog";
import { PlusIcon } from "../icons";

/** Linked accounts with their state: rename, link again, retry, log out (docs/UI.md §2.7). */
export default function AccountsSection() {
  const { t } = useTranslation();
  const openAddAccount = useUI((s) => s.openAddAccount);
  const accounts = linkedAccounts(useAccounts((s) => s.accounts));
  const [renaming, setRenaming] = useState<AccountStatus | null>(null);
  const [loggingOut, setLoggingOut] = useState<AccountStatus | null>(null);

  return (
    <section className="flex flex-col gap-3">
      <div className="flex items-center justify-end">
        <button
          type="button"
          onClick={() => openAddAccount()}
          className="flex h-[34px] items-center gap-2 rounded-control bg-accent px-3.5 font-semibold text-on-accent"
        >
          <PlusIcon size={16} />
          {t("settings.accounts.add")}
        </button>
      </div>
      {accounts.length === 0 ? (
        <div className="rounded-card border border-divider px-4 py-6 text-center text-text-secondary">
          {t("settings.accounts.empty")}
        </div>
      ) : (
        <ul className="overflow-hidden rounded-card border border-divider">
          {accounts.map((s, i) => (
            <AccountRow
              key={s.account.id}
              status={s}
              first={i === 0}
              onRename={() => setRenaming(s)}
              onLogOut={() => setLoggingOut(s)}
              onLinkAgain={() => openAddAccount(s.account.id)}
            />
          ))}
        </ul>
      )}
      {renaming && <RenameDialog status={renaming} onClose={() => setRenaming(null)} />}
      {loggingOut && <LogOutDialog status={loggingOut} onClose={() => setLoggingOut(null)} />}
    </section>
  );
}

interface AccountRowProps {
  status: AccountStatus;
  first: boolean;
  onRename: () => void;
  onLogOut: () => void;
  onLinkAgain: () => void;
}

const stateDot: Record<string, string> = {
  connected: "bg-status-success",
  reconnecting: "bg-status-warning",
  connecting: "bg-text-faint",
  disconnected: "bg-text-faint",
  loggedOut: "bg-status-danger",
  error: "bg-status-danger",
};

function AccountRow({ status, first, onRename, onLogOut, onLinkAgain }: AccountRowProps) {
  const { t } = useTranslation();
  const label = accountLabel(status);
  const masked = maskRemoteId(status.account.remoteId);
  const outlined = "h-8 rounded-control border-[1.5px] px-3 text-caption font-semibold";

  return (
    <li className={`flex flex-wrap items-center gap-3 px-4 py-3 ${first ? "" : "border-t border-divider"}`}>
      <AccountTile label={label} color={status.account.color} size={36} />
      <span className="flex min-w-[160px] flex-1 flex-col gap-0.5">
        <span className="font-semibold">{label}</span>
        <span className="text-caption text-text-secondary">
          {status.manifest.name}
          {masked && ` · ${masked}`}
        </span>
      </span>
      <span className="flex items-center gap-1.5 text-caption text-text-secondary">
        <span aria-hidden className={`h-2 w-2 rounded-full ${stateDot[status.state] ?? "bg-text-faint"}`} />
        {t(`account.state.${status.state}`)}
      </span>
      {status.state === "loggedOut" && (
        <button type="button" onClick={onLinkAgain} className={`${outlined} border-border hover:bg-raised`}>
          {t("chatList.banner.loggedOut.action")}
        </button>
      )}
      {status.state === "error" && (
        <button
          type="button"
          onClick={() => void reconnectAccount(status.account.id)}
          className={`${outlined} border-border hover:bg-raised`}
        >
          {t("chatList.banner.error.action")}
        </button>
      )}
      <button type="button" onClick={onRename} className={`${outlined} border-border hover:bg-raised`}>
        {t("settings.accounts.rename")}
      </button>
      <button type="button" onClick={onLogOut} className={`${outlined} border-status-danger text-status-danger`}>
        {t("settings.accounts.logOut")}
      </button>
    </li>
  );
}

function RenameDialog({ status, onClose }: { status: AccountStatus; onClose: () => void }) {
  const { t } = useTranslation();
  const titleId = useId();
  const nameId = useId();
  const [name, setName] = useState(status.account.displayName);
  const [color, setColor] = useState<AccountColor>(status.account.color);
  const [error, setError] = useState<string | null>(null);

  const save = async () => {
    try {
      await updateAccount(status.account.id, name, color);
      onClose();
    } catch (err) {
      setError(errorCode(err));
    }
  };

  return (
    <Dialog labelledBy={titleId} onClose={onClose}>
      <form
        className="flex flex-col gap-5 p-5"
        onSubmit={(e) => {
          e.preventDefault();
          void save();
        }}
      >
        <h2 id={titleId} className="text-heading font-semibold">
          {t("settings.accounts.renameTitle")}
        </h2>
        <div className="flex flex-col gap-2">
          <label htmlFor={nameId} className="text-caption font-semibold text-text-secondary">
            {t("link.accountName")}
          </label>
          <input
            id={nameId}
            value={name}
            maxLength={40}
            placeholder={status.manifest.name}
            onChange={(e) => setName(e.target.value)}
            className="h-10 rounded-control border-[1.5px] border-border bg-raised px-3 text-text placeholder:text-text-faint"
          />
        </div>
        <ColorPicker value={color} onChange={setColor} />
        {error && (
          <p role="alert" className="text-status-danger">
            {translateCode(t, error)}
          </p>
        )}
        <div className="flex justify-end gap-2">
          <button type="button" onClick={onClose} className="h-10 rounded-control border-[1.5px] border-border px-4 font-semibold hover:bg-raised">
            {t("common.cancel")}
          </button>
          <button type="submit" className="h-10 rounded-control bg-accent px-5 font-bold text-on-accent">
            {t("common.save")}
          </button>
        </div>
      </form>
    </Dialog>
  );
}

function LogOutDialog({ status, onClose }: { status: AccountStatus; onClose: () => void }) {
  const { t } = useTranslation();
  const titleId = useId();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [done, setDone] = useState(false);

  const logOut = async () => {
    setBusy(true);
    try {
      await removeAccount(status.account.id);
      onClose();
    } catch (err) {
      // The account may be gone already, with a notice to finish on the phone.
      const gone = !useAccounts.getState().accounts.some((s) => s.account.id === status.account.id);
      setDone(gone);
      setBusy(false);
      setError(errorCode(err));
    }
  };

  return (
    <Dialog labelledBy={titleId} onClose={onClose}>
      <div className="flex flex-col gap-4 p-5">
        <h2 id={titleId} className="text-heading font-semibold">
          {t("settings.accounts.logOutTitle", { account: accountLabel(status) })}
        </h2>
        <p className="leading-relaxed text-text-secondary">
          {t("settings.accounts.logOutBody", { network: status.manifest.name })}
        </p>
        {error && (
          <p role="alert" className="text-status-danger">
            {translateCode(t, error)}
          </p>
        )}
        <div className="flex justify-end gap-2">
          <button type="button" onClick={onClose} className="h-10 rounded-control border-[1.5px] border-border px-4 font-semibold hover:bg-raised">
            {done ? t("common.close") : t("common.cancel")}
          </button>
          {!done && (
            <button
              type="button"
              disabled={busy}
              onClick={() => void logOut()}
              className="h-10 rounded-control border-[1.5px] border-status-danger px-5 font-bold text-status-danger disabled:opacity-50"
            >
              {t("settings.accounts.logOut")}
            </button>
          )}
        </div>
      </div>
    </Dialog>
  );
}
