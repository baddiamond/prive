// Stand-in for the Wails JavaScript runtime (wailsjs/runtime/runtime) in the UI harness.
const listeners = new Map();

/** Subscribes to a backend event; returns the unsubscribe function. */
export function EventsOn(name, callback) {
  const set = listeners.get(name) ?? new Set();
  set.add(callback);
  listeners.set(name, set);
  return () => set.delete(callback);
}

/** Opening links does nothing in the harness. */
export function BrowserOpenURL() {}

/** Delivers an event to the UI, as the Go side would with runtime.EventsEmit. */
export function emit(name, payload) {
  for (const callback of listeners.get(name) ?? []) callback(payload);
}

// Lets the regression script push events: window.__emit("message:new", {...}).
window.__emit = emit;
