/**
 * 桌面环境原生能力桥接客户端
 * 适配 Wails v2 运行时机制，安全支持 Web / Desktop 双模运行
 */
import * as AppBindings from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime'

export const isDesktopApp = (): boolean => {
  return (
    typeof (window as unknown as { go?: unknown }).go !== 'undefined' ||
    window.location.hostname === 'wails.localhost' ||
    window.location.protocol === 'wails:'
  )
}

export interface DesktopInfo {
  isDesktop: boolean
  version: string
  os: string
  arch: string
  port: number
}

let cachedDesktopPort = 8080

export const getDesktopPort = (): number => cachedDesktopPort

export const desktopApi = {
  isDesktop: isDesktopApp,
  getPort: getDesktopPort,

  getInfo: async (): Promise<DesktopInfo | null> => {
    if (!isDesktopApp()) return null
    try {
      const info = await AppBindings.GetDesktopInfo()
      if (info.Port) {
        cachedDesktopPort = info.Port
      }
      return {
        isDesktop: info.IsDesktop,
        version: info.Version,
        os: info.OS,
        arch: info.Arch,
        port: info.Port || 8080,
      }
    } catch {
      return null
    }
  },

  getSystemProxy: async (): Promise<boolean> => {
    if (!isDesktopApp()) return false
    try {
      return await AppBindings.GetSystemProxyStatus()
    } catch {
      return false
    }
  },

  setSystemProxy: async (enabled: boolean): Promise<boolean> => {
    if (!isDesktopApp()) return false
    try {
      await AppBindings.SetSystemProxy(enabled)
      return true
    } catch (e) {
      console.error('Failed to set system proxy:', e)
      throw e
    }
  },

  getAutoStart: async (): Promise<boolean> => {
    if (!isDesktopApp()) return false
    try {
      return await AppBindings.GetAutoStartStatus()
    } catch {
      return false
    }
  },

  setAutoStart: async (enabled: boolean): Promise<boolean> => {
    if (!isDesktopApp()) return false
    try {
      await AppBindings.SetAutoStart(enabled)
      return true
    } catch (e) {
      console.error('Failed to set autostart:', e)
      throw e
    }
  },

  restartCore: async (): Promise<void> => {
    if (!isDesktopApp()) return
    await AppBindings.RestartCore()
  },

  sendNotification: async (title: string, message: string): Promise<void> => {
    if (!isDesktopApp()) return
    await AppBindings.SendNotification(title, message)
  },

  onSystemProxyChanged: (callback: (enabled: boolean) => void): (() => void) => {
    if (!isDesktopApp()) return () => {}
    return EventsOn('desktop:systemProxyChanged', callback)
  },

  showWindow: async (): Promise<void> => {
    if (!isDesktopApp()) return
    await AppBindings.ShowWindow()
  },

  hideWindow: async (): Promise<void> => {
    if (!isDesktopApp()) return
    await AppBindings.HideWindow()
  },

  quit: async (): Promise<void> => {
    if (!isDesktopApp()) return
    await AppBindings.Quit()
  },
}
