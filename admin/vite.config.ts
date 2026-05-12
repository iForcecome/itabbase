import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// Dev backend (Go server with pathPrefix). To target a different scaffold,
// edit this constant. Default matches itab-xxxxxx (config.dev.yaml:
// server.pathPrefix).
const DEV_API_BASE = 'http://localhost:8080/itab/ai-base-demo'

export default defineConfig({
  plugins: [vue()],
  // Relative asset paths so the embedded SPA works under any pathPrefix.
  base: './',
  build: {
    // Output into the Go server's embed root; admin_embed.go has
    // `//go:embed all:web` (Go embed cannot reach `..`).
    outDir: '../server/web',
    emptyOutDir: true,
    target: 'es2022',
    sourcemap: false,
  },
  server: {
    // 8079 to coexist with scaffold's client/ on 8081 (strictPort there).
    // Auth login uses ?return=window.location.href, so any localhost:* works;
    // cookies (host-only on localhost) are shared across ports.
    port: 8079,
    strictPort: true,
    proxy: {
      // SPA uses relative URLs `../api/...`; from vite root they resolve
      // to /api/..., proxied to the kernel under the dev pathPrefix.
      '/api': { target: DEV_API_BASE, changeOrigin: true },
    },
  },
})
