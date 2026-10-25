// sing-box 后端的 config 组装:仅暴露 clash-mode,其余配置项保持默认。
// 把 gRPC getClashModeStatus 的结果转换成 Clash 的 Config 形状,写入门面状态。
import { getSingboxClient } from '@/api/singbox/client'
import type { Config } from '@/types'
import { configs, defaultConfig } from './index'

const fetchSingboxConfigs = async (): Promise<Config> => {
  const client = getSingboxClient()?.client
  if (!client) return { ...defaultConfig }
  try {
    const status = await client.getClashModeStatus({})
    return {
      ...defaultConfig,
      mode: status.currentMode,
      'mode-list': status.modeList,
      modes: status.modeList,
    }
  } catch {
    // Clash mode is optional in sing-box (and may be unavailable during
    // startup). Keep the config facade usable instead of rejecting the
    // fire-and-forget startup request with an unhandled promise rejection.
    return { ...defaultConfig }
  }
}

export const fetchConfigs = async () => {
  configs.value = await fetchSingboxConfigs()
}

export const updateConfigs = async (cfg: Record<string, string | boolean | object | number>) => {
  if (typeof cfg.mode === 'string') {
    const client = getSingboxClient()?.client
    if (client) await client.setClashMode({ mode: cfg.mode })
  }
  fetchConfigs()
}
