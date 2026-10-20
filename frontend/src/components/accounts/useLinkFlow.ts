import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { addAccount, cancelLogin, login, type AccountColor } from "../../lib/accounts";
import { errorCode } from "../../lib/errors";
import { onBackendEvent } from "../../lib/events";
import { useAccounts } from "../../state/accounts";

/** Where linking stands. */
export type LinkPhase = "starting" | "waiting" | "linked" | "failed";

/** State of one linking attempt, shared by onboarding and the add-account dialog. */
export interface LinkFlow {
  phase: LinkPhase;
  /** Account being linked, once created. */
  accountId: string | null;
  /** Latest QR payload to display. */
  qr: string | null;
  /** Translation key of the failure, when phase is "failed". */
  failure: string | null;
  /** Asks for new QR codes after a failure. */
  retry: () => void;
}

/**
 * Creates an account for engineId (or reuses relinkAccountId) and starts QR login.
 * Leaving before the account is linked removes the half-created account.
 */
export function useLinkFlow(engineId: string, color: AccountColor, relinkAccountId?: string): LinkFlow {
  const [phase, setPhase] = useState<LinkPhase>("starting");
  const [accountId, setAccountId] = useState<string | null>(relinkAccountId ?? null);
  const [qr, setQR] = useState<string | null>(null);
  const [failure, setFailure] = useState<string | null>(null);
  const idRef = useRef<string | null>(relinkAccountId ?? null);
  const linkedRef = useRef(false);
  const initialColor = useRef(color);

  const start = useCallback(async (id: string) => {
    setPhase("starting");
    setFailure(null);
    setQR(null);
    try {
      await login(id, "qr");
      setPhase((p) => (p === "starting" ? "waiting" : p));
    } catch (err) {
      setFailure(errorCode(err));
      setPhase("failed");
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    const offs = [
      onBackendEvent("login:qr", (e) => {
        if (e.accountId === idRef.current) {
          setQR(e.code);
          setPhase("waiting");
        }
      }),
      onBackendEvent("login:success", (e) => {
        if (e.accountId === idRef.current) {
          linkedRef.current = true;
          setPhase("linked");
          void useAccounts.getState().refresh();
        }
      }),
      onBackendEvent("login:failed", (e) => {
        if (e.accountId === idRef.current) {
          setFailure(e.code);
          setPhase("failed");
        }
      }),
    ];

    const begin = async () => {
      let id = relinkAccountId;
      if (!id) {
        try {
          // The name is set after linking; until then the network's own name fills in.
          id = (await addAccount(engineId, "", initialColor.current)).id;
        } catch (err) {
          setFailure(errorCode(err));
          setPhase("failed");
          return;
        }
        if (cancelled) {
          void cancelLogin(id);
          return;
        }
      }
      idRef.current = id;
      setAccountId(id);
      await start(id);
    };
    void begin();

    return () => {
      cancelled = true;
      offs.forEach((off) => off());
      if (!linkedRef.current && !relinkAccountId && idRef.current) {
        void cancelLogin(idRef.current);
      }
    };
  }, [engineId, relinkAccountId, start]);

  const retry = useCallback(() => {
    if (idRef.current) void start(idRef.current);
  }, [start]);

  return useMemo(() => ({ phase, accountId, qr, failure, retry }), [phase, accountId, qr, failure, retry]);
}
