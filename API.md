# sing-box-webui API 契约 v1

Base URL: `http://127.0.0.1:8080`，所有接口 `Content-Type: application/json`。
可选鉴权：后端带 `-secret` 启动时，请求头需携带 `X-Secret: <secret>`。
错误统一返回：`{"error": "..."}`（HTTP 4xx/5xx）。

## 状态与配置

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/status` | `{running, config_path, inbounds, outbounds, rules}` |
| GET | `/api/config` | 当前完整 sing-box 配置（格式化 JSON） |
| PUT | `/api/config` | 整体替换配置。body = 完整配置 JSON；校验失败返回 400 |
| POST | `/api/reload` | 重新读取磁盘配置并重载实例 |
| POST | `/api/instance/start` | 启动 sing-box 实例 |
| POST | `/api/instance/stop` | 停止实例 |
| GET | `/api/schema` | 完整 JSON Schema（自动生成，前端表单可参考） |
| GET | `/api/types` | `{inbounds: [...], outbounds: [...], endpoints: [...], services: [...]}` 可用类型列表 |

## Outbound CRUD（重点：vless+reality、tuic）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/outbounds` | `{outbounds: [<outbound对象>, ...]}`（每个含 type/tag 及协议字段） |
| POST | `/api/outbounds` | 新建。body = outbound 对象，必须含 `type` 和 `tag` |
| GET | `/api/outbounds/{tag}` | 单个 |
| PUT | `/api/outbounds/{tag}` | 整体替换（body 含 tag，需与路径一致） |
| DELETE | `/api/outbounds/{tag}` | 删除。若被 route 引用返回 400 |

vless outbound 示例：
```json
{
  "type": "vless",
  "tag": "vless-reality",
  "server": "example.com",
  "server_port": 443,
  "uuid": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "flow": "xtls-rprx-vision",
  "network": "tcp",
  "tls": {
    "enabled": true,
    "server_name": "example.com",
    "utls": { "enabled": true, "fingerprint": "chrome" },
    "reality": { "enabled": true, "public_key": "...", "short_id": "" }
  }
}
```

tuic v5 outbound 示例：
```json
{
  "type": "tuic",
  "tag": "tuic-out",
  "server": "example.com",
  "server_port": 443,
  "uuid": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "password": "password",
  "congestion_control": "bbr",
  "udp_relay_mode": "native",
  "zero_rtt_handshake": false,
  "tls": { "enabled": true, "server_name": "example.com", "alpn": ["h3"] }
}
```

## Inbound CRUD（重点：mixed）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/inbounds` | `{inbounds: [...]}` |
| POST | `/api/inbounds` | 新建，必须含 `type` 和 `tag` |
| GET | `/api/inbounds/{tag}` | 单个 |
| PUT | `/api/inbounds/{tag}` | 整体替换 |
| DELETE | `/api/inbounds/{tag}` | 删除。若被 route 引用返回 400 |

mixed inbound 示例：
```json
{
  "type": "mixed",
  "tag": "mixed-in",
  "listen": "127.0.0.1",
  "listen_port": 2080,
  "users": [
    { "username": "user", "password": "pass" }
  ]
}
```

## Route 规则 CRUD（简单规则：inbound → outbound）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/routes` | `{routes: [{id, rule}, ...], final: "<outbound tag>"}` |
| POST | `/api/routes` | 新建。body = 规则 JSON（不含 id） |
| PUT | `/api/routes/{id}` | 替换指定规则 |
| DELETE | `/api/routes/{id}` | 删除 |

规则示例（sing-box 简单规则，重点形态）：
```json
{ "inbound": ["mixed-in"], "outbound": "vless-reality" }
```
其他可用匹配字段：`network`（"tcp"/"udp"）、`domain`、`domain_suffix`、`ip_cidr`、`port` 等，均为字符串数组；`outbound` 为单个字符串。

## 工具

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/tools/uuid` | `{uuid}` 生成 v4 uuid |
| POST | `/api/tools/reality-keypair` | `{private_key, public_key}` 生成 Reality X25519 密钥对 |

## 写操作响应

成功：`{"saved": true, "reloaded": true|false}`；
若配置已保存但实例重载失败：200 + `{"saved": true, "reload_error": "...", "message": "..."}`。
