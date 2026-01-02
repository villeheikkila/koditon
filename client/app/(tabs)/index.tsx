import { ThemedText } from "@/components/themed-text";
import { ThemedView } from "@/components/themed-view";
import { Button, Host, Text } from "@expo/ui/swift-ui";
import { StyleSheet, View } from "react-native";

export default function HomeScreen() {
  return (
    <View>
      <ThemedView style={styles.titleContainer}>
        <ThemedText type="title">Home</ThemedText>
      </ThemedView>
      <ThemedView style={styles.section}>
        <ThemedText type="subtitle">SwiftUI Button Preview</ThemedText>
        <Host matchContents>
          <Button variant="default" systemImage="sparkles" onPress={() => {}}>
            <Text>Primary Action</Text>
          </Button>
        </Host>
      </ThemedView>
      <ThemedView style={styles.section}>
        <ThemedText type="subtitle">Secondary</ThemedText>
        <Host matchContents>
          <Button
            variant="bordered"
            systemImage="chevron.right"
            onPress={() => {}}
          >
            <Text>See Details</Text>
          </Button>
        </Host>
      </ThemedView>
    </View>
  );
}

const styles = StyleSheet.create({
  titleContainer: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
  },
  section: {
    gap: 8,
    marginBottom: 16,
  },
});
