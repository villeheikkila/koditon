import { authManager, type AuthTokens } from "@/api/auth-manager";
import {
  createContext,
  useCallback,
  useContext,
  useSyncExternalStore,
  type ReactNode,
} from "react";

type AuthState = {
  isAuthenticated: boolean;
  tokens: AuthTokens | null;
  signOut: () => void;
  setTokens: (tokens: AuthTokens) => void;
};

const AuthContext = createContext<AuthState | null>(null);

type AuthProviderProps = {
  children: ReactNode;
};

const computeIsAuthenticated = (tokens: AuthTokens | null): boolean => {
  if (!tokens) {
    return false;
  }
  const now = Date.now();
  const refreshExpiresAt = tokens.refreshTokenExpiresAt * 1000;
  return now < refreshExpiresAt;
};

export const AuthProvider = ({ children }: AuthProviderProps) => {
  const tokens = useSyncExternalStore(
    (callback) => authManager.subscribe(callback),
    () => authManager.getTokens(),
    () => authManager.getTokens(),
  );
  const isAuthenticated = computeIsAuthenticated(tokens);
  const signOut = useCallback(() => {
    authManager.signOut();
  }, []);
  const setTokens = useCallback((newTokens: AuthTokens) => {
    authManager.setTokens(newTokens);
  }, []);

  return (
    <AuthContext.Provider
      value={{ isAuthenticated, tokens, signOut, setTokens }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const value = useContext(AuthContext);
  if (!value) {
    throw new Error("AuthProvider is missing");
  }
  return value;
};
