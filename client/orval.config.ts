import { defineConfig } from "orval";

const apiBaseUrl = process.env.EXPO_PUBLIC_API_URL || "http://localhost:8080";

export default defineConfig({
  koditon: {
    input: "http://localhost:8080/openapi.yaml",
    output: {
      mode: "tags-split",
      target: "api/client.ts",
      schemas: "api/models",
      client: "fetch",
      httpClient: "fetch",
      baseUrl: apiBaseUrl,
      clean: true,
    },
  },
});
