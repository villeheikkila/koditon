import { authManager, type AuthTokens } from "@/api/auth-manager";
import { initializeHeadersProvider } from "@/api/headers-provider";
import {
  authGetCurrentUser,
  authRefreshTokens,
} from "@/generated/authentication/authentication";
import { postalCities } from "@/generated/default/default";
import type { PostalCity } from "@/generated/models";
import { createMMKV } from "react-native-mmkv";

const refreshTokens = async (
  refreshToken: string,
): Promise<{ tokens: AuthTokens } | null> => {
  try {
    const response = await authRefreshTokens({ refresh_token: refreshToken });
    if (response.status !== 200) {
      return null;
    }
    return {
      tokens: {
        accessToken: response.data.access_token,
        accessTokenExpiresAt: response.data.access_token_expires_at,
        refreshToken: response.data.refresh_token,
        refreshTokenExpiresAt: response.data.refresh_token_expires_at,
        userId: response.data.user_id,
      },
    };
  } catch {
    return null;
  }
};

type AppState = {
  ready: boolean;
  cities: PostalCity[];
  loadingCities: boolean;
  error: string | null;
  featureFlags: string[];
};

const emptyFeatureFlags: string[] = [];

type Listener = () => void;

const storageKey = "app:cities:v1";
const emptyCities: PostalCity[] = [];
const storage = createMMKV({ id: "app" });
const readStoredObject = <T>(key: string, fallback: T) => {
  const raw = storage.getString(key);
  if (!raw) {
    return fallback;
  }
  try {
    const parsed = JSON.parse(raw) as T;
    return parsed ?? fallback;
  } catch {
    return fallback;
  }
};
const writeStoredObject = <T>(key: string, value: T) => {
  try {
    storage.set(key, JSON.stringify(value));
  } catch {
    return;
  }
};

class AppManager {
  private state: AppState;
  private listeners: Set<Listener>;
  private loadPromise: Promise<void> | null;
  constructor() {
    this.state = {
      ready: false,
      cities: emptyCities,
      loadingCities: false,
      error: null,
      featureFlags: emptyFeatureFlags,
    };
    this.listeners = new Set();
    this.loadPromise = null;
  }
  getState() {
    return this.state;
  }
  subscribe(listener: Listener) {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }
  async load() {
    if (this.loadPromise) {
      return this.loadPromise;
    }
    this.loadPromise = this.loadInternal();
    return this.loadPromise;
  }
  async refreshCities() {
    await this.loadCities();
  }
  private async loadInternal() {
    initializeHeadersProvider();
    authManager.setRefreshFn(refreshTokens);
    this.setState({ loadingCities: true, error: null });
    const cached = await this.readCachedCities();
    if (cached.length > 0) {
      this.setState({ cities: cached });
    }
    if (!authManager.isAuthenticated()) {
      this.setState({ loadingCities: false, ready: true });
      return;
    }
    await this.loadCurrentUser();
    try {
      const response = await postalCities();
      if (response.status !== 200) {
        throw new Error("Failed to load cities");
      }
      const cities = response.data.cities ?? emptyCities;
      this.setState({ cities, error: null });
      await this.writeCachedCities(cities);
    } catch {
      if (this.state.cities.length === 0) {
        this.setState({ error: "Failed to load cities" });
      }
    } finally {
      this.setState({ loadingCities: false, ready: true });
    }
  }
  private async loadCurrentUser() {
    try {
      const response = await authGetCurrentUser();
      if (response.status === 200) {
        this.setState({
          featureFlags: response.data.feature_flags ?? emptyFeatureFlags,
        });
      }
    } catch {
      // Feature flags are non-critical, continue without them
    }
  }
  private async loadCities() {
    this.setState({ loadingCities: true, error: null });
    try {
      const response = await postalCities();
      if (response.status !== 200) {
        throw new Error("Failed to load cities");
      }
      const cities = response.data.cities ?? emptyCities;
      this.setState({ cities, error: null });
      await this.writeCachedCities(cities);
    } catch {
      this.setState({ error: "Failed to load cities" });
    } finally {
      this.setState({ loadingCities: false });
    }
  }
  private async readCachedCities() {
    const cached = readStoredObject<PostalCity[]>(storageKey, emptyCities);
    return Array.isArray(cached) ? cached : emptyCities;
  }
  private async writeCachedCities(cities: PostalCity[]) {
    writeStoredObject(storageKey, cities);
  }
  private setState(next: Partial<AppState>) {
    this.state = { ...this.state, ...next };
    this.emit();
  }
  private emit() {
    this.listeners.forEach((listener) => listener());
  }
}

export const appManager = new AppManager();
