# Controller 配置手册 (controller.json)

`sing-controller` 本身的服务配置文件默认为：
- **Windows**: `controller.json`
- **Linux / systemd / OpenWrt**: `/etc/sing-controller/config.json`

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
    "working_dir": "",
    "reload": {
      "mode": "auto",
      "service": "sing-box",
      "pid_file": "/var/run/sing-box.pid",
      "hook": "/usr/local/bin/reload-sing-box.sh",
      "after_save": false
    }
  },

  "api": {
    "service": {
      "address": "http://127.0.0.1:9900",
      "secret": ""
    },
    "clash": {
      "address": "http://127.0.0.1:9090",
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

  "node_health": {
    "enabled": false,
    "interval": "5m",
    "test_url": "https://www.gstatic.com/generate_204",
    "timeout_ms": 5000,
    "failure_threshold": 2,
    "concurrency": 5,
    "webhook_url": "",
    "webhook_token": "",
    "webhook_sign_secret": ""
  },

  "database": {
    "dsn": ""
  },

  "node_probe": {
    "enabled": true,
    "interval": "120s",
    "test_url": "https://www.gstatic.com/generate_204",
    "timeout": "10s",
    "history_retention_days": 30,
    "webhooks": []
  }
}
```

---

## 字段详解

### 1. `server` (WebUI 与 HTTP API 监听)
- **`listen`**: 监听网络地址，如 `"::"`（支持 IPv4/IPv6 双栈）或 `"127.0.0.1"`（限制本地访问）。
- **`listen_port`**: 监听端口，默认 `8080`。
- **`secret`**: 可选 API 鉴权密钥。若非空，所有 RESTful API 请求必须携带 `X-Secret: <token>` 头。

### 2. `core` (sing-box 内核生命周期)
- **`config_path`**: 托管或管理的 `sing-box` 代理主配置文件绝对路径。
- **`managed`**: 是否由 Controller Supervisor 自动拉起与管理 sing-box 子进程（Windows 下默认 `true`，Linux 推荐由 systemd 独立管理因此默认 `false`）。
- **`binary_path`**: 指定 sing-box 可执行文件路径，为空时自动在 PATH 与常见系统目录探测。
- **`reload`**: 重载行为配置。`mode: "auto"` 支持自动适配平台环境；`after_save: true` 可以在每次保存新配置后立即无感重载内核。

### 3. `api` (上游代理内核连接)
- **`service`**: sing-box 原生服务 API 地址（如 `services[type=api]`），作为优先核心通道。
- **`clash`**: Clash External Controller 地址，作为兼容补充。
- **`v2ray`**: V2Ray Stats API 端点。

### 4. `stats` (流量与统计采集)
- **`poll_interval`**: 轮询流量聚合统计时间间隔（如 `"10s"`、`"5s"`）。
- **`real_time`**: 实时统计 WebSocket 广播器参数，支持配置推送毫秒频率与在线判定阈值。

### 5. `node_health` (节点可用性自动探测)
- **`enabled`**: 是否启用后台周期探测；关闭 WebUI 后仍由 Controller 执行。
- **`interval`**: 探测周期，例如 `"5m"`；最小值为 `30s`。
- **`test_url`**: Clash API 通道使用的 HTTP/HTTPS 探测地址。
- **`timeout_ms`**: 单节点探测超时，允许 `500` 到 `60000` 毫秒。
- **`failure_threshold`**: 连续失败多少次后将节点标记为不可用；成功探测后立即恢复。
- **`concurrency`**: 最大并发探测数，范围 `1` 到 `32`。
- **`webhook_url`**: 状态变化时接收 JSON `POST` 的 HTTP/HTTPS 地址。事件名为 `node.unavailable` / `node.available`；同一轮探测中多个节点的状态变化会合并为一次通知。
- **`webhook_token`**: 通用 Webhook 的可选 Bearer Token，通过 `Authorization: Bearer <token>` 发送。URL 域名包含 `feishu` 时不发送此 Header。
- **`webhook_sign_secret`**: URL 域名包含 `feishu` 时，按飞书自定义机器人规则生成 `timestamp` 和 `sign` 字段；WebUI 配置接口不会回传密钥。留空时发送飞书文本消息但不签名。

Webhook 请求体示例：
单个节点变化时：

```json
{
  "event": "node.unavailable",
  "node": {
    "tag": "proxy-hk-01",
    "type": "vless",
    "state": "unavailable",
    "consecutive_failures": 2,
    "checked_at": "2026-09-30T12:00:00Z",
    "error": "context deadline exceeded"
  },
  "consecutive_failures": 2,
  "checked_at": "2026-09-30T12:00:00Z"
}
```

同一轮探测有多个节点变化时，通用 Webhook 会发送一次 `POST`，请求体为 `{"events": [...]}`，其中每个元素结构与上面的单节点事件相同；飞书消息则会将多个节点变化合并到同一条文本消息中。

飞书 URL（域名包含 `feishu`）会自动改用飞书文本消息格式：
```json
{
  "timestamp": "1780185600",
  "sign": "Base64(HMAC-SHA256(timestamp + \"\\n\" + secret, empty))",
  "msg_type": "text",
  "content": {
    "text": "节点不可用\n节点: proxy-hk-01\n协议: vless\n状态: unavailable\n连续失败: 2"
  }
}
```

如果飞书机器人开启了签名校验，请在 WebUI 中填写 `webhook_sign_secret`。WebUI 配置接口不会回传 `webhook_token` 或 `webhook_sign_secret`。

---

### 6. `database`（PostgreSQL 持久化，可选）

| 字段 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `dsn` | string | `""` | PostgreSQL DSN；留空时禁用数据库，也可通过 `SING_CONTROLLER_DATABASE_DSN` 环境变量配置 |

节点探测历史、状态和 Webhook 投递记录会写入数据库；数据库未配置时，持久化节点探测与流量采集不可用。`GET /api/settings` 会对 DSN 脱敏。

---

### 7. `node_probe`（持久化节点状态与延迟探测）

此探测器按周期查询 sing-box 的叶子出站节点，并将状态与历史写入数据库。它与上面的 `node_health` 兼容并存：`node_health` 提供轻量内存探测，`node_probe` 提供数据库持久化及历史查询；`node_probe` 需要配置数据库。

| 字段 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `enabled` | bool | `true` | 是否启用周期探测 |
| `interval` | string | `"120s"` | 探测周期（Go duration） |
| `test_url` | string | `"https://www.gstatic.com/generate_204"` | 探测 URL |
| `timeout` | string | `"10s"` | 单次探测超时 |
| `history_retention_days` | int | `30` | 探测历史保留天数 |
| `webhooks` | array | `[]` | 节点状态变化时触发的 Webhook 列表；同一轮探测中多个节点的变化合并为一次通知。
| `webhooks[].url` | string | — | HTTP/HTTPS 接收地址 |
| `webhooks[].secret` | string | — | 可选密钥，通过 `X-Secret` 请求头发送；设置接口会脱敏 |

探测优先使用 `api.service` 的 gRPC-Web URLTest，失败时回退到 `api.clash` 的延迟接口。仅探测叶子节点，组出站会跳过。状态包括 `online`、`offline`、`degraded` 和 `unknown`；手动探测接口为 `POST /api/node-probes`，单节点历史接口为 `GET /api/node-probes/{tag}/history`。
