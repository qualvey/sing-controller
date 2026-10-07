# sing-controller 配置说明

`sing-controller` 的配置采用声明式模块化设计，语法和规范与 `sing-box` 保持高度同构。

默认配置文件路径：
- Linux / OpenWrt：`/etc/sing-controller/config.json`
- Windows：`controller.json`

---

## 完整配置规范示例

```json
{
  "$schema": "https://github.com/qualvey/sing-controller/schema.json",

  "server": {
    "listen": "::",
    "listen_port": 8080,
    "secret": ""
  },

  "log": {
    "level": "info",
    "output": "stderr",
    "outfile": ""
  },

  "core": {
    "config_path": "/etc/sing-box/config.json",
    "managed": false,
    "binary_path": "",
    "reload": {
      "mode": "auto",
      "service": "sing-box",
      "pid_file": "/var/run/sing-box.pid",
      "hook": "/usr/local/bin/reload-sing-box.sh",
      "after_save": false
    }
  },

  "api": {
    "clash": {
      "address": "http://127.0.0.1:9090",
      "secret": ""
    },
    "service": {
      "address": "http://127.0.0.1:9900",
      "secret": ""
    },
    "v2ray": {
      "address": "127.0.0.1:10085"
    }
  },

  "stats": {
    "poll_interval": "10s",
    "pattern": "",
    "reset_on_query": false,
    "real_time": {
      "enabled": false,
      "interval_ms": 1000,
      "online_threshold_sec": 60
    }
  },

  "database": {
    "dsn": "postgres://sing:secret@127.0.0.1:5432/sing_controller?sslmode=disable"
  },

  "node_probe": {
    "enabled": true,
    "interval": "120s",
    "test_url": "https://www.gstatic.com/generate_204",
    "timeout": "10s",
    "history_retention_days": 30,
    "webhooks": [
      {
        "url": "https://example.com/hooks/sing-node",
        "secret": "change-me"
      }
    ]
  }
}
```

---

## 模块字段详解

### 1. `server`（服务自身网络与安全）
Controller 自身 WebUI 与 REST API 监听端点。

| 字段 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `listen` | string | `"::"` | 监听 IP 地址。`"::"` 监听双栈所有网卡，`"127.0.0.1"` 仅本机访问 |
| `listen_port` | uint16 | `8080` | HTTP 监听端口 |
| `secret` | string | `""` | 可选 API 鉴权 Token。若设置，客户端需在请求头携带 `X-Secret` |

---

### 2. `log`（Controller 自身日志）
配置风格与 `sing-box` 的 `log` 段保持一致。

| 字段 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `level` | string | `"info"` | 日志级别：`trace` \| `debug` \| `info` \| `warn` \| `error` |
| `output` | string | `"stderr"` | 日志输出目标：`stderr` \| `stdout` \| `file` |
| `outfile` | string | `""` | 日志文件路径。当 `output` 为 `file` 或指定了文件路径时落盘写入 |

---

### 3. `core`（sing-box 内核与生命周期管理）
负责管理受控的 sing-box 配置文件及服务启停/热重载。

| 字段 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `config_path` | string | `/etc/sing-box/config.json`（Windows 为 `sing-box.json`） | sing-box 主配置文件路径 |
| `managed` | bool | Windows 为 `true`，Linux 为 `false` | 是否由 Controller Supervisor 托管启动 sing-box 子进程 |
| `binary_path` | string | `""` | `managed` 为 true 时指定 sing-box 二进制路径（留空则自动查找） |
| `reload.mode` | string | `"auto"` | 重载模式：`auto` \| `systemd` \| `procd` \| `rc-service` \| `pidfile` \| `hook` \| `none` |
| `reload.service` | string | `"sing-box"` | 系统服务名称（用于 systemd / procd / openrc） |
| `reload.pid_file` | string | `""` | PID 文件路径（用于 pidfile 模式） |
| `reload.hook` | string | `""` | 自定义重载命令（hook 模式） |
| `reload.after_save` | bool | `false` | WebUI 保存配置后是否自动触发重载 |

---

### 4. `api`（内核上游 API 通信与反向代理）
WebUI 页面及 Controller 内部反向代理到 sing-box 的通信端点。

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `clash.address` | string | sing-box 的 `experimental.clash_api.external_controller` 地址 |
| `clash.secret` | string | clash API 访问密钥（若配置，代理时自动注入 Bearer Token） |
| `service.address` | string | sing-box 的 `services[type=api]` gRPC-Web 端点 |
| `service.secret` | string | service API 访问密钥 |
| `v2ray.address` | string | sing-box 的 v2ray stats gRPC 统计端口（通常为 `127.0.0.1:10085`） |

> **自动推断机制**：若 `api` 段对应端点留空，Controller 启动时会自动尝试从 `core.config_path` 的 sing-box 配置中推断并自适应连接，无需手动重复配置。

---

### 5. `stats`（流量统计与实时监控）
控制后台 Collector 对流量和用户连接的数据统计行为。

