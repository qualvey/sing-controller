import { showNotification } from '@/helper/notification'

export type ToastType = 'success' | 'error' | 'info' | 'warning'

const TOAST_TYPE_MAP: Record<ToastType, 'alert-success' | 'alert-error' | 'alert-info' | 'alert-warning'> = {
  success: 'alert-success',
  error: 'alert-error',
  info: 'alert-info',
  warning: 'alert-warning',
}

/**
 * 全局统一 Toast 提示
 * 桥接至系统级的 showNotification，具备完整的主题色彩、动画、图标与倒计时效果
 */
export function showToast(message: string, type: ToastType = 'info', duration = 2500): void {
  showNotification({
    content: message,
    type: TOAST_TYPE_MAP[type] || 'alert-info',
    timeout: duration,
  })
}
