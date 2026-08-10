import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { lezer } from '@lezer/generator/rollup'

export default defineConfig({
  plugins: [vue(), lezer()],
  build: {
    rollupOptions: {
      output: {
        // CodeMirror 6 系提取为共享 chunk：多个动态 chunk 各自打包会
        // 产生多份 @codemirror/state，instanceof 检查失败（Unrecognized extension value）
        manualChunks: {
          codemirror: [
            'codemirror',
            '@codemirror/state',
            '@codemirror/view',
            '@codemirror/language',
            '@codemirror/commands',
            '@codemirror/lint',
            '@codemirror/search',
            '@codemirror/theme-one-dark',
            '@lezer/common',
            '@lezer/highlight',
            '@lezer/lr',
            'jsonc-parser'
          ]
        }
      }
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
