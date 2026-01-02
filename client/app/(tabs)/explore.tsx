import { useEffect, useState } from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";

import { healthz } from "@/api/default/default";
import { ThemedText } from "@/components/themed-text";
import { ThemedView } from "@/components/themed-view";
import { Fonts } from "@/constants/theme";

export default function TabTwoScreen() {
  const [status, setStatus] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
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
    fetchHealth();
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
        {status && <ThemedText>{status}</ThemedText>}
        {!status && !error && <ActivityIndicator />}
        {error && <ThemedText>{error}</ThemedText>}
      </ThemedView>
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
});
