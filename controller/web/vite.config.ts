import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { lezer } from '@lezer/generator/rollup'
import tailwindcss from '@tailwindcss/vite'
import { resolve } from 'node:path'
import vueJsx from '@vitejs/plugin-vue-jsx' // 1. 引入插件
import { version } from './package.json' with { type: 'json' }
import { execSync } from 'child_process'


// 子路径反代部署：构建时用 VITE_BASE_URL 指定挂载路径（必须以 / 开头、/ 结尾），
// 产物资源与路由 base 均以此为前缀；默认 '/' 保持根路径部署不变。
// 例：VITE_BASE_URL=/webui/ pnpm run build
// 注意：history 路由的 base 必须是绝对路径，不能用 './'（否则深链刷新失效）。
function resolveBase(): string {
  const raw = (process.env.VITE_BASE_URL || '/').trim()
  const withLeading = raw.startsWith('/') ? raw : `/${raw}`
  return withLeading.endsWith('/') ? withLeading : `${withLeading}/`
}
const font = process.env.FONT || 'all'

const getGitCommitId = (): string => {
  try {
    const commitMessage = execSync('git log -1 --pretty=%B', { encoding: 'utf8' }).trim()

    if (commitMessage.includes('chore(main): release')) {
      return ''
    }

    return execSync('git rev-parse --short HEAD', { encoding: 'utf8' }).trim()
  } catch (error) {
    console.warn('无法获取git commit ID:', error)
    return ''
  }
}
export default defineConfig({
  // 资源路径 base（同时注入 import.meta.env.BASE_URL，供路由/API 使用）
  define: {
    __APP_VERSION__: JSON.stringify(version),
    __COMMIT_ID__: JSON.stringify(getGitCommitId()),
    __FONT__: JSON.stringify(font),
  },
  esbuild: {
    // 生产环境打包时自动丢弃所有 debugger 和 console.log
    drop: process.env.NODE_ENV === 'production' ? ['debugger', 'console'] : []
  },
  base: resolveBase(),
  plugins: [vue(), lezer(), tailwindcss(), vueJsx()],
  resolve: {
    alias: {
      // 强制 @codemirror/state 单文件：rollup 可能因 ESM/CJS 双入口把它解析成
      // 两个模块（同 chunk 双实例 → Unrecognized extension value），这里统一到 ESM 入口
      '@codemirror/state': resolve(import.meta.dirname, 'node_modules/@codemirror/state/dist/index.js'),
      '@': resolve(import.meta.dirname, 'src')
    }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
        ws: true
      }
    }
  }
})
