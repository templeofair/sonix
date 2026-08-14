/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  // Keep cache writable even when node_modules/.vite-temp is root-owned (e.g. Docker installs).
  cacheDir: '.vite',
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    exclude: ['**/node_modules/**', 'src/lib/markdownNormalize.test.ts'],
  },
  server: {
    proxy: {
      '/api': { target: 'http://localhost:9080', changeOrigin: true },
      '/health': { target: 'http://localhost:9080', changeOrigin: true },
    },
  },
})
