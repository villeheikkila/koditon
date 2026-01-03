import { type AuthTokens } from "@/api/auth-manager";
import {
  authSignInAnonymous,
  authSignInWithApple,
} from "@/generated/authentication/authentication";
import { appManager } from "@/state/app-manager";
import { useAuth } from "@/state/auth-context";
import * as AppleAuthentication from "expo-apple-authentication";
import { router, type Href } from "expo-router";
import { useCallback, useEffect, useState } from "react";
import {
  ActivityIndicator,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  View,
} from "react-native";

export default function LoginScreen() {
  const { isAuthenticated, setTokens } = useAuth();
  const [loading, setLoading] = useState(false);
  const [appleAuthAvailable, setAppleAuthAvailable] = useState(false);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    if (Platform.OS === "ios") {
      AppleAuthentication.isAvailableAsync().then(setAppleAuthAvailable);
    }
  }, []);
  const handleAnonymousSignIn = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await authSignInAnonymous({});
      if (response.status !== 200) {
        throw new Error(response.data.detail ?? "Sign in failed");
      }
      const tokens: AuthTokens = {
        accessToken: response.data.access_token,
        accessTokenExpiresAt: response.data.access_token_expires_at,
        refreshToken: response.data.refresh_token,
        refreshTokenExpiresAt: response.data.refresh_token_expires_at,
        userId: response.data.user_id,
      };
      setTokens(tokens);
      await appManager.refreshCities();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Sign in failed");
    } finally {
      setLoading(false);
    }
  }, [setTokens]);
  const handleAppleSignIn = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const credential = await AppleAuthentication.signInAsync({
        requestedScopes: [
          AppleAuthentication.AppleAuthenticationScope.FULL_NAME,
          AppleAuthentication.AppleAuthenticationScope.EMAIL,
        ],
      });
      if (!credential.authorizationCode) {
        throw new Error("No authorization code received from Apple");
      }
      const response = await authSignInWithApple({
        authorization_code: credential.authorizationCode,
      });
      if (response.status !== 200) {
        throw new Error(response.data.detail ?? "Sign in failed");
      }
      const tokens: AuthTokens = {
        accessToken: response.data.access_token,
        accessTokenExpiresAt: response.data.access_token_expires_at,
        refreshToken: response.data.refresh_token,
        refreshTokenExpiresAt: response.data.refresh_token_expires_at,
        userId: response.data.user_id,
      };
      setTokens(tokens);
      await appManager.refreshCities();
    } catch (e: unknown) {
      if (
        e &&
        typeof e === "object" &&
        "code" in e &&
        e.code === "ERR_REQUEST_CANCELED"
      ) {
        return;
      }
      setError(e instanceof Error ? e.message : "Sign in failed");
    } finally {
      setLoading(false);
    }
  }, [setTokens]);
  useEffect(() => {
    if (isAuthenticated) {
      router.replace("/(tabs)" as Href);
    }
  }, [isAuthenticated]);
  if (isAuthenticated) {
    return null;
  }
  return (
    <View style={styles.container}>
      <View style={styles.content}>
        <Text style={styles.title}>Welcome to Koditon</Text>
        <Text style={styles.subtitle}>
          Sign in to access real estate data and insights
        </Text>
        {error && <Text style={styles.error}>{error}</Text>}
        <View style={styles.buttons}>
          {appleAuthAvailable && (
            <AppleAuthentication.AppleAuthenticationButton
              buttonType={
                AppleAuthentication.AppleAuthenticationButtonType.SIGN_IN
              }
              buttonStyle={
                AppleAuthentication.AppleAuthenticationButtonStyle.WHITE
              }
              cornerRadius={12}
              style={styles.appleButton}
              onPress={handleAppleSignIn}
            />
          )}
          <Pressable
            style={[styles.button, styles.anonymousButton]}
            onPress={handleAnonymousSignIn}
            disabled={loading}
          >
            {loading ? (
              <ActivityIndicator color="#fff" />
            ) : (
              <Text style={styles.buttonText}>Continue as Guest</Text>
            )}
          </Pressable>
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: "#000",
  },
  content: {
    flex: 1,
    justifyContent: "center",
    alignItems: "center",
    paddingHorizontal: 24,
  },
  title: {
    fontSize: 28,
    fontWeight: "700",
    color: "#fff",
    marginBottom: 8,
    textAlign: "center",
  },
  subtitle: {
    fontSize: 16,
    color: "#888",
    marginBottom: 48,
    textAlign: "center",
  },
  error: {
    fontSize: 14,
    color: "#ff4444",
    marginBottom: 16,
    textAlign: "center",
  },
  buttons: {
    width: "100%",
    gap: 12,
  },
  button: {
    width: "100%",
    height: 50,
    borderRadius: 12,
    justifyContent: "center",
    alignItems: "center",
  },
  anonymousButton: {
    backgroundColor: "#333",
  },
  appleButton: {
    width: "100%",
    height: 50,
  },
  buttonText: {
    fontSize: 17,
    fontWeight: "600",
    color: "#fff",
  },
});
