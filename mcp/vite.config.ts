import { defineConfig } from "vite";
import { viteSingleFile } from "vite-plugin-singlefile";

export default defineConfig({
  root: "src",
  build: {
    outDir: "../dist",
    emptyOutDir: false,
    rollupOptions: {
      input: "src/mcp-app.html"
    }
  },
  plugins: [viteSingleFile()]
});
