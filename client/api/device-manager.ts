import * as Crypto from "expo-crypto";
import { createMMKV } from "react-native-mmkv";

const storage = createMMKV({ id: "device" });
const DEVICE_ID_KEY = "device:id:v1";

class DeviceManager {
  private deviceId: string;
  constructor() {
    const stored = storage.getString(DEVICE_ID_KEY);
    if (stored) {
      this.deviceId = stored;
    } else {
      this.deviceId = Crypto.randomUUID();
      storage.set(DEVICE_ID_KEY, this.deviceId);
    }
  }
  getDeviceId(): string {
    return this.deviceId;
  }
}

export const deviceManager = new DeviceManager();
