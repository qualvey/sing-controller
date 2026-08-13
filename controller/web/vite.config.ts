import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { lezer } from '@lezer/generator/rollup'
import tailwindcss from '@tailwindcss/vite'
import { resolve } from 'node:path'

// 子路径反代部署：构建时用 VITE_BASE_URL 指定挂载路径（必须以 / 开头、/ 结尾），
// 产物资源与路由 base 均以此为前缀；默认 '/' 保持根路径部署不变。
// 例：VITE_BASE_URL=/webui/ pnpm run build
// 注意：history 路由的 base 必须是绝对路径，不能用 './'（否则深链刷新失效）。
function resolveBase(): string {
  const raw = (process.env.VITE_BASE_URL || '/').trim()
  const withLeading = raw.startsWith('/') ? raw : `/${raw}`
  return withLeading.endsWith('/') ? withLeading : `${withLeading}/`
}

export default defineConfig({
  // 资源路径 base（同时注入 import.meta.env.BASE_URL，供路由/API 使用）
  base: resolveBase(),
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
