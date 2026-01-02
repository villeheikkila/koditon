import { Colors } from "@/constants/theme";
import { useColorScheme } from "@/hooks/use-color-scheme";
import { Icon, Label, NativeTabs } from "expo-router/unstable-native-tabs";
import React from "react";

export default function TabLayout() {
  const colorScheme = useColorScheme();
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
