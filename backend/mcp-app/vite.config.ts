import { defineConfig } from "vite";
import { viteSingleFile } from "vite-plugin-singlefile";

export default defineConfig({
  root: "src",
  build: {
    outDir: "../../internal/transport/mcp/appdist",
    emptyOutDir: true,
    rollupOptions: {
      input: "mcp-app.html"
    }
  },
  plugins: [viteSingleFile()]
});
