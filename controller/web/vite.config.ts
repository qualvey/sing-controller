import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import { lezer } from '@lezer/generator/rollup'
import tailwindcss from '@tailwindcss/vite'
import { resolve } from 'node:path'
import vueJsx from '@vitejs/plugin-vue-jsx' // 1. 引入插件
import { version } from './package.json' with { type: 'json' }
import { execSync } from 'child_process'

function buildInfoPlugin(): Plugin {
  const virtualModuleId = 'virtual:recent-changes'
  const resolvedVirtualModuleId = '\0' + virtualModuleId

  function getBuildInfo() {
    let branch = ''
    let commit = ''
    try {
      branch = execSync('git rev-parse --abbrev-ref HEAD', { encoding: 'utf8' }).trim()
      commit = execSync('git rev-parse --short HEAD', { encoding: 'utf8' }).trim()
    } catch {
      // ignore
    }

    let commits: Array<{ hash: string; message: string; date: string; author: string }> = []
    try {
      const output = execSync('git log -5 --pretty=format:"%h%x09%s%x09%cd%x09%an" --date=format:"%Y-%m-%d %H:%M"', {
        encoding: 'utf8',
      }).trim()
      if (output) {
        commits = output
          .split('\n')
          .map((line) => {
            const [hash, message, date, author] = line.split('\t')
            return {
              hash: hash?.trim() || '',
              message: message?.trim() || '',
              date: date?.trim() || '',
              author: author?.trim() || '',
            }
          })
          .filter((c) => Boolean(c.hash && c.message))
      }
    } catch {
      // ignore
    }

    const now = new Date()
    const pad = (n: number) => n.toString().padStart(2, '0')
    const buildTime = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`

    return {
      version,
      branch,
      commit,
      buildTime,
      latestChanges: commits,
    }
  }

  return {
    name: 'vite-plugin-recent-changes',
    resolveId(id) {
      if (id === virtualModuleId) {
        return resolvedVirtualModuleId
      }
    },
    load(id) {
      if (id === resolvedVirtualModuleId) {
        const info = getBuildInfo()
        return `export const buildInfo = ${JSON.stringify(info)};`
      }
    },
    configureServer(server) {
      // dev 模式下每次请求自动刷新虚拟模块，保证本地新 commit 刷新页面即见
      server.middlewares.use((_req, _res, next) => {
        const mod = server.moduleGraph.getModuleById(resolvedVirtualModuleId)
        if (mod) {
          server.moduleGraph.invalidateModule(mod)
        }
        next()
      })
    },
  }
}

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
  build: {
    sourcemap: process.env.VITE_SOURCEMAP === 'true',
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules')) {
            if (id.includes('three')) return 'vendor-three'
            if (id.includes('@codemirror')) return 'vendor-codemirror'
            if (id.includes('vue') || id.includes('vue-router') || id.includes('pinia')) return 'vendor-vue'
          }
        }
      }
    }
  },
  // esbuild: {
  //   // 生产环境打包时自动丢弃所有 debugger 和 console.log
  //   drop: process.env.NODE_ENV === 'production' ? ['debugger', 'console'] : []
  // },
  base: resolveBase(),
  plugins: [vue(), lezer(), tailwindcss(), vueJsx(), buildInfoPlugin()],
  resolve: {
    alias: {
      // 强制 @codemirror/state 单文件：rollup 可能因 ESM/CJS 双入口把它解析成
      // 两个模块（同 chunk 双实例 → Unrecognized extension value），这里统一到 ESM 入口
      '@codemirror/state': resolve(import.meta.dirname, 'node_modules/@codemirror/state/dist/index.js'),
      '@': resolve(import.meta.dirname, 'src')
    }
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    cors: true,
    headers: {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, PATCH, OPTIONS',
      'Access-Control-Allow-Headers': 'X-Requested-With, content-type, Authorization',
    },
    sourcemapIgnoreList: false,
    proxy: {
      '/api': {
        // 本地开发默认连接本机 Controller；联调时可通过环境变量覆盖。
        target: process.env.VITE_API_PROXY_TARGET || 'http://127.0.0.1:8080',
        changeOrigin: true,
        ws: true,
        configure: (proxy) => {
          proxy.on('error', (err, _req, _res) => {
            console.error('[vite proxy error]', err)
          })
          proxy.on('proxyReq', (_proxyReq, req, _res) => {
            console.log('[vite proxy req]', req.method, req.url)
          })
        }
      }
    }
  }
})
