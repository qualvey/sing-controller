# sing-box-controller API 契约 v2

Base URL: `http://127.0.0.1:8080`，所有接口 `Content-Type: application/json`。
可选鉴权：controller 带 `-secret` 启动时，请求头需携带 `X-Secret: <secret>`。
错误统一返回：`{"error": "..."}`（HTTP 4xx/5xx）。

## 状态与配置

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/status` | `{config_path, controller_config, min_port, defaults{...}, inbounds, outbounds, rules}` |
| GET | `/api/config` | 当前 sing-box 主配置（格式化 JSON） |
| PUT | `/api/config` | 整体替换主配置。body = 完整配置 JSON；校验失败返回 400 |
| GET | `/api/schema` | 完整 JSON Schema（自动生成） |
| GET | `/api/types` | `{inbounds: [...], outbounds: [...], endpoints: [...], services: [...]}` |

## Controller 设置

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/settings` | controller 配置：`{config, min_port, defaults{inbound_type, outbound_type, listen, listen_port}}` |
| PUT | `/api/settings` | 更新。`config` 为 sing-box 主配置路径（相对 controller 工作目录）；变更路径立即切换，新路径不存在则自动生成骨架 |

## 端口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/ports/available?start=N` | 返回 `{port, start}`：从 start（默认 controller 的 min_port）起第一个可 bind 的 TCP 端口；N 需 ≥1024 |

## Outbound CRUD（vless+reality / tuic v5 重点）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/outbounds` | `{outbounds: [<outbound对象>, ...]}` |
| POST | `/api/outbounds` | 新建。body 必须含 `type` 和 `tag` |
| GET | `/api/outbounds/{tag}` | 单个 |
| PUT | `/api/outbounds/{tag}` | 整体替换 |
| DELETE | `/api/outbounds/{tag}` | 删除；被 route 引用时返回 400 |

vless 示例：
```json
{
  "type": "vless", "tag": "vless-reality", "server": "example.com", "server_port": 443,
  "uuid": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", "flow": "xtls-rprx-vision", "network": "tcp",
  "tls": {
    "enabled": true, "server_name": "example.com",
    "utls": { "enabled": true, "fingerprint": "chrome" },
    "reality": { "enabled": true, "public_key": "...", "short_id": "" }
  }
}
```
tuic v5 示例：
```json
{
  "type": "tuic", "tag": "tuic-out", "server": "example.com", "server_port": 443,
  "uuid": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", "password": "***",
  "congestion_control": "bbr", "udp_relay_mode": "native", "zero_rtt_handshake": false,
  "tls": { "enabled": true, "server_name": "example.com", "alpn": ["h3"] }
}
```

## Inbound CRUD（mixed 重点）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/inbounds` | `{inbounds: [...]}` |
| POST | `/api/inbounds` | 新建，必须含 `type` 和 `tag` |
| GET | `/api/inbounds/{tag}` | 单个 |
| PUT | `/api/inbounds/{tag}` | 整体替换 |
| DELETE | `/api/inbounds/{tag}` | 删除；被 route 引用时返回 400 |

mixed 示例：
```json
{
  "type": "mixed", "tag": "mixed-in", "listen": "127.0.0.1", "listen_port": 2080,
  "users": [{ "username": "user", "password": "***" }]
}
```

## Route 规则 CRUD

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/routes` | `{routes: [{id, rule}, ...], final: "<outbound tag>"}` |
| POST | `/api/routes` | 新建。body = 规则 JSON（不含 id） |
| PUT | `/api/routes/{id}` | 替换指定规则 |
| DELETE | `/api/routes/{id}` | 删除 |

规则示例（简单规则）：
```json
{ "inbound": ["mixed-in"], "outbound": "vless-reality" }
```
匹配字段均为字符串数组（`network`/`domain`/`domain_suffix`/`ip_cidr`/`port` 等），`outbound` 为单个字符串；
注意 sing-box `Listable` 单值序列化为字符串（如 `"inbound": "mixed-in"`），前端回填需兼容。

## 工具

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/tools/uuid` | `{uuid}` 生成 v4 uuid |
| POST | `/api/tools/reality-keypair` | `{private_key, public_key}` Reality X25519 密钥对（URL-safe base64） |
| POST | `/api/tools/parse-json` | body `{json: "<文本>"}`；合法 → `{ok:true, data:<解析结果>}`，非法 → 400 |

## 写操作响应

成功：`{"saved": true}`；配置保存均经过 sing-box 完整校验（严格解码 + box.New 干跑 + 原子写盘）。
