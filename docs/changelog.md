# 更新日志 (Changelog)

本文档记录 `sing-controller` 与 WebUI 各版本的更新历史与关键变更。

---

## v0.11.0（2026-09-01）

### 新功能
- **WebSocket 反代连通性校验与错误诊断**：
  - 新增前端 WebSocket 握手探活机制（`probeWebSocket`），支持自定义超时、错误分类与 `AbortSignal`
  - 连通性探测全面集成 WebSocket 校验：Clash 通道探测（`/api/clash/traffic`）、sing-box 通道探测及全局会话初始化均严格校验 WS 连通性
  - 针对 Nginx / Caddy 等反向代理漏配 `Upgrade` / `Connection` 请求头导致的实时流静默失效，提供直观的错误弹窗、状态指示栏（Controller / sing-box 核心 / WebSocket 三状态）及一键复制 Nginx 反代配置参考
- **全新控制台与双通道架构**：
  - 深度重构仪表盘，全面支持节点代理组、实时连接、实时日志、规则集、DNS、入站与用户管理
  - 统一支持 sing-box 原生 API 与 Clash 兼容 API 自动探测与能力适配

### 优化与修复
- 补充多语言诊断文案（包含 WebSocket 握手失败、CORS 拦截、混合内容限制等，支持 zh, zh-tw, en, ru）
- 优化 WebSocket 重连与连接状态管理

---

## v0.10.0（2026-08-10）

### 新功能
- **重载方式自动适配（OpenWrt / Alpine 支持）**：reload.mode 新增并默认 `auto`——自动探测 init 机制，有 systemd 用 systemd（systemctl reload），否则按序探测 openrc（`rc-service <svc> reload`，Alpine）、OpenWrt procd（`/etc/init.d/<svc> reload` / `service <svc> reload`）、SysV（`service <svc> reload`）；旧配置自动迁移为 auto。
- packaging 新增参考 init 脚本：`packaging/openrc/sing-box`（Alpine）、`packaging/openwrt/sing-box.init`（procd），均实现 reload = 发 SIGHUP。

---

## v0.9.0（2026-08-10）

### 新功能
- **shadowsocks 入站控制**：Inbounds 页新增 shadowsocks 表单，支持 9 种加密算法、自动生成随机高强度密码。
- **用户池与密码生成**：用户池统一分配与绑定入站，新增 `/api/tools/password` 密码生成工具。

---

## v0.8.0（2026-08-10）

### 新功能
- **日/夜主题切换**：支持右上角暗色/亮色切换，CodeMirror 编辑器外壳与高亮同步切换。
- **移动端全响应式设计**：窄屏自动折叠侧边栏为抽屉式汉堡导航。
