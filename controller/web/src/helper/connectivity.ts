// 连接诊断。
//
// 浏览器出于安全考虑,不会告诉页面跨域请求「为什么」失败:CORS 被拒、混合内容被拦、
// 服务根本没起、网线拔了 —— 统统压成同一句不透明的 "Failed to fetch" / "Network Error"。
// 把这句话原样丢给用户等于没说,而这几种原因的处置办法完全不同。
//
// 拆开的办法是失败后补一次 no-cors 探测(它不读响应,故不受 CORS 约束):
//   能通  ⇒ 服务在,是浏览器按同源策略拦下了响应,叫后端放开 CORS;
//   不通  ⇒ 再看页面协议,HTTPS 页面访问 http:// 后端会被混合内容策略拦死,
//           这种情况下 no-cors 也必然失败,与「服务没起」无法进一步区分;
//           环回地址是例外(浏览器把 127.0.0.1 视作安全上下文,可能真是服务没起),
//           所以只能给出两种可能。
import { i18n } from '@/i18n'

// 探测失败的分类。api 层只做它能确知的判断(HTTP 状态码 / 是否超时),
// 需要额外探测才能得出的结论留给下面的 diagnose。
export type ProbeFailureKind =
  // 401 / gRPC Unauthenticated:密码不对
  | 'unauthorized'
  // 连上了但状态码不对:多半路径写错,或那压根不是一个 Clash API
  | 'http'
  // 超时:没有任何响应
  | 'timeout'
  // 不透明的网络错误,交给 diagnose 进一步区分
  | 'network'
  // WebSocket 握手或升级失败（反向代理未配置 Upgrade 等）
  | 'websocket'

export type ProbeResult =
  | { ok: true; latency: number }
  | { ok: false; latency: number; kind: ProbeFailureKind; message: string }

export type ConnectionDiagnosis =
  | 'offline'
  | 'corsBlocked'
  | 'mixedContent'
  | 'mixedContentOrUnreachable'
  | 'unreachable'
  | 'websocketFailed'

const DIAGNOSIS_MESSAGE_KEY: Record<ConnectionDiagnosis, string> = {
  offline: 'diagnosisOffline',
  corsBlocked: 'diagnosisCorsBlocked',
  mixedContent: 'diagnosisMixedContent',
  mixedContentOrUnreachable: 'diagnosisMixedContentOrUnreachable',
  unreachable: 'diagnosisUnreachable',
  websocketFailed: 'diagnosisWebSocketFailed',
}

// 浏览器把跨域网络错误说成什么,各家不一样。
export const isOpaqueNetworkError = (message: string) =>
  message.includes('Failed to fetch') || // Chromium
  message.includes('Load failed') || // WebKit
  message.includes('NetworkError') || // Firefox
  message.includes('Network Error') // axios 的包装

// 浏览器把环回地址当作安全上下文,HTTPS 页面访问它不触发混合内容拦截。
export const isLoopbackHost = (hostname: string) =>
  hostname === 'localhost' ||
  hostname.endsWith('.localhost') ||
  hostname === '[::1]' ||
  hostname === '::1' ||
  /^127(\.\d{1,3}){3}$/.test(hostname)

const DIAGNOSE_TIMEOUT = 5000

export const diagnoseConnection = async (
  url: string,
  signal?: AbortSignal,
): Promise<ConnectionDiagnosis> => {
  if (!navigator.onLine) return 'offline'

  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), DIAGNOSE_TIMEOUT)
  const onAbort = () => controller.abort()

  signal?.addEventListener('abort', onAbort, { once: true })

  try {
    await fetch(url, { mode: 'no-cors', cache: 'no-store', signal: controller.signal })
    return 'corsBlocked'
  } catch {
    if (location.protocol === 'https:' && url.startsWith('http:')) {
      try {
        return isLoopbackHost(new URL(url).hostname) ? 'mixedContentOrUnreachable' : 'mixedContent'
      } catch {
        return 'mixedContent'
      }
    }
    return 'unreachable'
  } finally {
    clearTimeout(timer)
    signal?.removeEventListener('abort', onAbort)
  }
}

/**
 * 探测 WebSocket 握手与连通性
 * 确保反向代理已配置 Upgrade / Connection 头
 */
