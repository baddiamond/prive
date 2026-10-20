// Typed subscriptions to the backend's Wails events (docs/ARCHITECTURE.md, "Events").
import { EventsOn } from "../../wailsjs/runtime/runtime";
import type { AccountState } from "./accounts";
import type { ChatPayload } from "./chats";
import type { MessagePayload } from "./messages";

/** Payloads of the events the UI listens to. Every one carries its accountId. */
export interface BackendEvents {
  "account:state": { accountId: string; state: AccountState; code?: string };
  "account:updated": {
    accountId: string;
    account: { id: string; engineId: string; remoteId: string; displayName: string; color: string; position: number };
  };
  "account:removed": { accountId: string };
  "login:qr": { accountId: string; code: string; expiresAt: string };
  "login:success": { accountId: string; remoteId: string; displayName: string };
  "login:failed": { accountId: string; code: string };
  "chat:updated": { accountId: string; chat: ChatPayload };
  "chats:synced": { accountId: string; chats: number; messages: number };
  "sync:progress": { accountId: string; done: number; total: number };
  "message:new": { accountId: string; message: MessagePayload };
  "history:result": { accountId: string; chatId: string; messages: number; complete: boolean; code?: string };
}

/** Subscribes to one backend event; returns the unsubscribe function. */
export function onBackendEvent<K extends keyof BackendEvents>(
  name: K,
  handler: (payload: BackendEvents[K]) => void,
): () => void {
  return EventsOn(name, (payload: BackendEvents[K]) => handler(payload));
}
