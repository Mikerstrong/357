import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  base: './',
  plugins: [
    vue(),
    // Strip crossorigin attribute from script/link tags — cPanel shared hosting
    // doesn't send CORS headers for same-origin assets, causing module load failures
    {
      name: 'remove-crossorigin',
      transformIndexHtml(html) {
        return html
          .replace(/<script([^>]*) crossorigin([^>]*)>/g, '<script$1$2>')
          .replace(/<link([^>]*) crossorigin([^>]*?)>/g, '<link$1$2>')
      },
    },
  ],
  server: {
    port: 5173,
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
  build: {
    outDir: '../dist',
  },
})
