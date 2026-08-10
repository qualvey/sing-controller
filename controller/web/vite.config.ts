import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { lezer } from '@lezer/generator/rollup'
import tailwindcss from '@tailwindcss/vite'
import { resolve } from 'node:path'

export default defineConfig({
  plugins: [vue(), lezer(), tailwindcss()],
  resolve: {
    alias: {
      // 强制 @codemirror/state 单文件：rollup 可能因 ESM/CJS 双入口把它解析成
      // 两个模块（同 chunk 双实例 → Unrecognized extension value），这里统一到 ESM 入口
      '@codemirror/state': resolve(__dirname, 'node_modules/@codemirror/state/dist/index.js'),
      '@': resolve(__dirname, 'src')
    }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true
      }
    }
  }
})
