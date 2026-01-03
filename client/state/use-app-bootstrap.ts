import * as SplashScreen from "expo-splash-screen";
import { useEffect, useState } from "react";
import { appManager } from "./app-manager";

void SplashScreen.preventAutoHideAsync();

export const useAppBootstrap = () => {
  const [ready, setReady] = useState(appManager.getState().ready);
  useEffect(() => {
    let active = true;
    const bootstrap = async () => {
      try {
        await appManager.load();
      } catch {
        return;
      } finally {
        if (!active) {
          return;
        }
        setReady(true);
        await SplashScreen.hideAsync();
      }
    };
    void bootstrap();
    return () => {
      active = false;
    };
  }, []);
  return ready;
};
