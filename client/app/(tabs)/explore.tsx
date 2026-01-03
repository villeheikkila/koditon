import { useEffect, useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View } from "react-native";

import { ThemedText } from "@/components/themed-text";
import { ThemedView } from "@/components/themed-view";
import { Fonts } from "@/constants/theme";
import { getHealthzUrl, healthz, ping } from "@/generated/default/default";
import { useAuth } from "@/state/auth-context";

export default function TabTwoScreen() {
  const { signOut } = useAuth();
  const [status, setStatus] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pingEcho, setPingEcho] = useState<string | null>(null);
  const [pingError, setPingError] = useState<string | null>(null);
  useEffect(() => {
    let mounted = true;
    const fetchHealth = async () => {
      try {
        const res = await healthz();
        if (!mounted) return;
        if (res.status === 200) {
          setStatus(res.data.status);
          setError(null);
          return;
        }
        setError(res.data?.status?.toString() ?? "Health check failed");
      } catch (err) {
        if (!mounted) return;
        setError(err instanceof Error ? err.message : "Health check failed");
      }
    };
    const fetchPing = async () => {
      try {
        const res = await ping({ message: "ping" });
        if (!mounted) return;
        if (res.status === 200) {
          setPingEcho(res.data.echo);
          setPingError(null);
          return;
        }
        setPingError(res.data?.status?.toString() ?? "Ping failed");
      } catch (err) {
        if (!mounted) return;
        setPingError(err instanceof Error ? err.message : "Ping failed");
      }
    };
    fetchHealth();
    fetchPing();
    return () => {
      mounted = false;
    };
  }, []);
  return (
    <View>
      <ThemedView style={styles.titleContainer}>
        <ThemedText
          type="title"
          style={{
            fontFamily: Fonts.rounded,
          }}
        >
          Explore
        </ThemedText>
      </ThemedView>
      <ThemedText>
        This app includes example code to help you get started.
      </ThemedText>
      <ThemedView style={styles.healthRow}>
        <ThemedText type="defaultSemiBold">Health</ThemedText>
        <ThemedText type="defaultSemiBold">{getHealthzUrl()}</ThemedText>
        {status && <ThemedText>{status}</ThemedText>}
        {!status && !error && <ActivityIndicator />}
        {error && <ThemedText>{error}</ThemedText>}
      </ThemedView>
      <ThemedView style={styles.healthRow}>
        <ThemedText type="defaultSemiBold">Ping</ThemedText>
        {pingEcho && <ThemedText>{pingEcho}</ThemedText>}
        {!pingEcho && !pingError && <ActivityIndicator />}
        {pingError && <ThemedText>{pingError}</ThemedText>}
      </ThemedView>
      <Pressable style={styles.signOutButton} onPress={signOut}>
        <ThemedText style={styles.signOutText}>Sign Out</ThemedText>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  headerImage: {
    color: "#808080",
    bottom: -90,
    left: -35,
    position: "absolute",
  },
  titleContainer: {
    flexDirection: "row",
    gap: 8,
  },
  healthRow: {
    alignItems: "center",
    flexDirection: "row",
    gap: 8,
    marginTop: 12,
  },
  signOutButton: {
    marginTop: 32,
    backgroundColor: "#333",
    paddingVertical: 12,
    paddingHorizontal: 24,
    borderRadius: 8,
    alignSelf: "flex-start",
  },
  signOutText: {
    color: "#fff",
    fontWeight: "600",
  },
});