export const probeWebSocket = (
  url: string,
  timeout = 3000,
  signal?: AbortSignal,
): Promise<{ ok: true; latency: number } | { ok: false; latency: number; kind: 'websocket' | 'timeout'; message: string }> => {
  return new Promise((resolve) => {
    if (signal?.aborted) {
      resolve({ ok: false, latency: 0, kind: 'timeout', message: 'Aborted' })
      return
    }

    const startAt = Date.now()
    const latency = () => Date.now() - startAt

    let timer: ReturnType<typeof setTimeout> | null = null
    let ws: WebSocket | null = null
    let settled = false

    const done = (result: {
      ok: true; latency: number
    } | {
      ok: false; latency: number; kind: 'websocket' | 'timeout'; message: string
    }) => {
      if (settled) return
      settled = true
      if (timer !== null) {
        clearTimeout(timer)
        timer = null
      }
      signal?.removeEventListener('abort', onAbort)
      if (ws) {
        try {
          ws.onopen = null
          ws.onerror = null
          ws.onclose = null
          ws.onmessage = null
          ws.close()
        } catch {
          // ignore
        }
        ws = null
      }
      resolve(result)
    }

    const onAbort = () => {
      done({ ok: false, latency: latency(), kind: 'timeout', message: 'Aborted' })
    }

    signal?.addEventListener('abort', onAbort, { once: true })

    timer = setTimeout(() => {
      done({ ok: false, latency: latency(), kind: 'timeout', message: 'WebSocket 连接超时' })
    }, timeout)

    try {
      ws = new WebSocket(url)
      ws.onopen = () => {
        done({ ok: true, latency: latency() })
      }
      ws.onmessage = () => {
        done({ ok: true, latency: latency() })
      }
      ws.onerror = () => {
        done({
          ok: false,
          latency: latency(),
          kind: 'websocket',
          message: 'WebSocket 握手失败（反向代理未配置 Upgrade 支持）',
        })
      }
      ws.onclose = (e) => {
        if (!settled) {
          done({
            ok: false,
            latency: latency(),
            kind: 'websocket',
            message: e.reason || `WebSocket 连接断开 (code: ${e.code || 'unknown'})，请检查反代 WebSocket 配置`,
          })
        }
      }
    } catch (e) {
      done({
        ok: false,
        latency: latency(),
        kind: 'websocket',
        message: e instanceof Error ? e.message : String(e),
      })
    }
  })
}

// 把一次探测失败翻译成可以直接显示给用户的一句话。
// 只有不透明的网络错误才值得再探一次;其余分类本身已经说清了原因。
export const describeProbeFailure = async (
  failure: { kind: ProbeFailureKind; message: string },
  url: string,
  signal?: AbortSignal,
): Promise<string> => {
  const t = i18n.global.t

  switch (failure.kind) {
    case 'unauthorized':
      return t('diagnosisUnauthorized') || '认证失败，密码或密钥错误'
    case 'http':
      return `${failure.message} — ${t('diagnosisBadEndpoint') || 'API 端点异常'}`
    case 'timeout':
      return t('diagnosisTimeout') || '连接超时'
    case 'websocket':
      return failure.message || t('diagnosisWebSocketFailed') || 'WebSocket 连接失败（反向代理未配置 Upgrade 支持）'
    default: {
      const diagnosis = await diagnoseConnection(url, signal)
      return t(DIAGNOSIS_MESSAGE_KEY[diagnosis]) || failure.message || '无法连接到控制服务'
    }
  }
}

// 会话期的失败(走 axios / gRPC,拿到的是 Error 而非 ProbeResult)同样值得诊断:
// 不透明的就补探测,能自证的原样返回。
export const describeConnectionError = async (
  message: string,
  url: string,
  signal?: AbortSignal,
): Promise<string> => {
  if (
    message.includes('WebSocket') ||
    message.includes('websocket') ||
    message.includes('Upgrade') ||
    message.includes('upgrade') ||
    message.includes('反代')
  ) {
    return message
  }

  if (!isOpaqueNetworkError(message)) return message

  const diagnosis = await diagnoseConnection(url, signal)
  return i18n.global.t(DIAGNOSIS_MESSAGE_KEY[diagnosis]) || message
}

