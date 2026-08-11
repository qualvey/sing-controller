# 集群架构设计：一主多从（sing-box-webui）

> 状态：评审稿 v0.1（未实现） · 日期：2026-08-10
> 相关文档：[需求评审](./requirement-review.md) · [技术栈与协议选型](./tech-stack-protocol.md)

## 1. 总体拓扑

```
                        ┌─────────────────────────────────────────┐
                        │  主节点 (Master)                          │
                        │                                         │
  管理员 ── HTTPS ──▶  │  WebUI（面板，NodesView 等）               │
                        │  REST API（现有 /api/*，X-Secret 鉴权）    │
                        │  ┌─────────────────────────────────────┐ │
                        │  │ Cluster Hub（WS 服务端 :25774）       │ │
                        │  │  会话注册表 / 心跳 / 任务调度 / Token   │ │
                        │  └─────────────────────────────────────┘ │
                        │  节点注册信息 cluster.json                 │
                        └──────────────┬──────────────────────────┘
                    WS(wss) 出站连接 ───┼─── Token 认证（NAT 友好）
        ┌───────────────┬──────────────┴───────────────┐
        ▼               ▼                              ▼
┌───────────────┐ ┌───────────────┐            ┌───────────────┐
│ 从节点 A       │ │ 从节点 B       │    ...     │ 从节点 N       │
│ (NAT 后)       │ │ (公网)         │            │               │
│ Controller    │ │ Controller    │            │ Controller    │
│  ├ Agent(WS)  │ │  ├ Agent(WS)  │            │  ├ Agent(WS)  │
│  ├ 配置校验管线 │ │  ├ 配置校验管线 │            │  ├ 配置校验管线 │
│  └ reload 适配 │ │  └ reload 适配 │            │  └ reload 适配 │
│        │       │        │       │            │        │       │
│   sing-box 实例 │   sing-box 实例 │            │   sing-box 实例 │
└───────────────┘ └───────────────┘            └───────────────┘
```

要点：
- **控制面/数据面分离**：master 只做配置与状态管理；流量在 sing-box 节点间直连，master 永不中转
- **从节点主动出站**：NAT 后无需任何入站端口
- **单二进制**：主/从都是同一个 sing-controller，按配置决定角色（`cluster.mode: master|node|off`）
- **从节点无 webui**：`-webui=false`（运行时禁用，MVP 先不做编译裁剪）

## 2. 角色与部署形态

| 角色 | 部署内容 | 说明 |
|---|---|---|
| master | 面板 + REST + Cluster Hub + 本地（可选）sing-box | 需要公网可达端口（Hub + 面板端口） |
| node | Controller 核心（store/settings/reload）+ Agent | 无 webui；REST 可监听仅本地（安全） |

同一二进制，启动参数/配置文件区分：

```jsonc
// 主节点 controller 配置（config.json 增量）
{
  "cluster": {
    "mode": "master",
    "hub_listen": "0.0.0.0:25774",
    "token_file": "./cluster-tokens.json",   // 节点 Token 注册表（自动生成/管理）
    "tls": { "enabled": false }               // wss：自签证书或反代终止
  }
}

// 从节点 controller 配置（增量）
{
  "cluster": {
    "mode": "node",
    "master_url": "wss://panel.example.com:25774/cluster/v2/rpc",
    "token": "xxxxxxxx-xxxx-...",             // 主节点预生成
    "node_id": "hk-01",                       // 可选，缺省取主机名
    "heartbeat_interval": "5s"
  }
}
```

无 `cluster` 段 = 纯单机模式（现状，零回归）。

## 3. 模块设计（新增 `cluster/` 包，与现有代码物理隔离）

```
controller/
├── internal/
│   ├── cluster/
│   │   ├── protocol.go        # JSON-RPC 2.0 消息类型、方法常量（node.*/hub.*/task.*）
│   │   ├── hub.go             # master：WS 服务端、会话注册表、心跳超时、Token 校验
│   │   ├── hub_task.go        # master：任务调度（分发/超时/重试/结果回调）
│   │   ├── agent.go           # node：WS 客户端、注册、心跳、指数退避重连、HTTP 降级
│   │   ├── agent_task.go      # node：命令执行器（白名单分派到现有能力）
│   │   └── store.go           # 节点注册信息持久化（cluster.json / tokens）
│   ├── api/
│   │   ├── cluster.go         # master 面板侧 REST：节点 CRUD / 状态 / 下发 / 重载 / Token 管理
│   │   └── agent_hooks.go     # node 侧：把现有 handler 能力暴露给命令执行器（接口化）
│   └── settings/              # 增量：cluster 段配置
└── web/src/
    ├── views/NodesView.vue    # 节点列表（在线/离线、hash 同步状态、一键重载）
    ├── views/NodeDetailView.vue # 节点详情 + 配置编辑/下发（复用 Config 编辑组件）
    └── api/cluster.ts         # 主面板 REST 客户端 + 可选 WS 推送
```

### 3.1 命令执行器（node 侧关键设计）

**不重写业务逻辑**：现有 `internal/api` 的 handler 已封装全部能力（读/写配置、校验管线、重载、诊断、用户池…）。Agent 执行器做**薄适配**：

