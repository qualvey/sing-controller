// 组装层 · 后端会话。
//
// 一次会话 = 面板为某个后端建立起来的整套运行时状态:内核探测 + 首屏数据 +
// 三条常驻流(connections / logs / traffic)。切后端、改当前后端的连接参数、
// 用户手动重连,本质都是「结束旧会话、开一条新的」,所以共用 startBackendSession——
// 重连不需要额外的响应式开关,再调一次就是了。

import { PROXY_TAB_TYPE, RULE_TAB_TYPE } from '@/constant'
import { initConnections, stopConnections } from '@/store/connections'
import { initSatistic, stopSatistic } from '@/store/overview'
import { activeBackend } from '@/store/setup'
import { watch } from 'vue'
import { api } from '@/api/controller'
import { Channel, channel, resolveAvailableChannel } from './backend'
import { fetchConfigs } from './config'
import { initLogs, stopLogs } from './logs'
import { fetchProxies, proxiesTabShow, resetProxies } from './proxies'
import { fetchRules, rulesTabShow } from './rules'
import { probeActiveBackend } from './version'

let sessionGeneration = 0

const startBackendData = () => {
  probeActiveBackend()
  fetchConfigs()
  fetchProxies()
  fetchRules()
  initConnections()
  initLogs()
  initSatistic()
}

// Stop every long-lived request before a controller process handoff. In
// particular, an in-flight gRPC-Web request against the old controller must
// not survive the UAC restart and be mistaken for a response from the new one.
export const stopBackendSession = () => {
  sessionGeneration++
  stopConnections()
  stopLogs()
  stopSatistic()
  void resetProxies()
}

export const startBackendSession = () => {
  const generation = ++sessionGeneration
  // 探测不 await:连通性提示要的正是「正在连接」这个中间态,数据流也不必等它。
  // 三条常驻流连同各自的数据在这里先丢掉,不能留到下面重建时再清 ——
  // 上一个后端的日志与统计会安静地冒充新后端的数据。
  stopConnections()
  stopLogs()
  stopSatistic()
  // The sing-box proxy facade owns its own long-lived gRPC subscriptions.
  // Reset them as well when channel resolution changes gRPC ↔ Clash.
  void resetProxies()

  // 后端被清空(登出 / 401 / 新增后端)时就停在这:常驻流上面已经关掉,
  // 否则它们会以无主状态留在 Setup 页继续运行并无限重连。
  const backend = activeBackend.value
  if (!backend) return

  rulesTabShow.value = RULE_TAB_TYPE.RULES
  proxiesTabShow.value = PROXY_TAB_TYPE.PROXIES

  // Resolve the available API channel before creating long-lived streams. gRPC is
  // preferred for sing-box; use the Clash API when the service API is unavailable.
  void api.channels().then((availability) => {
    if (generation !== sessionGeneration || activeBackend.value?.uuid !== backend.uuid) return
    const preferred = backend.type === 'singbox' ? Channel.Singbox : Channel.Clash
    const selected = resolveAvailableChannel(backend.uuid, preferred, availability)
    if (selected !== channel.value) return // channel watcher starts a fresh session
    startBackendData()
  }).catch(() => {
    if (generation !== sessionGeneration || activeBackend.value?.uuid !== backend.uuid) return
    startBackendData()
  })
}

// 会话跟着 activeBackend 走:换后端要重建,把当前后端的地址 / 密码改掉同样要重建。
watch([activeBackend, channel], startBackendSession, { immediate: true })
