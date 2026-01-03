import { useSyncExternalStore } from "react";
import { appManager } from "./app-manager";

export const useAppState = () =>
  useSyncExternalStore(
    (listener) => appManager.subscribe(listener),
    () => appManager.getState(),
    () => appManager.getState(),
  );

export const useHasFeatureFlag = (flag: string) => {
  const state = useAppState();
  return state.featureFlags.includes(flag);
};
