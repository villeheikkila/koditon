import { createMMKV } from "react-native-mmkv";

export type AuthHeader = {
  name: string;
  value: string;
};

export type AuthTokens = {
  accessToken: string;
  accessTokenExpiresAt: number;
  refreshToken: string;
  refreshTokenExpiresAt: number;
  userId: string;
};

const storage = createMMKV({ id: "auth" });
const TOKENS_KEY = "auth:tokens:v1";
const ACCESS_TOKEN_BUFFER_MS = 60 * 1000;

const readTokens = (): AuthTokens | null => {
  const raw = storage.getString(TOKENS_KEY);
  if (!raw) {
    return null;
  }
  try {
    return JSON.parse(raw) as AuthTokens;
  } catch {
    return null;
  }
};

const writeTokens = (tokens: AuthTokens | null) => {
  if (!tokens) {
    storage.remove(TOKENS_KEY);
    return;
  }
  storage.set(TOKENS_KEY, JSON.stringify(tokens));
};

type RefreshFn = (
  refreshToken: string,
) => Promise<{ tokens: AuthTokens } | null>;

class AuthManager {
  private tokens: AuthTokens | null = null;
  private refreshPromise: Promise<AuthTokens | null> | null = null;
  private refreshFn: RefreshFn | null = null;
  private listeners: Set<() => void> = new Set();
  constructor() {
    this.tokens = readTokens();
  }
  setRefreshFn(fn: RefreshFn) {
    this.refreshFn = fn;
  }
  getTokens(): AuthTokens | null {
    return this.tokens;
  }
  setTokens(tokens: AuthTokens | null) {
    this.tokens = tokens;
    writeTokens(tokens);
    this.emit();
  }
  isAuthenticated(): boolean {
    if (!this.tokens) {
      return false;
    }
    const now = Date.now();
    const refreshExpiresAt = this.tokens.refreshTokenExpiresAt * 1000;
    return now < refreshExpiresAt;
  }
  subscribe(listener: () => void): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }
  async getAuthHeader(): Promise<AuthHeader | null> {
    const tokens = await this.getValidAccessToken();
    if (!tokens) {
      return null;
    }
    return { name: "Authorization", value: `Bearer ${tokens.accessToken}` };
  }
  async getValidAccessToken(): Promise<AuthTokens | null> {
    if (!this.tokens) {
      return null;
    }
    const now = Date.now();
    const accessExpiresAt = this.tokens.accessTokenExpiresAt * 1000;
    if (now < accessExpiresAt - ACCESS_TOKEN_BUFFER_MS) {
      return this.tokens;
    }
    return this.refreshTokens();
  }
  async refreshTokens(): Promise<AuthTokens | null> {
    if (this.refreshPromise) {
      return this.refreshPromise;
    }
    this.refreshPromise = this.doRefresh();
    try {
      return await this.refreshPromise;
    } finally {
      this.refreshPromise = null;
    }
  }
  private async doRefresh(): Promise<AuthTokens | null> {
    if (!this.tokens || !this.refreshFn) {
      return null;
    }
    const now = Date.now();
    const refreshExpiresAt = this.tokens.refreshTokenExpiresAt * 1000;
    if (now >= refreshExpiresAt) {
      this.setTokens(null);
      return null;
    }
    try {
      const result = await this.refreshFn(this.tokens.refreshToken);
      if (!result) {
        this.setTokens(null);
        return null;
      }
      this.setTokens(result.tokens);
      return result.tokens;
    } catch {
      this.setTokens(null);
      return null;
    }
  }
  signOut() {
    this.setTokens(null);
  }
  private emit() {
    this.listeners.forEach((listener) => listener());
  }
}

export const authManager = new AuthManager();
