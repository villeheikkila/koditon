import { ExpoConfig } from "expo/config";

const owner = process.env.EXPO_OWNER;
if (!owner) {
  throw new Error("Missing required environment variable: EXPO_OWNER");
}
const bundleIdentifier = process.env.BUNDLE_IDENTIFIER;
if (!bundleIdentifier) {
  throw new Error("Missing required environment variable: BUNDLE_IDENTIFIER");
}

const projectId = process.env.EAS_PROJECT_ID;
if (!bundleIdentifier) {
  throw new Error("Missing required environment variable: EAS_PROJECT_ID");
}

const config: ExpoConfig = {
  name: "Koditon",
  slug: "koditon",
  platforms: ["ios"],
  version: "1.0.0",
  orientation: "portrait",
  icon: "./assets/images/icon.png",
  scheme: "koditon",
  userInterfaceStyle: "automatic",
  newArchEnabled: true,
  ios: {
    supportsTablet: true,
    bundleIdentifier,
    usesAppleSignIn: true,
    infoPlist: {
      ITSAppUsesNonExemptEncryption: false,
    },
  },
  plugins: [
    "expo-router",
    "expo-apple-authentication",
    [
      "expo-splash-screen",
      {
        image: "./assets/images/splash-icon.png",
        imageWidth: 200,
        resizeMode: "contain",
        backgroundColor: "#ffffff",
        dark: {
          backgroundColor: "#000000",
        },
      },
    ],
  ],
  experiments: {
    typedRoutes: true,
    reactCompiler: true,
  },
  extra: {
    router: {},
    eas: {
      projectId: projectId,
    },
  },
  owner,
  updates: {
    url: `https://u.expo.dev/${projectId}`,
  },
  runtimeVersion: {
    policy: "appVersion",
  },
};

export default config;
