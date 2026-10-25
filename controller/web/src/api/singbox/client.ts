import { StartedService } from '@/gen/daemon/started_service_pb'
import { getSingboxSecret, getSingboxUrlFromBackend } from '@/helper/utils'
import { activeBackend } from '@/store/setup'
import type { Backend } from '@/types'
import { createClient, type Client, type Interceptor } from '@connectrpc/connect'
import { createGrpcWebTransport } from '@connectrpc/connect-web'

const authInterceptor = (secret: string): Interceptor => {
  return (next) => (request) => {
    if (secret) request.header.set('Authorization', `Bearer ${secret}`)
    return next(request)
  }
}

export class SingboxClient {
  readonly client: Client<typeof StartedService>

  constructor(baseUrl: string, secret: string) {
    this.client = createClient(
      StartedService,
      createGrpcWebTransport({
        baseUrl,
        interceptors: secret ? [authInterceptor(secret)] : [],
      }),
    )
  }
}

// --- Singleton manager keyed to the active backend's sing-box channel ---

let current: { key: string; client: SingboxClient } | null = null

// const backendKey = (backend: Backend) =>
//   `${backend.uuid}|${getSingboxUrlFromBackend(backend)}|${getSingboxSecret(backend)}`

export const getSingboxClient = (): SingboxClient | null => {
  const baseUrl =  getSingboxUrlFromBackend() 
  const key = '1213'
  current = {  key, client: new SingboxClient(baseUrl,'')}
  return current.client
}

import { api } from '@/api/controller'
import { getClashWsUrl } from '@/api/clash'
import { probeWebSocket, type ProbeResult } from '@/helper/connectivity'

// Probe the sing-box channel for the Setup connectivity test.
export const probeSingboxChannel = async (
  backend: Backend,
  timeout = 10000,
  signal?: AbortSignal,
): Promise<ProbeResult> => {
  const controller = new AbortController()
  const timeoutId = setTimeout(() => controller.abort(), timeout)
  const onAbort = () => controller.abort()
  signal?.addEventListener('abort', onAbort, { once: true })

  const startAt = Date.now()
  const latency = () => Date.now() - startAt

  // 1. 优先通过 controller /api/status 探测核心与链路状态
  try {
    const status = await api.status()
    if (status) {
      if (status.core && !status.core.alive) {
        return {
          ok: false,
          latency: latency(),
          kind: 'network',
          message: status.core.error || 'sing-box 核心未运行',
        }
      }
      if (status.core_alive === false) {
        return {
          ok: false,
          latency: latency(),
          kind: 'network',
          message: status.core_error || 'sing-box 核心未运行',
        }
      }

      // HTTP 与 Controller 正常，核心存活，立即判定成功（Clash API / WebSocket 作为可选项不阻断主链路）
      return { ok: true, latency: latency() }
    }
  } catch {
    // continue to gRPC check
  }

  // 2. 备选探测 sing-box gRPC 接口
  const baseUrl = getSingboxUrlFromBackend()
  if (!baseUrl) {
    clearTimeout(timeoutId)
    signal?.removeEventListener('abort', onAbort)
    return { ok: false, latency: latency(), kind: 'network', message: 'No sing-box URL' }
  }

  const client = createClient(
    StartedService,
    createGrpcWebTransport({
      baseUrl,
    }),
  )

  try {
    await client.getVersion({}, { signal: controller.signal })
    // gRPC 正常，同样验证 WebSocket
    const wsUrl = getClashWsUrl(backend, 'traffic')
    const wsRes = await probeWebSocket(wsUrl, Math.min(timeout, 3000), controller.signal)
    if (!wsRes.ok) {
      return {
        ok: false,
        latency: latency(),
        kind: 'websocket',
        message: wsRes.message || 'WebSocket 握手失败（反向代理未配置 Upgrade 支持）',
      }
    }
    return { ok: true, latency: latency() }
  } catch (e) {
    return {
      ok: false,
      latency: latency(),
      kind: controller.signal.aborted ? 'timeout' : 'network',
      message: e instanceof Error ? e.message : String(e),
    }
  } finally {
    clearTimeout(timeoutId)
    signal?.removeEventListener('abort', onAbort)
  }
}
