// sing-box service API（gRPC-Web）客户端（移植自 zashboard，MIT）
// 经 controller 的 /api/grpc/* 反向代理同源访问 services[type=api]：
// 事件流式推送（连接/日志/状态/代理组/出站），比 clash WS 轮询更实时。
import { createClient, type Client } from '@connectrpc/connect'
import { createGrpcWebTransport } from '@connectrpc/connect-web'
import { StartedService } from '@/gen/daemon/started_service_pb'
import type { MessageInitShape } from '@bufbuild/protobuf'
import type { DescMessage } from '@bufbuild/protobuf'

const transport = createGrpcWebTransport({ baseUrl: import.meta.env.BASE_URL + 'api/grpc' })

export type StartedClient = Client<typeof StartedService>

let client: StartedClient | null = null
export const getClient = (): StartedClient => {
  client ??= createClient(StartedService, transport)
  return client
}

// ============ 订阅流（server-streaming，事件推送） ============

export type SubscriptionId = 'logs' | 'connections' | 'status' | 'groups' | 'outbounds'

const INTERVAL = 1_000_000_000n // 1s（SubscribeStatus/SubscribeConnections 的 interval 参数，ns）

export async function* subscribeStream<Req extends DescMessage, Res extends DescMessage>(
  method: (c: StartedClient) => AsyncIterable<Res>,
  _req?: MessageInitShape<Req>
): AsyncIterable<Res> {
  for await (const msg of method(getClient())) {
    yield msg as Res
  }
}

// 便捷订阅器：返回 AsyncIterable，组件内 for await 消费；AbortSignal 停止
export const subscribeLogs = (signal?: AbortSignal) => getClient().subscribeLog({}, { signal })
export const subscribeStatus = (signal?: AbortSignal) =>
  getClient().subscribeStatus({ interval: INTERVAL }, { signal })
export const subscribeGroups = (signal?: AbortSignal) => getClient().subscribeGroups({}, { signal })
export const subscribeOutbounds = (signal?: AbortSignal) => getClient().subscribeOutbounds({}, { signal })
export const subscribeConnections = (signal?: AbortSignal) =>
  getClient().subscribeConnections({ interval: INTERVAL }, { signal })

// ============ unary 操作 ============

export const getVersion = () => getClient().getVersion({})
export const getStartedAt = () => getClient().getStartedAt({})
export const selectOutbound = (groupTag: string, outboundTag: string) =>
  getClient().selectOutbound({ groupTag, outboundTag })
export const setClashMode = (mode: string) => getClient().setClashMode({ mode })
export const uRLTest = (outboundTag: string) => getClient().uRLTest({ outboundTag })
export const closeConnection = (id: string) => getClient().closeConnection({ id })
export const closeAllConnections = () => getClient().closeAllConnections({})

// 便捷：单次取当前状态快照（connHistory 等场景）
export const snapshotGroups = async () => {
  for await (const g of getClient().subscribeGroups({})) return g
  return null
}
