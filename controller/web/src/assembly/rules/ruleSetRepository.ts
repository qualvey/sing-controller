import { api } from '@/api'
import { compareVersions } from '@/assembly/version'

/** Normalize values returned as "1.14.0", "sing-box 1.14.0", or "... v1.14.0". */
export const normalizeSingboxVersion = (value: unknown): string => {
  const text = String(value ?? '').trim()
  const match = text.match(/(?:^|\s|v)(\d+\.\d+(?:\.\d+)?(?:-[0-9A-Za-z.-]+)?)(?:\s|$)/i)
  return match?.[1] || ''
}

/** The repository representation uses one remote rule set with multiple tags. */
export const supportsMultiTagRuleSet = (version: unknown): boolean => {
  const normalized = normalizeSingboxVersion(version)
  return Boolean(normalized) && compareVersions(normalized, '1.4.0') >= 0
}

export const hasRuleSetTemplate = (url: unknown): boolean => String(url ?? '').includes('{tag}')

/** 读取 route.rule_set 段配置的全部 tag（单个 rule set 的 tag 可为字符串或数组）。 */
export const fetchRuleSetTags = async (): Promise<string[]> => {
  const res = await api.ruleSets()
  const tags: string[] = []
  const seen = new Set<string>()

  for (const item of res.rule_sets || []) {
    const rawTag = item.rule_set?.tag
    const values = Array.isArray(rawTag) ? rawTag : rawTag == null ? [] : [rawTag]
    for (const value of values) {
      const tag = String(value ?? '').trim()
      if (tag && !seen.has(tag)) {
        seen.add(tag)
        tags.push(tag)
      }
    }
  }

  return tags
}
