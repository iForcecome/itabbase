import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import tailwindcss from "@tailwindcss/vite";
import Components from "unplugin-vue-components/vite";
import { AntDesignVueResolver } from "unplugin-vue-components/resolvers";
import AutoImport from "unplugin-auto-import/vite";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));

const DEV_API_BASE = "http://localhost:8080";

export default defineConfig({
  base: "./",
  plugins: [
    vue(),
    tailwindcss(),
    Components({
      resolvers: [AntDesignVueResolver({ importStyle: false, resolveIcons: true })],
      dts: "src/types/components.d.ts",
    }),
    AutoImport({
      imports: ["vue", "vue-router", "pinia", "@vueuse/core"],
      dts: "src/types/auto-imports.d.ts",
      eslintrc: { enabled: true },
    }),
  ],
  resolve: { alias: { "@": resolve(__dirname, "src") } },
  build: {
    outDir: "../server/web",
    emptyOutDir: true,
    target: "es2022",
    sourcemap: false,
    rollupOptions: {
      output: {
        manualChunks: {
          vue: ["vue", "vue-router", "pinia"],
        },
      },
    },
  },
  server: {
    port: 8079,
    strictPort: true,
    proxy: {
      "/api": { target: DEV_API_BASE, changeOrigin: true },
    },
  },
});
