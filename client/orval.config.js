import { defineConfig } from "orval";

export default defineConfig({
  api: {
    output: {
      mode: "tags-split",
      target: "src/generated/api.ts",
      schemas: "src/generated/model",
      client: "swr",
    },
    input: "http://localhost:8080/openapi.json",
  },
});
