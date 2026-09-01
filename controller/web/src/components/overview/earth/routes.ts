import { connectionAccessor } from '@/assembly/connections'
import { proxyMap } from '@/assembly/proxies'
import { cleanHostOrDomain, destinationIP, normalizeIP, resolveDomains } from '@/helper/dns'
import type { Connection } from '@/types'
import type { EarthHostTraffic, EarthLocation, EarthRoute } from './types'

interface RawCandidate {
  rawTarget: string
  isDirectIP: boolean
  upload: number
  download: number
  host: string
  downloaded: number
}

interface ResolvedCandidate {
  destinationIP: string
  upload: number
  download: number
  host: string
  downloaded: number
}

/**
 * 从当前活跃连接中提取目标（直连 IP、域名或出站节点服务器）
 */
const extractRawCandidates = (connections: readonly Connection[]): RawCandidate[] => {
  const accessor = connectionAccessor()
  const rawCandidates: RawCandidate[] = []

  for (const connection of connections) {
    const rawDest = accessor.destination(connection) || ''
    const cleanDest = cleanHostOrDomain(rawDest)
    const directIP = destinationIP(cleanDest) || destinationIP(rawDest)

    let target = ''
    let isDirectIP = false

    if (directIP) {
      target = directIP
      isDirectIP = true
    } else if (cleanDest && !cleanDest.includes(' ')) {
      // 目标是域名
      target = cleanDest
    } else {
      // 目标为空或无效时，回退到出站代理节点
      const outbound = accessor.fromOutbound?.(connection) || (connection as any).outbound || ''
      if (outbound && outbound !== 'direct' && outbound !== 'block' && outbound !== 'dns-out') {
        const node = proxyMap.value[outbound]
        const serverRaw = (node as any)?.server || (node as any)?.host || ''
        const cleanServer = cleanHostOrDomain(serverRaw)
        const serverIP = destinationIP(cleanServer)
        if (serverIP) {
          target = serverIP
          isDirectIP = true
        } else if (cleanServer) {
          target = cleanServer
        }
      }
    }

    if (!target) continue

    const rawHost = accessor.hostname(connection).trim().replace(/\.$/, '')

    rawCandidates.push({
      rawTarget: target,
      isDirectIP,
      upload: Math.max(0, connection.uploadSpeed),
      download: Math.max(0, connection.downloadSpeed),
      host: rawHost || target,
      downloaded: Math.max(0, accessor.download(connection)),
    })
  }

  return rawCandidates
}

/**
 * 借助全局 DNS 服务批量将域名解析为 IP
 */
const resolveCandidateIPs = async (
  rawCandidates: readonly RawCandidate[],
): Promise<ResolvedCandidate[]> => {
  const domainList: string[] = []

  for (const candidate of rawCandidates) {
    if (!candidate.isDirectIP && candidate.rawTarget) {
      domainList.push(candidate.rawTarget)
    }
  }

  const resolvedMap = domainList.length > 0 ? await resolveDomains(domainList) : {}

  const resolved: ResolvedCandidate[] = []

  for (const candidate of rawCandidates) {
    let finalIP = ''
    if (candidate.isDirectIP) {
      finalIP = candidate.rawTarget
    } else {
      const resolvedFromDNS = resolvedMap[candidate.rawTarget]
      if (resolvedFromDNS) {
        finalIP = destinationIP(resolvedFromDNS) || resolvedFromDNS
      }
    }

    if (!finalIP) continue

    resolved.push({
      destinationIP: finalIP,
      upload: candidate.upload,
      download: candidate.download,
      host: candidate.host,
      downloaded: candidate.downloaded,
    })
  }

  return resolved
}

const coordinateKey = ({ latitude, longitude }: EarthLocation) =>
  `${latitude.toFixed(4)},${longitude.toFixed(4)}`

const mergeTopHosts = (...groups: EarthHostTraffic[][]) =>
  groups
    .flat()
    .sort((left, right) => right.downloaded - left.downloaded)
    .slice(0, 5)

export const buildEarthRoutes = async (
  connections: readonly Connection[],
  originIP: string,
  locale: string,
  lookup: (ips: string[], locale: string) => Promise<Record<string, EarthLocation | null>>,
) => {
  const normalizedOrigin = normalizeIP(originIP)

  if (!normalizedOrigin) return { routes: [] as EarthRoute[], origin: null }

  const rawCandidates = extractRawCandidates(connections)
  const candidates = await resolveCandidateIPs(rawCandidates)

  const ips = new Set<string>([normalizedOrigin])
  for (const candidate of candidates) ips.add(candidate.destinationIP)

  const locations = await lookup([...ips], locale)
  const origin = locations[normalizedOrigin]

  if (!origin) return { routes: [] as EarthRoute[], origin: null }

  const aggregated = new Map<string, EarthRoute>()

  for (const candidate of candidates) {
    const destination = locations[candidate.destinationIP]
    if (!destination) continue

    const path: EarthRoute['path'] = [
      { ...origin, role: 'origin' },
      { ...destination, role: 'destination' },
    ]
    const key = path.map((point) => `${point.role}:${coordinateKey(point)}`).join('>')
    const existing = aggregated.get(key)

    if (existing) {
      existing.connections += 1
      existing.upload += candidate.upload
      existing.download += candidate.download
      existing.topHosts = mergeTopHosts(
        existing.topHosts,
        candidate.host ? [{ host: candidate.host, downloaded: candidate.downloaded }] : [],
      )
    } else {
      aggregated.set(key, {
        key,
        path,
        connections: 1,
        upload: candidate.upload,
        download: candidate.download,
        topHosts: candidate.host
          ? [{ host: candidate.host, downloaded: candidate.downloaded }]
          : [],
      })
    }
  }

  return { routes: [...aggregated.values()], origin }
}
