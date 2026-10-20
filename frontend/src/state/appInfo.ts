import { useEffect } from "react";
import { create } from "zustand";
import { getAppInfo, type AppInfo } from "../lib/api";

const useStore = create<{ info: AppInfo | null }>()(() => ({ info: null }));

let requested = false;

/** Returns the app version and data folder, loading them once on first use. */
export function useAppInfo(): AppInfo | null {
  const info = useStore((s) => s.info);
  useEffect(() => {
    if (requested) return;
    requested = true;
    getAppInfo()
      .then((loaded) => useStore.setState({ info: loaded }))
      .catch(() => {
        requested = false; // try again next time the section opens
      });
  }, []);
  return info;
}
