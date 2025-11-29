import { defineConfig } from "orval";

export default defineConfig({
  api: {
    output: {
      mode: "tags-split",
      target: "src/generated/petstore.ts",
      schemas: "src/generated/models",
      client: "fetch",
      baseUrl: "http://localhost:8080",
    },
    input: "http://localhost:8080/openapi.json",
  },
});
