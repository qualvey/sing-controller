# API 体系与设计规范

`sing-controller` 提供了一套统一的 RESTful 管理接口，用于控制面板、自动化脚本与外部运维工具与代理服务交互。

---

## 核心设计原则

### 1. 通信协议与基地址
- **Base URL**: `http://<listen_host>:<listen_port>`（默认 `http://127.0.0.1:8080`）
- **数据格式**: 请求与响应默认均为 `application/json; charset=utf-8`。
- **CORS 支持**: 开发环境下原生支持跨域携带凭证与自定义 Header。

### 2. 鉴权机制 (Optional Secret)
当 Controller 启动时通过 `-secret <token>` 或 `controller.json` 中配置了 `server.secret` 时：
- **请求头鉴权**: `X-Secret: <your_secret>`
- **URL 查询参数鉴权**: `?token=<your_secret>`（便于 EventStream 与 WebSocket 场景）
- 鉴权未通过时统一返回 `401 Unauthorized`:
  ```json
  { "error": "invalid secret" }
  ```

---

## 接口分层与服务架构

> [!IMPORTANT] 第一公民原则
> 本项目以 **sing-box gRPC API** 为第一公民。优先保证 sing-box 官方 gRPC 服务的直通与可用性，Clash API 端点作为兼容生态的可选项。

```
                    ┌─────────────────────────┐
                    │      HTTP Request       │
                    └────────────┬────────────┘
                                 │
                     ┌───────────▼───────────┐
                     │   X-Secret 鉴权中间件  │
                     └───────────┬───────────┘
                                 │
         ┌───────────────────────┼────────────────────────┐
         │                       │                        │
┌────────▼────────┐    ┌─────────▼────────┐    ┌──────────▼───────────┐
│ /api/* 控制面    │    │ /api/grpc/*      │    │ /api/clash/* (可选)   │
│ 配置/状态/用户/  │    │ sing-box 原生服务 │    │ External Controller  │
│ 订阅/内核生命周期│    │ (gRPC-Web / WS)  │    │ 兼容代理转发          │
└─────────────────┘    └──────────────────┘    └──────────────────────┘
```

---

## 在线交互式文档

不再需要依赖容易与代码脱节的手写 Markdown，Controller 内置了轻量级在线交互调试页面：
- **Scalar 现代化接口面板**: `http://localhost:8080/api/docs`
- **OpenAPI 3.0 / Swagger 规范**: `http://localhost:8080/api/swagger.json`
- 详细说明参见 [在线交互文档](./interactive.md)。
