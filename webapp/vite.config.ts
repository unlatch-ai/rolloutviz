import react from "@vitejs/plugin-react";
import { configDefaults, defineConfig } from "vitest/config";

const dependency = (name: string) => new URL(`./node_modules/${name}`, import.meta.url).pathname;

export default defineConfig({
  plugins: [react()],
  resolve: {
    dedupe: ["react", "react-dom"],
    alias: {
      react: dependency("react"),
      "react-dom": dependency("react-dom"),
    },
  },
  server: { fs: { allow: [".."] } },
  build: { outDir: "dist", emptyOutDir: true, minify: "terser" },
  test: {
    environment: "jsdom",
    setupFiles: "./src/test-setup.ts",
    css: true,
    exclude: [...configDefaults.exclude, "e2e/**"],
  },
});
