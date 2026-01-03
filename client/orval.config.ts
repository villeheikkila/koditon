import { defineConfig } from "orval";

// Default to localhost, use ORVAL_PROD=1 to generate from production API
const API_BASE = process.env.ORVAL_PROD
  ? "https://api.bytesized.solutions"
  : "http://localhost:8080";

export default defineConfig({
  koditon: {
    input: `${API_BASE}/openapi.yaml`,
    output: {
      mode: "tags-split",
      target: "generated/client.ts",
      schemas: "generated/models",
      client: "fetch",
      httpClient: "fetch",
      baseUrl: "http://localhost:8080",
      clean: true,
      override: {
        mutator: {
          path: "api/orval-mutator.ts",
          name: "authFetch",
        },
      },
    },
  },
});