| 字段　　　　　　　　　　　　　　 | 类型　 | 默认值　| 说明　　　　　　　　　　　　　　　　　　　　　　　　　　　　　 |
| :---------------------------------| :-------| :--------| :---------------------------------------------------------------|
| `poll_interval`　　　　　　　　　| string | `"10s"` | 从 sing-box v2ray stats 采集数据的轮询周期（Go duration 格式） |
| `pattern`　　　　　　　　　　　　| string | `""`　　| 流量统计规则匹配过滤正则　　　　　　　　　　　　　　　　　　　 |
| `reset_on_query`　　　　　　　　 | bool　 | `false` | 查询时是否通知内核重置计数器　　　　　　　　　　　　　　　　　 |
| `real_time.enabled`　　　　　　　| bool　 | `false` | 是否开启实时 WebSocket 流量推送广播　　　　　　　　　　　　　　|
| `real_time.interval_ms`　　　　　| int　　| `1000`　| 实时广播频率（毫秒）　　　　　　　　　　　　　　　　　　　　　 |
| `real_time.online_threshold_sec` | int　　| `60`　　| 判定用户/节点在线的活跃超时时间（秒）　　　　　　　　　　　　　|

---

### 6. `database`（PostgreSQL 持久化，可选）

节点在线状态、探测历史与 Webhook 投递记录都落库。**该段留空时 Controller 不会初始化数据库**，
流量统计（Collector）与节点探测一并保持停用，其余功能不受影响。

| 字段 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `dsn` | string | `""` | PostgreSQL DSN。留空表示禁用数据库；也可用环境变量 `SING_CONTROLLER_DATABASE_DSN` 提供 |

> 启动时会自动迁移（`AutoMigrate`）`node_probe_statuses`、`node_probe_records`、`webhook_deliveries` 等表。
> `GET /api/settings` 不会返回 DSN（响应中该字段为空），以免泄露数据库口令。

---

### 7. `node_probe`（节点在线状态与延迟探测）

周期性对 sing-box 的叶子出站节点做连通性/延迟探测，结果写入数据库，并在状态发生变化时通过 Webhook 通知。
仅在 `database.dsn` 非空时生效；`node_probe.enabled` 可单独关闭探测而不影响历史查询。

| 字段 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `enabled` | bool | `true` | 是否启用周期性探测 |
| `interval` | string | `"120s"` | 探测周期（Go duration）。非法值回退为 `120s` |
| `test_url` | string | `"https://www.gstatic.com/generate_204"` | 探测使用的 URL |
| `timeout` | string | `"10s"` | 单次探测超时（Go duration），同时作为 Clash 侧 `timeout` 参数 |
| `history_retention_days` | int | `30` | 探测历史保留天数，`<= 0` 视为 `30`；历史清理按小时节流并分批删除 |
| `webhooks` | array | `[]` | 状态变化通知端点列表 |
| `webhooks[].url` | string | `""` | 接收端 URL，仅接受 `http` / `https` |
| `webhooks[].secret` | string | `""` | 可选密钥，以 `X-Secret` 请求头发送 |

> **探测后端优先级**：`api.service`（sing-box `services[type=api]` gRPC-Web `URLTest`）→ `api.clash`
> （`GET /proxies/{tag}/delay`）。两者都未配置时本轮结果为 `unknown`，并不会把节点误判为离线。
> Clash 侧采用 16 并发，整轮时间上限 = `timeout × 波次数`（封顶 5 分钟），
> 因此单节点超时不会线性拉长一轮探测，也不会让后续波次的健康节点被误判离线。
> 超出整轮预算、没轮到探测的节点记为 `unknown`（本轮没测到），而不是 `offline`。
> Clash API 有响应但所有节点都拿不到延迟（地址指向别的内核、tag 失配、不提供 `/delay`）时，
> 本轮判为后端故障并回退到 gRPC 结果，而不是把全部节点写成离线。
> gRPC 侧以首帧 URLTest 历史为基线做差分，避免把上一轮遗留的延迟当成本轮结果；
> sing-box 的 `URLTest` 是 fire-and-forget 并发执行，整轮上限 = `timeout` + 广播余量。
> 整轮超时只会把「本轮确实没跑完」的节点判为 `offline`，已知结果仍会正常落库并返回（HTTP 502 + 部分结果）。
>
> **探测范围**：只探测叶子节点，自动跳过 `direct` / `block` / `dns` / `selector` / `urltest` / `fallback` /
> `loadbalance` 等组类型，避免后台 `URLTest` 触发组内选择变化。
>
> **状态集合**：`online`（探测成功，带 `latency_ms`）、`offline`（探测失败或超时）、`degraded`、`unknown`。
>
> **Webhook 载荷**（状态变化时才发送，失败按指数退避重试，最多 5 次）：
> ```json
> {
>   "event": "node.status_changed",
>   "source": "sing-controller",
>   "occurred_at": "2026-03-01T10:00:00Z",
>   "node":     { "node_tag": "jp-01", "node_type": "vless" },
>   "previous": { "node_tag": "jp-01", "node_type": "vless", "status": "online", "latency_ms": 128 },
>   "current":  { "node_tag": "jp-01", "node_type": "vless", "status": "offline", "error": "..." }
> }
> ```
> 请求头包含 `X-Sing-Controller-Event`、`X-Sing-Controller-Event-ID` 与可选的 `X-Secret`。