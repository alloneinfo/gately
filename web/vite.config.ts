import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

// 构建后保留 dist/.gitkeep，保证未构建前端时 Go 的 go:embed 仍可编译
const keepGitkeep = {
  name: 'keep-gitkeep',
  closeBundle() {
    writeFileSync(resolve(import.meta.dirname, 'dist/.gitkeep'), '')
  },
}

export default defineConfig({
  plugins: [vue(), keepGitkeep],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    chunkSizeWarningLimit: 1500,
  },
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:16680',
    },
  },
})
