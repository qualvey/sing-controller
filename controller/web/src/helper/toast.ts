import { showNotification, type NotificationAction } from '@/helper/notification'
import { ROUTE_NAME } from '@/constant'

export type ToastType = 'success' | 'error' | 'info' | 'warning'
export type ToastAction = NotificationAction

export interface ToastOptions {
  type?: ToastType
  duration?: number
  action?: ToastAction
}

const TOAST_TYPE_MAP: Record<ToastType, 'alert-success' | 'alert-error' | 'alert-info' | 'alert-warning'> = {
  success: 'alert-success',
  error: 'alert-error',
  info: 'alert-info',
  warning: 'alert-warning',
}

/**
 * 自动识别报错/提示信息中引用的目标，生成快捷跳转动作
 */
export const detectNotificationAction = (content: string): ToastAction | undefined => {
  if (!content || typeof content !== 'string') return undefined

  // 1. route 规则 #X 引用了该 inbound / outbound
  const ruleMatch = content.match(/route\s*规则\s*#(\d+)/i)
  if (ruleMatch) {
    const ruleNum = ruleMatch[1]
    return {
      text: `修改规则 #${ruleNum}`,
      to: {
        name: ROUTE_NAME.routes,
        query: { rule: ruleNum, edit: '1' },
      },
    }
  }

  // 2. route.final 引用了该 outbound
  if (/route\.final/i.test(content)) {
    return {
      text: '前往路由设置',
      to: {
        name: ROUTE_NAME.routes,
      },
    }
  }

  // 3. DNS 规则引用：dns 规则 #X
  const dnsMatch = content.match(/dns\s*规则\s*#(\d+)/i)
  if (dnsMatch) {
    const dnsNum = dnsMatch[1]
    return {
      text: `查看 DNS 规则 #${dnsNum}`,
      to: {
        name: ROUTE_NAME.dns,
        query: { rule: dnsNum },
      },
    }
  }

  return undefined
}

/**
 * 全局统一 Toast 提示
 * 桥接至系统级的 showNotification，具备完整的主题色彩、动画、图标、倒计时与交互事件
 */
export function showToast(
  message: string,
  type?: ToastType,
  duration?: number,
  action?: ToastAction
): void
export function showToast(
  message: string,
  type?: ToastType,
  action?: ToastAction
): void
export function showToast(
  message: string,
  options?: ToastOptions
): void
export function showToast(
  message: string,
  typeOrOptions: ToastType | ToastOptions = 'info',
  durationOrAction?: number | ToastAction,
  maybeAction?: ToastAction
): void {
  let type: ToastType = 'info'
  let duration = 2500
  let action: ToastAction | undefined

  if (typeof typeOrOptions === 'object' && typeOrOptions !== null) {
    type = typeOrOptions.type || 'info'
    duration = typeOrOptions.duration ?? (typeOrOptions.action ? 6000 : 2500)
    action = typeOrOptions.action
  } else {
    if (typeOrOptions) {
      type = typeOrOptions
    }
    if (typeof durationOrAction === 'number') {
      duration = durationOrAction
      action = maybeAction
    } else if (typeof durationOrAction === 'object' && durationOrAction !== null) {
      action = durationOrAction
      duration = 6000
    }
  }

  // 若未手动传入 action，则尝试从文案中自动检测依赖报错
  if (!action) {
    action = detectNotificationAction(message)
    if (action && duration === 2500) {
      duration = 6000
    }
  }

  showNotification({
    content: message,
    type: TOAST_TYPE_MAP[type] || 'alert-info',
    timeout: duration,
    action,
  })
}
