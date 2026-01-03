import { Colors } from "@/constants/theme";
import { useColorScheme } from "@/hooks/use-color-scheme";
import { useAuth } from "@/state/auth-context";
import { router, type Href } from "expo-router";
import { Icon, Label, NativeTabs } from "expo-router/unstable-native-tabs";
import { useEffect } from "react";

export default function TabLayout() {
  const colorScheme = useColorScheme();
  const { isAuthenticated } = useAuth();
  useEffect(() => {
    if (!isAuthenticated) {
      router.replace("/login" as Href);
    }
  }, [isAuthenticated]);
  if (!isAuthenticated) {
    return null;
  }
  return (
    <NativeTabs tintColor={Colors[colorScheme ?? "light"].tint}>
      <NativeTabs.Trigger name="index">
        <Label>Home</Label>
        <Icon sf="house.fill" />
      </NativeTabs.Trigger>
      <NativeTabs.Trigger name="explore">
        <Label>Explore</Label>
        <Icon sf="paperplane.fill" />
      </NativeTabs.Trigger>
    </NativeTabs>
  );
}
