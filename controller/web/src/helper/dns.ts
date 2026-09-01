import { api } from '@/api/controller'
import * as ipaddr from 'ipaddr.js'
import { reactive, ref } from 'vue'

const DNS_CACHE_MAX = 4096

// 响应式 DNS 缓存与版本号，方便 computed / UI 依赖组件在异步解析完成时触发重渲染
export const dnsCache = reactive(new Map<string, string>())
export const dnsRevision = ref(0)

const inflight = new Map<string, Promise<string | null>>()

export const normalizeIP = (value: string): string | null => {
  try {
    return ipaddr.parse(value).toNormalizedString()
  } catch {
    return null
  }
}

/**
 * 校验并提取纯 IP（支持裸 IPv4、裸 IPv6、[IPv6]:port 及 IPv4:port，不把域名识别为 IP）
 */
export const destinationIP = (rawValue: string): string | null => {
  const value = rawValue.trim()
  if (!value) return null

  const bareIP = normalizeIP(value)
  if (bareIP) return bareIP

  if (value.startsWith('[')) {
    const closingBracket = value.indexOf(']')
    if (closingBracket > 1) return normalizeIP(value.slice(1, closingBracket))
  }

  const lastColon = value.lastIndexOf(':')
  if (lastColon > 0 && /^\d+$/.test(value.slice(lastColon + 1))) {
    return normalizeIP(value.slice(0, lastColon))
  }

  return null
}

/**
 * 去除端口号、方括号与结尾点，提取纯 Host / 域名 / IP
 */
export const cleanHostOrDomain = (rawValue: string): string => {
  let val = rawValue.trim()
  if (!val) return ''

  if (val.startsWith('[')) {
    const closing = val.indexOf(']')
    if (closing > 1) val = val.slice(1, closing)
  } else {
    const lastColon = val.lastIndexOf(':')
    if (lastColon > 0 && /^\d+$/.test(val.slice(lastColon + 1))) {
      val = val.slice(0, lastColon)
    }
  }
  return val.replace(/\.$/, '')
}

const cacheResolvedIP = (domain: string, ip: string) => {
  dnsCache.set(domain, ip)
  dnsRevision.value++

  while (dnsCache.size > DNS_CACHE_MAX) {
    const oldest = dnsCache.keys().next().value
    if (oldest === undefined) break
    dnsCache.delete(oldest)
  }
}

/**
 * 单域名异步解析（内置并发请求合并与内存缓存）
 */
export const resolveDomain = async (rawDomain: string): Promise<string | null> => {
  const domain = cleanHostOrDomain(rawDomain)
  if (!domain) return null

  // 如果本身已经是 IP，直接返回
  const directIP = destinationIP(domain)
  if (directIP) return directIP

  // 查内存缓存
  const cached = dnsCache.get(domain)
  if (cached) return cached

  // 请求去重（Inflight）
  if (inflight.has(domain)) {
    return inflight.get(domain)!
  }

  const promise = (async () => {
    try {
      const results = await api.dnsLookup([domain])
      const ip = results[domain]
      if (ip) {
        const normalized = destinationIP(ip) || ip
        cacheResolvedIP(domain, normalized)
        return normalized
      }
      return null
    } catch {
      return null
    } finally {
      inflight.delete(domain)
    }
  })()

  inflight.set(domain, promise)
  return promise
}

/**
 * 批量域名异步解析
 */
export const resolveDomains = async (rawDomains: string[]): Promise<Record<string, string>> => {
  const uncachedDomains = new Set<string>()
  const result: Record<string, string> = {}

  for (const raw of rawDomains) {
    const domain = cleanHostOrDomain(raw)
    if (!domain) continue

    const directIP = destinationIP(domain)
    if (directIP) {
      result[domain] = directIP
      continue
    }

    const cached = dnsCache.get(domain)
    if (cached) {
      result[domain] = cached
    } else {
      uncachedDomains.add(domain)
    }
  }

  if (uncachedDomains.size > 0) {
    try {
      const lookupResults = await api.dnsLookup([...uncachedDomains])
      for (const [domain, ip] of Object.entries(lookupResults)) {
        if (ip) {
          const normalized = destinationIP(ip) || ip
          cacheResolvedIP(domain, normalized)
          result[domain] = normalized
        }
      }
    } catch {
      // 忽略网络或解析错误
    }
  }

  return result
}

/**
 * 同步读取已缓存的 IP；若未命中则在后台静默发起解析（供渲染函数/列表单元格使用）
 */
export const getCachedIPSync = (rawDomain: string): string | null => {
  const domain = cleanHostOrDomain(rawDomain)
  if (!domain) return null

  const directIP = destinationIP(domain)
  if (directIP) return directIP

  const cached = dnsCache.get(domain)
  if (cached) return cached

  // 静默触发异步解析，解析完成后会递增 dnsRevision
  resolveDomain(domain).catch(() => {})
  return null
}
