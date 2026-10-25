import { buildInfo } from 'virtual:recent-changes'

export function printAppBanner() {
  const isDev = import.meta.env.DEV
  const brand = 'sing-box WebUI'
  const version = `v${buildInfo.version || '0.11.0'}`
  const branchInfo = `${buildInfo.branch ? `${buildInfo.branch}#` : ''}${buildInfo.commit || 'dev'}`
  const envTag = isDev ? 'DEV' : 'PROD'

  // 将 buildInfo 挂载到 window 便于控制台调试排查
  if (typeof window !== 'undefined') {
    ;(window as unknown as Record<string, unknown>).__BUILD_INFO__ = buildInfo
  }

  // 1. 顶部状态徽标
  console.log(
    `%c ${brand} %c ${version} %c ${branchInfo} %c ${envTag} `,
    'background: #0f172a; color: #38bdf8; font-weight: bold; border-radius: 4px 0 0 4px; padding: 3px 6px; font-size: 11px;',
    'background: #1e293b; color: #f8fafc; font-weight: 500; padding: 3px 6px; font-size: 11px;',
    'background: #334155; color: #cbd5e1; font-family: monospace; padding: 3px 6px; font-size: 11px;',
    isDev
      ? 'background: #ca8a04; color: #ffffff; font-weight: bold; border-radius: 0 4px 4px 0; padding: 3px 6px; font-size: 11px;'
      : 'background: #16a34a; color: #ffffff; font-weight: bold; border-radius: 0 4px 4px 0; padding: 3px 6px; font-size: 11px;',
  )

  // 2. 最新改动重点突出展示
  const latest = buildInfo.latestChanges?.[0]
  if (latest) {
    console.log(
      `%c🚀 最新功能改动: %c${latest.message} %c(${latest.hash} · ${latest.date})`,
      'color: #059669; font-weight: bold; font-size: 12px;',
      'color: #0f172a; font-weight: 600; font-size: 12px;',
      'color: #64748b; font-size: 11px;'
    )
  }

  // 3. 近期历史提交（折叠展示，不遮挡控制台）
  if (buildInfo.latestChanges && buildInfo.latestChanges.length > 1) {
    console.groupCollapsed(
      `%c📜 近期功能改动清单 (%c${buildInfo.latestChanges.length} 条更新%c) %c点击展开详情`,
      'color: #0284c7; font-weight: 600; font-size: 11px;',
      'color: #ea580c; font-weight: bold; font-size: 11px;',
      'color: #0284c7; font-weight: 600; font-size: 11px;',
      'color: #94a3b8; font-size: 10px;'
    )
    buildInfo.latestChanges.forEach((c, idx) => {
      const isFirst = idx === 0
      console.log(
        `%c${isFirst ? '★' : '•'} [%c${c.hash}%c] %c${c.message} %c(${c.date}${c.author ? ` · ${c.author}` : ''})`,
        isFirst ? 'color: #f59e0b; font-weight: bold;' : 'color: #94a3b8;',
        'color: #0284c7; font-family: monospace; font-weight: bold;',
        isFirst ? 'color: #f59e0b;' : 'color: #94a3b8;',
        'color: #1e293b; font-weight: 500;',
        'color: #94a3b8; font-size: 11px;'
      )
    })
    console.log(
      `%c构建时间: %c${buildInfo.buildTime}`,
      'color: #94a3b8; font-size: 11px;',
      'color: #475569; font-size: 11px; font-weight: 500;'
    )
    console.groupEnd()
  }
}
