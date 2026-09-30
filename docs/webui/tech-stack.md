# 前端技术栈与通信协议

---

## 核心技术选型

WebUI 基于现代化 Web 生态构建，强调高性能、低包体积与极致交互：

| 领域 | 选型 | 版本 / 说明 |
| :--- | :--- | :--- |
| **基础框架** | Vue 3 (Composition API) | `<script setup lang="ts">` 单文件组件 |
| **构建工具** | Vite 8 + vue-tsc | 毫秒级 HMR 模块热重载 |
| **CSS 样式系统** | TailwindCSS v4 + DaisyUI | 现代化工具类优先，暗黑模式原生支持 |
| **代码编辑** | CodeMirror 6 | 轻量、按需加载的 JSON 与规则编辑器 |
| **类型生成** | json-schema-to-typescript | 自动从 sing-box schema 生成 TypeScript 类型定义 |
| **状态管理** | Pinia | 轻量模块化响应式状态树 |
| **网络通信** | Axios + 原生 WebSocket | RESTful 控制 API 与实时日志/流量流 |

---

## 通信协议矩阵

| 场景 | 协议 | 端点 | 说明 |
| :--- | :--- | :--- | :--- |
| **常规配置管理** | HTTP/REST | `/api/*` | 增删改查节点、入站、出站、DNS、用户池 |
| **日志实时流** | Server-Sent Events (SSE) | `/api/core/logs/stream` | 控制器内核运行日志实时推送 |
| **实时流量与监控** | WebSocket | `/api/realtime` | 毫秒级上下行速率与连接数广播 |
| **代理内核原生通信** | gRPC-Web / WS | `/api/grpc/*` | 直接与 sing-box 原生服务对话（第一公民） |
| **Clash 生态兼容** | HTTP 代理转发 | `/api/clash/*` | 兼容旧版生态与可视化看板（可选） |
