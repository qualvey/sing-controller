# 技术栈与通信协议选型：集群通道

> 状态：评审稿 v0.1（未实现） · 日期：2026-08-10 · 依据：需求评审 Q「通信方式参考 komari」
> 相关文档：[需求评审](./requirement-review.md) · [集群架构设计](./cluster-architecture.md)

## 1. 选型目标

- 主节点（面板+Hub）↔ 从节点（controller+Agent）控制面通道
- **必须 NAT 友好**（从节点在 NAT 后，公网无入站端口）→ 从节点必须主动出站连接
- 双向实时：主→从下发命令；从→主注册/心跳/状态/结果
- 与现有 Go + Vue3 技术栈低摩擦，单二进制交付
- 可加密、可认证、可重连、实现成本可控

## 2. 通信协议选型

| 方案 | 双向实时 | NAT 友好 | 实现成本 | 与现有栈匹配 | 结论 |
|---|---|---|---|---|---|
| **WebSocket + JSON-RPC 2.0**（komari v2 范式） | ✅ 全双工 | ✅ 主动出站 | 低（Go 标准库 / 已有 ws 依赖） | ✅ 前后端原生支持 | **选用** |
| gRPC（双向流） | ✅ | ✅ 主动出站 | 中（需要 protobuf + 连接管理，跨语言重） | 部分（前端已有 connect-web，但后端引入重） | 备选，P1 评估 |
| 纯 HTTP REST 轮询 | ❌ 主→从有延迟 | ✅ | 低 | ✅ | 降级通道（komari 同款兜底） |
| MQTT | ✅ | ✅（broker 中转） | 中（引入外部 broker 依赖） | 弱 | 否决（引入第三方组件） |
| TCP 长连接自定义协议 | ✅ | ✅ | 高（自造协议/编解码） | 弱 | 否决（重复造轮子） |

**结论：WebSocket 长连接 + JSON-RPC 2.0 消息规范**，与 komari v2 协议对齐：

- 端点：`/cluster/v2/rpc?token=<node-token>`（主节点 Hub 提供，端口可配，默认 25774 风格的自定义端口）
- 消息：JSON-RPC 2.0（`id/method/params`），方法名命名空间化（`node.*` / `hub.*` / `task.*`）
- 可选 Gzip 压缩（komari v2 默认开启；控制面消息小，压缩收益有限，P1 再开）
- 降级：WebSocket 不可达时退化为 HTTP POST 上报（状态），此时主节点失去主动下发能力（与 komari 一致，接受该限制）

**理由**：
1. komari 已验证该范式在 NAT 小鸡场景的健壮性（指数退避重连、HTTP 降级、轻量）
2. 前后端均可直接消费 JSON/WS，无需引入新语言栈；Go 侧可复用 sing-box 已依赖的 `github.com/sagernet/ws`（go.mod 已有），前端可用原生 WebSocket（或 @vueuse/useWebSocket，依赖已存在）
3. 消息量小（配置/状态/命令），JSON 可读可调试，protobuf 收益低

## 3. 技术栈增量

### 3.1 后端（Go，沿用现有单二进制）

| 组件 | 选择 | 理由 |
|---|---|---|
| WS 服务端（Hub） | `github.com/sagernet/ws`（已依赖）或 `nhooyr.io/websocket` | 零新增依赖优先；二者 API 均轻量，选型时以测试为准 |
| WS 客户端（Agent） | 同上库的客户端侧 | 对称实现 |
| 连接管理 | 自研轻量会话层（注册表 + 心跳 + 重连状态机） | 节点规模 ≤50，不需要框架 |
| 任务执行 | 复用现有 `Handler` 能力（store 校验管线 / reload 自动适配 / REST 语义） | 命令执行器 = 现有 handler 的接口化封装，不重写逻辑 |
| 配置存储（主侧） | 沿用 JSON 文件 + 旁车 meta 模式；节点注册信息存 `cluster.json`（主） | 与现有 store 风格一致，无 DB 依赖 |
| 鉴权 | 现有 `-secret`（面板/API）+ 节点 Token（通道） | 两套各司其职 |

### 3.2 前端（Vue3 + Element Plus，沿用）

| 组件 | 选择 | 理由 |
|---|---|---|
| 节点管理页 | 新 `NodesView.vue`（列表/详情/配置下发/重载） | 复用 Inbounds/Config 编辑组件模式 |
| 实时状态 | WebSocket 或轮询 `/api/cluster/nodes` | MVP 轮询（5s）+ 状态上报驱动；P1 接 WS 推送 |
| 配置编辑 | 复用 ResourceSourceTab / ConfigView 组件 | 下发前预览/对比 |

### 3.3 协议安全

- 传输：支持 `ws://` 与 `wss://`（自签证书或反代终止 TLS）；公网部署必须 wss
- 认证：节点 Token（主节点生成，URL 参数或首帧携带，参考 komari）；Token 可轮换
- 授权：命令白名单（config/reload/status），拒绝任意命令执行（评审 Q10）
- 纵深：主面板 API 继续走 `X-Secret`；集群通道与面板通道分离

## 4. 接口协议草案（v0，供架构评审）

### 4.1 节点 → 主（Agent 上报）

| 方法 | 时机 | 载荷 |
|---|---|---|
| `node.register` | 连接建立后首帧 | 节点 id、版本、主机名、sing-box 版本、能力声明 |
| `node.heartbeat` | 每 5s | 在线状态、sing-box 运行状态、配置 hash、负载（可选） |
| `task.result` | 任务执行完成/失败 | task_id、成功/失败、输出摘要 |

### 4.2 主 → 节点（Hub 下发）

| 方法 | 说明 |
|---|---|
| `task.dispatch` | 下发任务：`{task_id, type, payload}`；type ∈ `get_status / get_config / put_config / reload / restart`（白名单） |
| `hub.hello` | 握手响应：节点配置确认、能力协商 |

### 4.3 配置下发语义

- `put_config`：携带完整 sing-box 配置 JSON；节点侧走**现有校验管线**（严格解码 → box.New 干跑）→ 通过后落盘（含 .bak）→ 按节点 reload 配置执行重载（systemd/rc-service/OpenWrt 自动适配已具备）
- 下发前主节点可 `get_config` + 比对 hash，发现本地漂移则要求确认（P1）
- 结果经 `task.result` 回传，主面板展示成功/失败原因

## 5. 否决项记录

- **gRPC 主通道**：功能满足但引入 protobuf 编解码 + 双向流管理，相对 JSON-RPC 收益小；前端虽有 connect-web（用于 sing-box service API 透传），但集群通道保持简单
- **MQTT / 消息中间件**：外部依赖与部署复杂度，违背单二进制原则
- **从节点独立 Agent 二进制**：与 controller 同进程集成（运行时可禁用 webui），减少运维面
- **主节点做流量中继**：控制面/数据面分离是硬边界
