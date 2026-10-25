// api 层公共入口。
// 1. 独立封装的 sing-controller 专有配置管理 API (根据 API.md)
export * from './controller'

// 2. zashboard 遗产：Clash 兼容接口与拦截器
import './http'
export * from './clash'
