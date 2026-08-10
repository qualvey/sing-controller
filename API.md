# sing-box-controller API 契约 v3

Base URL: `http://127.0.0.1:8080`（由 controller 配置 `listen` 决定，页面与 API 同端口）。
所有接口 `Content-Type: application/json`（除页面与 healthz）。
可选鉴权：controller 带 `-secret` 启动时，请求头需携带 `X-Secret: <secret>`（或 `?token=`）。
错误统一返回：`{"error": "..."}`（HTTP 4xx/5xx）。

## 页面与健康检查

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/` | webui 页面（嵌入的 SPA，含前端路由 fallback）。`web/dist` 未构建时返回纯文本提示（API-only 模式） |
| GET | `/healthz` | 健康检查 `{"status":"ok"}`（systemd/监控探活） |

静态资源：`/assets/*`（带 hash）返回 `Cache-Control: public, max-age=31536000, immutable`；`index.html` 不缓存。

## 状态与配置

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/status` | `{config_path, controller_config, listen, log_level, min_port, defaults{...}, inbounds, outbounds, rules}` |
| GET | `/api/config` | 当前 sing-box 主配置（格式化 JSON） |
| PUT | `/api/config` | 整体替换主配置。body = 完整配置 JSON；校验失败返回 400 且不落盘 |
| GET | `/api/schema` | 完整 JSON Schema（自动生成，约 440KB，与代码同步） |
| GET | `/api/types` | `{inbounds: [...], outbounds: [...], endpoints: [...], services: [...]}` 可用类型列表 |

## Controller 设置

```jsonc
{
  "config": "./sing-box-config.json",   // sing-box 主配置路径（相对 controller 工作目录）
  "listen": "127.0.0.1:8080",           // HTTP 监听地址；修改后需重启生效
  "log": { "level": "info" },           // trace/debug/info/warn/error/fatal/panic
  "min_port": 8000,                     // 自动分配端口起点
  "defaults": { "inbound_type": "mixed", "outbound_type": "vless", "listen": "127.0.0.1", "listen_port": 2080 }
}
```

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/settings` | 当前 controller 配置（全量） |
| PUT | `/api/settings` | 全量替换。校验：config 非空、listen 为 host:port、log.level 合法、min_port 1024-65535 |

PUT 响应语义：
- `{"saved": true}` 正常
- `{"saved": true, "warning": "..."}` 已保存，但有警告（如 listen 为特权端口 <1024）
- `{"saved": true, "load_error": "...", "message": "..."}` 已保存，但 `config` 路径切换后新主配置加载失败

## 端口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/ports/available?start=N` | 返回 `{port, start}`：从 start（默认 min_port）起第一个可 bind 的 TCP 端口；N 需 ≥1024 |

## Outbound CRUD（vless+reality / tuic v5 重点）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/outbounds` | `{outbounds: [<outbound对象>, ...]}` |
| POST | `/api/outbounds` | 新建。body 必须含 `type` 和 `tag`；tag 重复返回 400 |
| GET | `/api/outbounds/{tag}` | 单个 |
| PUT | `/api/outbounds/{tag}` | 整体替换（body 的 tag 需与路径一致） |
| DELETE | `/api/outbounds/{tag}` | 删除；被 route 规则/final 引用时返回 400；被 selector/urltest 组引用时返回 **409** `{error, references: [组tag,...]}`，加 `?force=true` 重试会自动从所有引用组拔除该 tag 后删除 |

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
| POST | `/api/inbounds` | 新建，必须含 `type` 和 `tag`；tag 重复返回 400 |
| GET | `/api/inbounds/{tag}` | 单个 |
| PUT | `/api/inbounds/{tag}` | 整体替换 |
| DELETE | `/api/inbounds/{tag}` | 删除；被 route 规则引用时返回 400 |

mixed 示例：
```json
{
  "type": "mixed", "tag": "mixed-in", "listen": "127.0.0.1", "listen_port": 2080,
  "users": [{ "username": "user", "password": "***" }]
}
```

## Route 规则 CRUD

规则无 tag/id 字段（sing-box 严格解码禁止自定义字段），id 由旁车 meta（`config.json.meta`）维护，映射规则数组下标。

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/routes` | `{routes: [{id, rule}, ...], final: "<outbound tag>"}` |
| POST | `/api/routes` | 新建。body = 规则 JSON（不含 id）；响应无 id，需重新 GET |
| PUT | `/api/routes/{id}` | 替换指定规则 |
| DELETE | `/api/routes/{id}` | 删除 |

规则示例（简单规则）：
```json
{ "inbound": ["mixed-in"], "outbound": "vless-reality", "network": "tcp" }
```
- 匹配字段均为字符串数组（`inbound`/`network`/`domain`/`domain_suffix`/`ip_cidr`/`port` 等），`outbound` 为单个字符串
- **注意**：sing-box `Listable` 单值序列化为字符串（如 `"inbound": "mixed-in"` 而非数组），前端回填需兼容
- 外部手工改配置后 meta 数量不匹配会自动重新生成 id（旧 id 失效）

## 工具

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/tools/uuid` | `{uuid}` 生成 v4 uuid |
| POST | `/api/tools/reality-keypair` | `{private_key, public_key}` Reality X25519 密钥对（URL-safe base64，与 `sing-box generate reality-keypair` 一致） |
| POST | `/api/tools/parse-json` | body `{json: "<文本>"}`；合法 → `{ok: true, data: <解析结果>}`，非法 → 400 |

## 写操作响应与校验

- 成功：`{"saved": true}`
- 所有写操作（PUT/POST/DELETE）执行统一校验管线：
  **严格解码（未知字段/多态/重复 tag）→ `box.New` 干跑预检 → 原子写盘（`.bak` 备份）**
- 校验失败：400 + `{"error": "..."}`，**不落盘、内存回滚**
- 删除 outbound/inbound 时检查 route 引用（`route.final` + 规则 `outbound`/`inbound` 字段），被引用则 400
