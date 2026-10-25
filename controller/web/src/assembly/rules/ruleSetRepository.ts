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
