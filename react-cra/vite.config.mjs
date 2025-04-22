import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

export default defineConfig(({ mode }) => {
  return {
    plugins: [
      react(),
    ],
    server: mode === 'development' ? {
      port: 3000,
      host: "0.0.0.0",
      proxy: {
        '/api': {
          target: 'http://localhost:8080/api',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, '')
        }
      }
    } : undefined,
    build: {
      outDir: "build",
    },
  }
});