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
| * | /api/clash/* | **clash API 反向代理**（转发到 sing-box external_controller，自动注入 secret；覆盖 /proxies /connections /logs /traffic /rules /configs /dns/query 等 clash API 全部端点） |

静态资源：`/assets/*`（带 hash）返回 `Cache-Control: public, max-age=31536000, immutable`；`index.html` 不缓存。

## 状态与配置

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/status` | `{config_path, controller_config, listen, log_level, min_port, defaults{...}, inbounds, outbounds, rules}` |
| GET | `/api/config` | 当前 sing-box 主配置（解析后重序列化，注释丢失） |
| PUT | `/api/config` | 整体替换主配置。body = 完整配置 JSON；校验失败返回 400 且不落盘 |
| GET | `/api/config/raw` | **主配置文件原始内容**（保留注释/格式/字段顺序，text/plain） |
| PUT | `/api/config/raw` | **原样保存配置文本**（注释/尾逗号保留；sing-box 解析 + 干跑校验通过后原子写盘，同步内存 Options） |
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

## Inbound CRUD（mixed / shadowsocks / tuic 重点）

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

shadowsocks 示例（webui 表单默认值：method `chacha20-ietf-poly1305`、listen `::`、端口 `23010`、密码自动生成）：
```json
{
  "type": "shadowsocks", "tag": "ss-in", "listen": "::", "listen_port": 23010,
  "method": "chacha20-ietf-poly1305", "password": "8JCsPssfgS8tiRwiMlhARg=="
}
```
- method 枚举：`none / aes-128-gcm / aes-192-gcm / aes-256-gcm / chacha20-ietf-poly1305 / xchacha20-ietf-poly1305 / 2022-blake3-aes-128-gcm / 2022-blake3-aes-256-gcm / 2022-blake3-chacha20-poly1305`（`none` 无需密码）
- 密码建议 16 字节随机 base64（`POST /api/tools/password` 生成）；多用户走用户池绑定（`users[]` 注入 name+password，用户池为唯一真相，内联 users[] 会被投影覆盖）

tuic 示例（webui 表单：TLS 强制 + 证书 Provider，用户走用户池绑定）：
```json
{
  "type": "tuic", "tag": "tuic-in", "listen": "0.0.0.0", "listen_port": 443,
  "congestion_control": "bbr",
  "tls": { "enabled": true, "certificate_provider": "letsencrypt", "server_name": "example.com", "alpn": ["h3"] }
}
```

## 用户池（全局用户，多对多绑定入站）

用户池是「用户」的唯一真相（存于旁车 meta `config.json.meta` 的 `users` 段）：用户不直接写进 inbounds，
而是绑定到若干入站 tag，保存时由后端按入站类型投影注入对应 inbounds 的 `users[]`（覆盖式同步，
内联 users 会被投影覆盖）。

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/users` | `{users: [{name, uuid?, password?, flow?, bound_inbounds: [tag,...]}]}` |
| POST | `/api/users` | 新建。`name` 必填且唯一；`uuid`/`password` 至少填一个；`bound_inbounds` 为绑定入站 tag 列表 |
| PUT | `/api/users/{name}` | 整体替换（不允许重命名；body.name 需与路径一致或省略） |
| DELETE | `/api/users/{name}` | 删除并从所有绑定入站的 users[] 移除 |

- 支持 users[] 的入站类型 → 投影字段：vless(name,uuid,flow) / vmess(name,uuid) / trojan(name,password) / tuic(name,uuid,password) / hysteria2(name,password) / hysteria(name,password) / shadowsocks(name,password) / anytls(name,password) / shadowtls(name,password)
- 新建/编辑/删除用户会自动同步所有关联入站的 users[]（走统一校验管线；绑定时对应入站需已存在）
- 新建用户前端默认自动生成密码（`POST /api/tools/password`，16 字节 base64）

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
- 匹配字段完整覆盖 sing-box `RawDefaultRule`：inbound / ip_version / network / protocol / auth_user / client / domain 系列（domain、domain_suffix、domain_keyword、domain_regex）/ geosite / rule_set / geoip / source_geoip / ip_cidr / source_ip_cidr / ip_is_private / source_ip_is_private / port / port_range / source_port / source_port_range / process 系列 / package 系列 / user / user_id / network_type / wifi_ssid / wifi_bssid / source_mac_address / source_hostname / preferred_by / clash_mode / invert，以及 action 系列（outbound / action / sniffer / server 及 route-options 参数）
  （字段元数据见 `controller/web/src/views/routeFields.ts`，与 sing-box fork 源码 option/rule.go 同步；interface_address 等复杂 map 字段用原始 JSON 兜底）
- `outbound` 为单个字符串；其余匹配字段为字符串/数字数组（Listable）
- **注意**：sing-box `Listable` 单值序列化为字符串（如 `"inbound": "mixed-in"` 而非数组），前端回填需兼容
- 外部手工改配置后 meta 数量不匹配会自动重新生成 id（旧 id 失效）

## DNS 管理

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/dns` | `{servers: [...], rules: [{id, rule}, ...], options: {final, strategy, timeout, disable_cache, independent_cache, reverse_mapping, client_subnet}}` |
| POST | `/api/dns/servers` | 新建 DNS server（多态 transport：local/udp/tcp/tls/https/quic/h3/fakeip/hosts/dhcp/mdns，registry 解码）；tag 必填且唯一 |
| PUT | `/api/dns/servers/{tag}` | 整体替换（body tag 需与路径一致） |
| DELETE | `/api/dns/servers/{tag}` | 被 `dns.final`/规则引用时返回 409 + references；`?force=true` 自动清除引用后删除 |
| POST | `/api/dns/rules` | 新建 DNS 规则（`server` 字段指向 DNS server，旧 `outbound` 字段兼容）；id 存旁车 meta |
| PUT | `/api/dns/rules/{id}` | 替换 |
| DELETE | `/api/dns/rules/{id}` | 删除 |
| PUT | `/api/dns/options` | 部分更新 `{final, strategy, timeout, disable_cache, independent_cache, reverse_mapping}`；final 引用不存在时 400 |

- **注意（sing-box 1.14 testing）**：DNS 规则动作字段是 `server`（不是 route 规则的 `outbound`）；`outbound` 已废弃但兼容解码
- 所有写操作走统一校验管线（box.New 干跑）；DNS 段不存在时自动创建

## 配置诊断

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/diagnostics` | `{diagnostics: [{level: error\|warning\|info, message}]}` 静态分析：重复 tag、route/dns 悬空引用（final/规则/组/rule_set/detour）、inbound 监听冲突、未使用 outbound、缺 route/dns 段警告、资源统计 |

> 说明：sing-box 校验不拦截悬空引用（`box.New` 通过），诊断页是这些问题的唯一发现渠道。

## 规则集（route.rule_set 段）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/rule-sets` | `{rule_sets: [{id, rule_set}, ...]}` |
| POST | `/api/rule-sets` | 新建（inline 内联 / local 本地 / remote 远程，参照 option/rule_set.go）；tag 必填 |
| PUT | `/api/rule-sets/{id}` | 替换 |
| DELETE | `/api/rule-sets/{id}` | 被 route/dns 规则引用时返回 409 + references；`?force=true` 自动从规则中移除引用后删除 |

- inline：`{tag, rules: [HeadlessRule...]}`；local：`{type, tag, format, path}`；remote：`{type, tag, format, url, initial_path?, update_interval?}`
- format 省略时按扩展名推断（.json→source，.srs→binary）；多 tag 时 local/remote 的 path/url 需含 `{tag}` 占位符
- local 校验会实际读取文件（空/非法文件报错）；remote 仅校验 URL 格式

## 证书

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/certificate` | `{certificate: {...}|null, providers: [{id, provider}, ...]}` |
| PUT | `/api/certificate` | 整体替换顶层 certificate 段（store/certificate/certificate_path/certificate_directory_path）；body 为 null 清空 |
| POST | `/api/certificate/providers` | 新建 provider（acme，registry 多态解码）；tag 可选 |
| PUT | `/api/certificate/providers/{id}` | 替换 |
| DELETE | `/api/certificate/providers/{id}` | 被 tls.certificate_provider 引用时返回 409 + references；`?force=true` 自动清除引用后删除 |

- provider 引用方式：tag 字符串或内联对象（option/certificate_provider.go）；acme 需构建 tag `with_acme`
- certificate_path / PEM 校验会实际读取文件

## 工具

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/tools/uuid` | `{uuid}` 生成 v4 uuid |
| POST | `/api/tools/password` | `{password}` 生成 16 字节随机密码（标准 base64，shadowsocks 入站默认密码格式） |
| POST | `/api/tools/reality-keypair` | `{private_key, public_key}` Reality X25519 密钥对（URL-safe base64，与 `sing-box generate reality-keypair` 一致） |
| POST | `/api/tools/parse-json` | body `{json: "<文本>"}`；合法 → `{ok: true, data: <解析结果>}`，非法 → 400 |
| POST | `/api/reload` | 重载运行中的 sing-box（SIGHUP）；按 settings.reload 执行（auto 自动适配 systemd/rc-service/OpenWrt；或 systemd/pidfile/hook）；mode 为 none 返回 400 |

## sing-box 重载（settings.reload）

- sing-box 官方重载机制只有 SIGHUP（`cmd_run.go` 收到 SIGHUP 重载配置）；clash_api **无** reload 端点
- `mode`：`auto`（默认，自动探测：systemd → openrc/rc-service → OpenWrt/procd → SysV service，`service` 字段为服务名/init 脚本名）/ `systemd`（systemctl reload &lt;service&gt;，默认 sing-box）/ `pidfile`（读 pid 文件 → kill -HUP，仅 Linux）/ `hook`（sh -c 自定义命令）/ `none`（禁用）
- `after_save`：开启后所有配置写操作保存成功自动触发重载（成功响应含 `reloaded: true`；失败附 `reload_error` 不影响保存）
- 权限提示：systemd 模式需 sing-controller 用户有 unit 管理权限（root/polkit）；hook 模式如需 sudo 配置 NOPASSWD

## 写操作响应与校验

- 成功：`{"saved": true}`
- 所有写操作（PUT/POST/DELETE）执行统一校验管线：
  **严格解码（未知字段/多态/重复 tag）→ `box.New` 干跑预检 → 原子写盘（`.bak` 备份）**
- 校验失败：400 + `{"error": "..."}`，**不落盘、内存回滚**
- 删除 outbound/inbound 时检查 route 引用（`route.final` + 规则 `outbound`/`inbound` 字段），被引用则 400
