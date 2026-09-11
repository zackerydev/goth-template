import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [react()],
  base: "/app/",
  build: {
    outDir: resolve(root, "../assets/app"),
    emptyOutDir: true,
    sourcemap: false,
    minify: true,
  },
});
