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
- **`webhook_url`**: 状态变化时接收 JSON `POST` 的 HTTP/HTTPS 地址。事件名为 `node.unavailable` / `node.available`。
- **`webhook_token`**: 通用 Webhook 的可选 Bearer Token，通过 `Authorization: Bearer <token>` 发送。URL 域名包含 `feishu` 时不发送此 Header。
- **`webhook_sign_secret`**: URL 域名包含 `feishu` 时，按飞书自定义机器人规则生成 `timestamp` 和 `sign` 字段；WebUI 配置接口不会回传密钥。留空时发送飞书文本消息但不签名。

Webhook 请求体示例：
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
