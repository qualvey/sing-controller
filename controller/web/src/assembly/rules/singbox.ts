// sing-box 后端 rules 适配层：将 sing-box /routes 与 /rule-sets 适配为 zashboard Rule / RuleProvider 状态。
import { api, type RouteRule } from '@/api'
import type { Rule, RuleProvider } from '@/types'
import { ruleProviderList, rules } from './index'

function extractRuleSummary(rule: RouteRule): { type: string; payload: string } {
  if (rule.type === 'logical') {
    const mode = rule.mode || 'and'
    const count = Array.isArray(rule.rules) ? rule.rules.length : 0
    return {
      type: rule.invert ? `!logical(${mode})` : `logical(${mode})`,
      payload: `${count} 条子规则`,
    }
  }

  // 按优先级匹配主要字段
  if (rule.rule_set) {
    const p = Array.isArray(rule.rule_set) ? rule.rule_set.join(', ') : String(rule.rule_set)
    return { type: 'RuleSet', payload: p }
  }
  if (rule.geosite) {
    const p = Array.isArray(rule.geosite) ? rule.geosite.join(', ') : String(rule.geosite)
    return { type: 'GeoSite', payload: p }
  }
  if (rule.geoip) {
    const p = Array.isArray(rule.geoip) ? rule.geoip.join(', ') : String(rule.geoip)
    return { type: 'GeoIP', payload: p }
  }
  if (rule.domain_suffix) {
    const p = Array.isArray(rule.domain_suffix)
      ? rule.domain_suffix.join(', ')
      : String(rule.domain_suffix)
    return { type: 'DomainSuffix', payload: p }
  }
  if (rule.domain_keyword) {
    const p = Array.isArray(rule.domain_keyword)
      ? rule.domain_keyword.join(', ')
      : String(rule.domain_keyword)
    return { type: 'DomainKeyword', payload: p }
  }
  if (rule.domain) {
    const p = Array.isArray(rule.domain) ? rule.domain.join(', ') : String(rule.domain)
    return { type: 'Domain', payload: p }
  }
  if (rule.ip_cidr) {
    const p = Array.isArray(rule.ip_cidr) ? rule.ip_cidr.join(', ') : String(rule.ip_cidr)
    return { type: 'IP-CIDR', payload: p }
  }
  if (rule.source_ip_cidr) {
    const p = Array.isArray(rule.source_ip_cidr)
      ? rule.source_ip_cidr.join(', ')
      : String(rule.source_ip_cidr)
    return { type: 'Src-IP-CIDR', payload: p }
  }
  if (rule.port) {
    const p = Array.isArray(rule.port) ? rule.port.join(', ') : String(rule.port)
    return { type: 'Dst-Port', payload: p }
  }
  if (rule.source_port) {
    const p = Array.isArray(rule.source_port)
      ? rule.source_port.join(', ')
      : String(rule.source_port)
    return { type: 'Src-Port', payload: p }
  }
  if (rule.protocol) {
    const p = Array.isArray(rule.protocol) ? rule.protocol.join(', ') : String(rule.protocol)
    return { type: 'Protocol', payload: p }
  }
  if (rule.inbound) {
    const p = Array.isArray(rule.inbound) ? rule.inbound.join(', ') : String(rule.inbound)
    return { type: 'Inbound', payload: p }
  }

  // 兜底：提取任意第一个有值的匹配条件
  const ignoreKeys = new Set(['action', 'outbound', 'type', 'invert', 'sniffer', 'server'])
  const remainingKeys = Object.keys(rule).filter((k) => !ignoreKeys.has(k) && (rule as any)[k] != null)
  if (remainingKeys.length > 0) {
    const k = remainingKeys[0]
    const v = (rule as any)[k]
    const p = Array.isArray(v) ? v.join(', ') : String(v)
    const extraCount = remainingKeys.length - 1
    return {
      type: rule.invert ? `!${k}` : k,
      payload: extraCount > 0 ? `${p} (+${extraCount})` : p,
    }
  }

  return {
    type: rule.invert ? '!MatchAll' : 'MatchAll',
    payload: '',
  }
}

function extractProxyAction(rule: RouteRule, finalOutbound: string): string {
  const action = rule.action || 'route'
  if (action === 'route') {
    return rule.outbound || finalOutbound || 'DIRECT'
  }
  if (action === 'direct') return 'DIRECT'
  if (action === 'reject') return 'REJECT'
  if (action === 'bypass') return 'BYPASS'
  if (action === 'hijack-dns') return 'hijack-dns'
  if (action === 'sniff') {
    return rule.sniffer
      ? `sniff(${Array.isArray(rule.sniffer) ? rule.sniffer.join(',') : rule.sniffer})`
      : 'sniff'
  }
  if (action === 'resolve') {
    return rule.server ? `resolve(${rule.server})` : 'resolve'
  }
  return rule.outbound || action
}

export const fetchRules = async () => {
  try {
    const [routesRes, ruleSetsRes] = await Promise.allSettled([api.routes(), api.ruleSets()])

    if (routesRes.status === 'fulfilled') {
      const data = routesRes.value
      const finalOutbound = data.final || ''
      rules.value = (data.routes || []).map((item, idx) => {
        const routeRule = item.rule || {}
        const { type, payload } = extractRuleSummary(routeRule)
        const proxy = extractProxyAction(routeRule, finalOutbound)

        return {
          type,
          payload,
          proxy,
          size: -1,
          uuid: item.id || String(idx),
          index: idx + 1,
          disabled: item.disabled === true,
        } as Rule
      })
    } else {
      rules.value = []
    }

    if (ruleSetsRes.status === 'fulfilled') {
      const rsData = ruleSetsRes.value
      const rawList = rsData.rule_sets || []
      ruleProviderList.value = rawList.map((item) => {
        const rs = item.rule_set || {}
        return {
          name: rs.tag || item.id,
          type: rs.type || 'remote',
          vehicleType: rs.type === 'inline' ? 'Inline' : 'HTTP',
          behavior: rs.format || 'binary',
          ruleCount: Array.isArray(rs.rules) ? rs.rules.length : -1,
          format: rs.format || '',
          updatedAt: '',
        } as RuleProvider
      })
    } else {
      ruleProviderList.value = []
    }
  } catch {
    rules.value = []
    ruleProviderList.value = []
  }
}