```go
// agent_task.go（伪代码，仅表达设计）
func (a *Agent) dispatch(task cluster.Task) error {
    switch task.Type {
    case "get_status":
        return a.reply(task, collectStatus())          // 复用 handleStatus 的数据源
    case "get_config":
        return a.reply(task, a.handler.GetConfig(ctx)) // 复用 store.Content
    case "put_config":
        // 复用完整校验管线：store.RawSave / store.Update + reload 自动适配
        if err := a.handler.PutConfig(ctx, task.Payload); err != nil { return a.replyError(task, err) }
        return a.reply(task, a.handler.Reload(ctx))
    case "reload":
        return a.reply(task, a.handler.Reload(ctx))    // 复用 reloadNow（auto 适配已具备）
    default:
        return a.replyError(task, errUnknownTask)
    }
}
```

安全：白名单分发；**不提供** shell/任意命令（需求评审 Q10）。

### 3.2 Hub 会话模型（master 侧）

```
NodeSession {
    id, token, name, version
    conn      *ws.Conn          # 唯一活跃连接（同一节点新连踢旧连，参考 komari）
    lastSeen  time.Time         # 心跳超时（3×interval 判离线）
    status    online|offline    # 由 Hub 维护，面板只读
    pending   map[string]Task   # 未确认/进行中的任务（超时重发）
    configHash string           # 最近上报的配置 hash（面板显示同步状态）
}
```

- 注册流程：握手带 token → 校验（对照 token 注册表）→ `node.register` → Hub 回 `hub.hello` → 进入心跳循环
- 断连：会话标记离线；Agent 指数退避重连（1s→2s→4s…上限 60s，参考 komari）
- 任务：`task.dispatch` → 等待 `task.result`（超时 30s 重试 2 次）→ 回调更新面板

## 4. 数据模型

```jsonc
// 主节点 cluster.json（注册表）
{
  "nodes": [
    {
      "id": "hk-01",
      "token": "uuid",            // 唯一，可轮换
      "name": "香港-01",
      "tags": ["hk", "relay"],    // P1 分组
      "created_at": "...",
      "last_seen": "...",
      "last_status": { "singbox_running": true, "config_hash": "sha256:...", "version": "..." }
    }
  ]
}
```

配置同步模型（评审 Q1 结论）：
- **从节点配置本地化持久化**（现状不变），主节点下发 = 显式覆盖（写盘前走现有校验管线 + .bak）
- 主节点**不常驻持有**从节点配置副本；面板查看时实时 `get_config`（或 P1 缓存 + hash 比对）
- 漂移检测：面板显示「本地 hash ≠ 上次下发 hash」时提示确认，防静默覆盖（P1）

## 5. 关键流程

### 5.1 从节点注册与上线
1. 主节点面板「添加节点」→ 生成 token 与节点 id → 展示部署命令（含 wss 地址 + token）
2. 从节点配置 cluster 段后启动 → Agent 出站连接 → 握手校验 token → `node.register` 上报基本信息
3. Hub 更新注册表 → 面板节点变「在线」→ 进入心跳

### 5.2 配置下发 + 重载（核心闭环）
1. 面板打开节点详情 → 实时 `get_config`（走 Agent 通道）→ 复用现有 Config 编辑组件编辑
2. 保存 → `task.dispatch(put_config, 完整配置)` → Agent 校验管线 → 落盘 → reload 自动适配执行 → `task.result`
3. 面板展示结果；失败展示校验/重载错误原文（与单机体验一致）

### 5.3 断连与恢复
- Agent 断连 → 本地服务不受影响（配置已在盘上）→ 指数退避重连
- 重连成功 → 补报状态（heartbeat 立即一次）→ 面板恢复在线
- 主节点重启 → 会话全断，Agent 自动重连；注册表持久化不丢失

## 6. 安全模型

| 层 | 机制 |
|---|---|
| 传输 | wss（自签证书/反代）；内网可 ws |
| 通道认证 | 节点 Token（随机 UUID，面板生成，支持轮换）；握手即校验 |
| 命令授权 | 白名单类型（get_status/get_config/put_config/reload）；拒绝任意命令 |
| 面板 API | 现有 X-Secret 不变；集群管理端点并入同一鉴权 |
| 数据保护 | 配置下发经校验管线（非法配置不落盘）；.bak 保留 |

## 7. 演进路径

| 阶段 | 范围 | 验收 |
|---|---|---|
| **P0（MVP）** | 单从节点闭环：token 注册、心跳、面板看状态、配置下发+重载、`-webui=false` | 需求评审 §7 验收 1-5 |
| **P1** | 多节点/分组/批量、hash 漂移确认、service API 透传、审计、Token 轮换 UI、gzip | 50 节点稳定性测试 |
| **P2** | 模板变量、计划任务、Web Terminal（安全评审）、多主 | 按需 |

## 8. 兼容性与回归

- 无 `cluster` 配置 = 单机模式，行为与 v0.10.0 完全一致（集群代码不进入单机热路径）
- 从节点复用现有 store/settings/reload 全部能力；`internal/api` 仅新增文件、不改既有 handler 签名（执行器通过新增的薄接口调用）
- 单测守门：注册/心跳/任务/重连核心路径 + 现有 111 用例零回归

## 9. 待评审的开放项

1. Hub 端口与协议端点命名（`/cluster/v2/rpc` vs 自定义）——对齐 komari 便于对照实现
2. 主节点是否内置轻量用户/权限（Q5）还是继续单管理员
3. 心跳间隔与超时判离线阈值（默认 5s/15s，按规模调）
4. 配置下发是「整包替换」还是「段级合并」（MVP 整包，P1 评估段级）
