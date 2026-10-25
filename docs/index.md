---
layout: home

hero:
  name: "sing-controller"
  text: "现代化的 sing-box 托管与配置管理系统"
  tagline: "双服务架构 · 内核生命周期托管 · 动态配置校验 · 现代化 WebUI 控制台"
  image:
    src: /logo.svg
    alt: sing-controller
  actions:
    - theme: brand
      text: 快速开始 →
      link: /guide/quickstart
    - theme: alt
      text: 系统架构设计
      link: /architecture/overview
    - theme: alt
      text: API 文档与调试
      link: /api/index

features:
  - icon: 🛡️
    title: 双服务安全架构
    details: Controller 与 sing-box 独立进程运行。通过 systemd 或内置 Supervisor 精确控制内核启停与平滑重载，崩溃即时自愈。
  - icon: ⚡
    title: 声明式原子校验
    details: 配置写盘前进行 sing-box 真实内存试跑校验。语法错误或端口冲突自动回滚，杜绝断网与配置损坏。
  - icon: 🎯
    title: sing-box gRPC 第一公民
    details: 原生以 sing-box 官方 gRPC 服务与 API 为核心，同时向前兼容 Clash External Controller 协议。
  - icon: 🌐
    title: 响应式高颜值 WebUI
    details: 内置 Vue 3 + TailwindCSS 控制面板，提供实时流量监控、节点延迟测速、路由规则可视化与 CodeMirror 配置编辑器。
  - icon: 📦
    title: 多平台原生支持
    details: 专为 Linux (systemd / OpenWrt apk)、Windows 桌面托管模式及 Docker 容器环境量身定制。
  - icon: 📖
    title: 自动化 OpenAPI 体系
    details: 代码级注解驱动生成 Swagger/OpenAPI 规范，提供现代化在线交互式调试（Scalar UI），文档与代码零距离同步。
---
