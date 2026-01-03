import Constants from "expo-constants";
import * as Crypto from "expo-crypto";
import { Platform } from "react-native";
import { authManager } from "./auth-manager";
import { deviceManager } from "./device-manager";
import { setHeadersProvider } from "./orval-mutator";

const buildUserAgent = (): string => {
  const appName = Constants.expoConfig?.name ?? "Koditon";
  const appVersion = Constants.expoConfig?.version ?? "1.0.0";
  const os = Platform.OS;
  const osVersion = Platform.Version;
  return `${appName}/${appVersion} (${os}; ${osVersion})`;
};

export const initializeHeadersProvider = () => {
  const userAgent = buildUserAgent();
  const deviceId = deviceManager.getDeviceId();
  setHeadersProvider(async () => {
    const headers: Record<string, string> = {
      "X-Request-ID": Crypto.randomUUID(),
      "X-Device-ID": deviceId,
      "User-Agent": userAgent,
    };
    const authHeader = await authManager.getAuthHeader();
    if (authHeader) {
      headers[authHeader.name] = authHeader.value;
    }
    return headers;
  });
};
