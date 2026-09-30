# sing-controller RESTful API 规范契约

本文档为 `sing-controller` 配置管理后端服务的完整 RESTful API 契约规范，严格与后端实现代码（Go `controller/internal/api/`）对齐。

---

## 1. 全局设计规范

### 1.1 通信协议与基地址
- **Base URL**: `http://<listen_host>:<listen_port>`（默认 `http://127.0.0.1:8080`）
- **数据格式**: 请求与响应默认均为 `application/json; charset=utf-8`（除 Raw 配置接口与静态资源外）
- **CORS 支持**: 开发环境下允许跨域携带凭证与自定义 Header

### 1.2 认证鉴权 (Optional Auth)
当 controller 启动时指定了 `-secret <token>`（或通过环境变量配置）时，所有非公开请求需在请求头或查询参数中携带鉴权信息：
- **Request Header**: `X-Secret: <your_secret>`
- **Query Parameter**: `?token=<your_secret>`
- **鉴权失败响应 (`401 Unauthorized`)**:
  ```json
  { "error": "invalid secret" }
  ```

### 1.3 统一写操作管线与响应格式
所有涉及 sing-box 配置的写操作（POST / PUT / DELETE）均遵循 **内存修改 → sing-box (box.New) 全量干跑校验 → 原子写盘** 管线。校验失败一律不写盘、内存自动回滚。

- **标准成功响应 (`200 OK`)**:
  ```json
  {
    "saved": true
  }
  ```
  > 若配置中开启了 `settings.reload.after_save: true`，成功响应会附加重载状态：
  > ```json
  > {
  >   "saved": true,
  >   "reloaded": true
  > }
  > ```
  > 若重载失败但配置已保存：
  > ```json
  > {
  >   "saved": true,
  >   "reload_error": "systemctl reload sing-box 失败: ...",
  >   "message": "配置已保存，但 sing-box 重载失败"
  > }
  > ```

- **标准客户端错误响应 (`400 Bad Request`)**:
  ```json
  {
    "error": "错误详细描述（如校验失败、参数缺失、tag 冲突等）"
  }
  ```

- **资源未找到响应 (`404 Not Found`)**:
  ```json
  {
    "error": "inbound 不存在: mixed-in"
  }
  ```

- **依赖引用冲突响应 (`409 Conflict`)**:
  当尝试删除被其他资源（路由规则、选择器组、DNS 配置等）引用的对象且未加强制标志时触发：
  ```json
  {
    "error": "outbound 被组引用: proxy-group, fallback-group",
    "references": [
      "proxy-group",
      "fallback-group"
    ]
  }
  ```
  > **处理方式**：客户端可在用户确认后，追加 `?force=true` 查询参数重新发起 DELETE 请求，后端将自动清除关联引用后执行删除。

---

## 2. 概览与速查表

| 分组 | 方法 | 路径 | 核心功能说明 |
|---|---|---|---|
| **服务与探活** | `GET` | `/` | 获取服务元信息（API-only 模式）或 SPA 页面 |
| | `GET` | `/healthz` | 服务探活健康检查 |
| | `*` | `/api/clash/*` | Clash API 透明反向代理 |
| | `*` | `/api/grpc/*` | gRPC-Web / WebSocket 服务代理 |
| **状态与配置** | `GET` | `/api/status` | 获取运行状态与资源统计 |
| | `GET` | `/api/config` | 获取完整解析后的 sing-box 主配置 (JSON) |
| | `PUT` | `/api/config` | 全量更新主配置 (JSON) |
| | `GET` | `/api/config/raw` | 获取配置文件原始文本（保留注释） |
| | `PUT` | `/api/config/raw` | 保存配置文件原始文本（保留注释） |
| | `GET` | `/api/schema` | 获取 sing-box 完整 JSON Schema |
| | `GET` | `/api/types` | 获取支持的 Inbounds / Outbounds 等类型列表 |
| | `GET` | `/api/diagnostics` | 静态配置分析诊断（引用悬空、tag 重复等） |
| **Controller 设置** | `GET` | `/api/settings` | 读取 controller 自身运行设置 |
| | `PUT` | `/api/settings` | 更新 controller 运行设置 |
| **节点健康探测** | `GET` | `/api/node-health` | 获取节点可用状态与最近探测结果 |
| | `GET` | `/api/node-health/config` | 读取探测与 webhook 配置（Webhook 凭据不回传） |
| | `PUT` | `/api/node-health/config` | 更新探测与 webhook 配置 |
| | `GET` | `/api/ports/available` | 探测并分配下一个可用 TCP 端口 |
| **用户池 (Users)** | `GET` | `/api/users` | 获取全局用户池列表 |
| | `POST` | `/api/users` | 创建用户并同步绑定到指定入站 |
| | `PUT` | `/api/users/{name}` | 更新用户凭证及绑定关系 |
| | `DELETE` | `/api/users/{name}` | 删除用户并从所有入站中解绑 |
| **入站 (Inbounds)** | `GET` | `/api/inbounds` | 获取入站列表 |
| | `POST` | `/api/inbounds` | 创建入站配置（自动注入池用户） |
| | `GET` | `/api/inbounds/{tag}` | 获取指定入站详情 |
| | `PUT` | `/api/inbounds/{tag}` | 替换指定入站配置 |
| | `DELETE` | `/api/inbounds/{tag}` | 删除入站配置 |
| **出站 (Outbounds)** | `GET` | `/api/outbounds` | 获取出站列表 |
| | `POST` | `/api/outbounds` | 创建出站（支持自动并入 Selector 组） |
| | `GET` | `/api/outbounds/{tag}` | 获取指定出站详情 |
| | `PUT` | `/api/outbounds/{tag}` | 替换指定出站配置 |
| | `DELETE` | `/api/outbounds/{tag}` | 删除出站（支持 `?force=true`） |
| **路由规则 (Routes)** | `GET` | `/api/routes` | 获取路由规则列表及 final 出站 |
| | `POST` | `/api/routes` | 新建路由规则 |
| | `GET` | `/api/routes/{id}` | 获取指定路由规则详情 |
| | `PUT` | `/api/routes/{id}` | 替换指定路由规则 |
| | `DELETE` | `/api/routes/{id}` | 删除路由规则 |
| | `PUT` | `/api/routes/{id}/toggle` | 切换路由规则启用/禁用状态 |
| **DNS 管理** | `GET` | `/api/dns` | 获取 DNS 服务器、规则与基础选项 |
| | `POST` | `/api/dns/servers` | 新建 DNS 服务器 |
| | `PUT` | `/api/dns/servers/{tag}` | 替换 DNS 服务器配置 |
| | `DELETE` | `/api/dns/servers/{tag}` | 删除 DNS 服务器（支持 `?force=true`） |
| | `POST` | `/api/dns/rules` | 新建 DNS 规则 |
| | `PUT` | `/api/dns/rules/{id}` | 替换 DNS 规则 |
| | `DELETE` | `/api/dns/rules/{id}` | 删除 DNS 规则 |
| | `PUT` | `/api/dns/options` | 更新全局 DNS 基础选项 |
| **规则集 (Rule-Sets)**| `GET` | `/api/rule-sets` | 获取规则集列表 |
| | `POST` | `/api/rule-sets` | 新建规则集 (inline / local / remote) |
| | `PUT` | `/api/rule-sets/{id}` | 替换规则集配置 |
| | `DELETE` | `/api/rule-sets/{id}` | 删除规则集（支持 `?force=true`） |
| **证书管理** | `GET` | `/api/certificate` | 获取全局证书配置及 Providers 列表 |
| | `PUT` | `/api/certificate` | 替换或清空全局 certificate 段 |
| | `POST` | `/api/certificate/providers` | 新建证书 Provider（如 ACME） |
| | `PUT` | `/api/certificate/providers/{id}` | 替换证书 Provider 配置 |
| | `DELETE` | `/api/certificate/providers/{id}` | 删除证书 Provider（支持 `?force=true`） |
| **工具与控制** | `POST` | `/api/tools/uuid` | 生成标准 UUID v4 |
| | `POST` | `/api/tools/password` | 生成 16 字节随机 Base64 密码 |
| | `POST` | `/api/tools/reality-keypair`| 生成 Reality X25519 密钥对 |
| | `POST` | `/api/tools/parse-json` | 校验并解析 JSON 字符串 |
| | `POST` | `/api/tools/parse-node` | 解析节点配置或代理链接（vless://、vmess://、ss://、JSON 等） |
| | `POST` | `/api/reload` | 手动触发 sing-box 进程重载 (SIGHUP) |

---

## 3. 详细接口说明

### 3.1 服务与健康检查

#### GET `/`
获取服务基本信息（API-only 模式）或 SPA Web 界面。
- **Response `200 OK`**:
  ```json
  {
    "service": "sing-controller",
    "version": "1.0.0",
    "api": "/api",
    "status": "/api/status",
    "schema": "/api/schema",
    "listen": "127.0.0.1:8080",
    "config": "/etc/sing-box/config.json"
  }
  ```

#### GET `/healthz`
健康检查探活端点（通常用于 systemd/Kubernetes 探针，内部采用 Debug 日志避免刷屏）。
- **Response `200 OK`**:
  ```json
  { "status": "ok" }
  ```

#### ANY `/api/clash/*`
Clash API 透明反向代理，无缝转发到 sing-box 的 `experimental.clash_api`（自动携带 Secret 鉴权）。
覆盖 `/proxies`, `/connections`, `/logs`, `/traffic`, `/rules`, `/configs`, `/dns/query` 等所有 Clash REST 端点。

---

### 3.2 系统状态与配置管理

#### GET `/api/status`
获取 Controller 及 sing-box 整体运行统计与元数据。
- **Response `200 OK`**:
  ```json
  {
    "config_path": "./config.json",
    "controller_config": "./controller.json",
    "listen": "127.0.0.1:8080",
    "log_level": "info",
    "min_port": 8000,
    "defaults": {
      "inbound_type": "mixed",
      "outbound_type": "vless",
      "listen": "127.0.0.1",
      "listen_port": 2080,
      "attach_to_selector": true,
      "proxy_selector": "Proxy"
    },
    "inbounds": 3,
    "outbounds": 8,
    "rules": 12
  }
  ```

#### GET `/api/config`
获取当前已解析的 sing-box 完整配置 JSON 对象（经 Go 数据结构反序列化再序列化，不保留原始注释）。
- **Response `200 OK`**:
  ```json
  {
    "log": { "level": "info" },
    "inbounds": [ ... ],
    "outbounds": [ ... ],
    "route": { ... }
  }
  ```

#### PUT `/api/config`
全量替换 sing-box 主配置（JSON 格式）。通过 dry-run 校验后写入磁盘。
- **Request Body**: 完整的 sing-box 配置 JSON 对象。
- **Response `200 OK`**: `{"saved": true}`

#### GET `/api/config/raw`
获取配置文件的原始文本内容（保留开发者手动编写的注释、格式及字段顺序）。
- **Response Headers**: `Content-Type: text/plain; charset=utf-8`
- **Response Body**: 原始文件文本。

#### PUT `/api/config/raw`
原样保存配置文本。后端进行 JSON 解析与 `box.New` 校验，校验通过后原子写入磁盘。
- **Request Headers**: `Content-Type: text/plain`
- **Request Body**: 配置文件字符串。
- **Response `200 OK`**: `{"saved": true}`

#### GET `/api/schema`
获取 sing-box 针对当前版本的完整 JSON Schema 定义（自动生成，约 400KB+）。
- **Response `200 OK`**: JSON Schema 根对象。

#### GET `/api/types`
获取 sing-box 内置支持的协议与模块类型枚举。
- **Response `200 OK`**:
  ```json
  {
    "inbounds": ["mixed", "shadowsocks", "tuic", "vless", "vmess", "trojan", "hysteria2", "direct"],
    "outbounds": ["direct", "block", "dns", "selector", "urltest", "vless", "vmess", "trojan", "shadowsocks", "tuic", "hysteria2"],
    "endpoints": ["wireguard"],
    "services": []
  }
  ```

#### GET `/api/diagnostics`
执行静态配置诊断分析（检查重复 tag、悬空引用、端口冲突、未引用的出站等）。
- **Response `200 OK`**:
  ```json
  {
    "diagnostics": [
      {
        "level": "error",
        "message": "outbound tag 重复: proxy-01（2 次）"
      },
      {
        "level": "warning",
        "message": "route 规则 #3 引用了不存在的 outbound: hk-node"
      },
      {
        "level": "info",
        "message": "共 3 个入站, 8 个出站, 12 条路由规则"
      }
    ]
  }
  ```

---

### 3.3 Controller 设置 (Settings)

#### GET `/api/settings`
获取 controller 自身的配置文件设置。
- **Response `200 OK`**:
  ```json
  {
    "config": "./config.json",
    "listen": "127.0.0.1:8080",
    "log": { "level": "info" },
    "min_port": 8000,
    "reload": {
      "mode": "auto",
      "service": "sing-box",
      "after_save": false
    },
    "defaults": {
      "inbound_type": "mixed",
      "outbound_type": "vless",
      "listen": "0.0.0.0",
      "listen_port": 10808,
      "attach_to_selector": true,
      "proxy_selector": "Proxy"
    }
  }
  ```

#### PUT `/api/settings`
更新 controller 自身设置。
- **Request Body**: 与 GET `/api/settings` 结构一致的 JSON 对象。
- **Response `200 OK`**:
  ```json
  { "saved": true }
  ```

#### GET `/api/ports/available`
探测系统下一个可绑定的空闲 TCP 端口。
- **Query Parameters**:
  - `start` *(integer, 可选)*: 起始端口号（须 $\ge 1024$，缺省时使用 `settings.min_port`）。
- **Response `200 OK`**:
  ```json
  {
    "port": 8001,
    "start": 8000
  }
  ```

---

### 3.4 全局用户池 (Users)

用户池在旁车元数据中维护，多对多绑定到各个入站。当用户增删改时，系统会自动将用户的凭证（按入站协议类型自动投影相应字段）同步到各个入站的 `users[]` 数组中。

#### GET `/api/users`
获取所有用户列表。
- **Response `200 OK`**:
  ```json
  {
    "users": [
      {
        "name": "alice",
        "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "password": "secretPassword123",
        "flow": "xtls-rprx-vision",
        "bound_inbounds": ["vless-in", "tuic-in"]
      }
    ]
  }
  ```

#### POST `/api/users`
新建全局用户并同步绑定至目标入站。
- **Request Body**:
  ```json
  {
    "name": "bob",
    "uuid": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
    "password": "password456",
    "flow": "",
    "bound_inbounds": ["mixed-in"]
  }
  ```
- **Response `200 OK`**: `{"saved": true}`

#### PUT `/api/users/{name}`
更新指定用户的凭证与入站绑定（注：不允许修改 `name`）。
- **Path Parameters**:
  - `name` *(string)*: 用户名称。
- **Request Body**:
  ```json
  {
    "uuid": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
    "password": "newPassword789",
    "bound_inbounds": ["mixed-in", "trojan-in"]
  }
  ```
- **Response `200 OK`**: `{"saved": true}`

#### DELETE `/api/users/{name}`
删除用户并从其绑定的所有入站中清除该用户。
- **Path Parameters**:
  - `name` *(string)*: 用户名称。
- **Response `200 OK`**: `{"saved": true}`

---

### 3.5 入站管理 (Inbounds)

#### GET `/api/inbounds`
获取所有入站配置列表。
- **Response `200 OK`**:
  ```json
  {
    "inbounds": [
      {
        "type": "mixed",
        "tag": "mixed-in",
        "listen": "::",
        "listen_port": 2080,
        "sniff": true
      }
    ]
  }
  ```

#### POST `/api/inbounds`
创建入站配置。创建完成后会自动根据用户池绑定关系将用户信息注入到本入站的 `users[]` 字段中。
- **Request Body**:
  ```json
  {
    "type": "tuic",
    "tag": "tuic-in",
    "listen": "::",
    "listen_port": 8443,
    "congestion_control": "bbr",
    "tls": {
      "enabled": true,
      "server_name": "example.com",
      "certificate_provider": "my-acme"
    }
  }
  ```
- **Response `200 OK`**: `{"saved": true}`

#### GET `/api/inbounds/{tag}`
获取单个入站的完整配置。
- **Path Parameters**:
  - `tag` *(string)*: 入站唯一标识。
- **Response `200 OK`**: 单个入站 JSON 对象。

#### PUT `/api/inbounds/{tag}`
全量更新入站配置（请求体中的 `tag` 必须与路径参数一致）。
- **Path Parameters**:
  - `tag` *(string)*: 入站唯一标识。
- **Request Body**: 单个入站 JSON 对象。
- **Response `200 OK`**: `{"saved": true}`

#### DELETE `/api/inbounds/{tag}`
删除入站配置（若被路由规则引用，将返回 400 阻止删除）。
- **Path Parameters**:
  - `tag` *(string)*: 入站唯一标识。
- **Response `200 OK`**: `{"saved": true}`

---

### 3.6 出站管理 (Outbounds)

#### GET `/api/outbounds`
获取所有出站配置列表。
- **Response `200 OK`**:
  ```json
  {
    "outbounds": [
      {
        "type": "selector",
        "tag": "Proxy",
        "outbounds": ["vless-out", "direct"]
      },
      {
        "type": "vless",
        "tag": "vless-out",
        "server": "example.com",
        "server_port": 443,
        "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "flow": "xtls-rprx-vision",
        "network": "tcp",
        "tls": {
          "enabled": true,
          "server_name": "example.com",
          "utls": { "enabled": true, "fingerprint": "chrome" },
          "reality": { "enabled": true, "public_key": "xxx" }
        }
      }
    ]
  }
  ```

#### POST `/api/outbounds`
创建出站。若在 `settings` 中启用了 `attach_to_selector`，系统会自动将新建的 outbound tag 追加到指定的 selector/urltest 组中。
- **Request Body**: 出站 JSON 对象（必须包含 `type` 和 `tag`）。
- **Response `200 OK`**:
  ```json
  {
    "saved": true,
    "attached": true
  }
  ```

#### GET `/api/outbounds/{tag}`
获取指定出站配置。
- **Path Parameters**:
  - `tag` *(string)*: 出站唯一标识。
- **Response `200 OK`**: 出站 JSON 对象。

#### PUT `/api/outbounds/{tag}`
全量更新指定出站配置。
- **Path Parameters**:
  - `tag` *(string)*: 出站唯一标识。
- **Request Body**: 出站 JSON 对象。
- **Response `200 OK`**: `{"saved": true}`

#### DELETE `/api/outbounds/{tag}`
删除出站配置。
- **Path Parameters**:
  - `tag` *(string)*: 出站唯一标识。
- **Query Parameters**:
  - `force` *(boolean, 可选, 默认 false)*: 若为 `true`，当被 selector/urltest 组引用时，自动从这些组的列表中移除该 tag 后删除。
- **Response `200 OK`**: `{"saved": true}`
- **Response `409 Conflict`** (当被组引用且 `force=false` 时):
  ```json
  {
    "error": "outbound 被组引用: Proxy, Auto",
    "references": ["Proxy", "Auto"]
  }
  ```

---

### 3.7 路由规则 (Routes)

路由规则内部无原生 id 字段，由 Controller 旁车元数据维护稳定 `id`（UUID）。

#### GET `/api/routes`
获取路由规则列表以及默认兜底出站 (`final`)。
- **Response `200 OK`**:
  ```json
  {
    "routes": [
      {
        "id": "c3d4e5f6-789a-bcde-f012-3456789abcde",
        "rule": {
          "inbound": ["mixed-in"],
          "outbound": "Proxy",
          "network": "tcp"
        },
        "disabled": false
      }
    ],
    "final": "direct"
  }
  ```

#### POST `/api/routes`
新建路由规则（追加至规则末尾）。
- **Request Body**: 标准 sing-box 路由规则 JSON 对象。
  ```json
  {
    "domain_suffix": [".google.com", ".github.com"],
    "outbound": "Proxy"
  }
  ```
- **Response `200 OK`**: `{"saved": true}`

#### GET `/api/routes/{id}`
获取单个路由规则详情。
- **Path Parameters**:
  - `id` *(string)*: 规则唯一 ID。
- **Response `200 OK`**:
  ```json
  {
    "id": "c3d4e5f6-789a-bcde-f012-3456789abcde",
    "rule": {
      "domain_suffix": [".google.com"],
      "outbound": "Proxy"
    },
    "disabled": false
  }
  ```

#### PUT `/api/routes/{id}`
替换指定路由规则内容。
- **Path Parameters**:
  - `id` *(string)*: 规则唯一 ID。
- **Request Body**: 标准 sing-box 路由规则 JSON 对象。
- **Response `200 OK`**: `{"saved": true}`

#### DELETE `/api/routes/{id}`
删除指定路由规则。
- **Path Parameters**:
  - `id` *(string)*: 规则唯一 ID。
- **Response `200 OK`**: `{"saved": true}`

#### PUT `/api/routes/{id}/toggle`
切换规则启用/禁用状态。
- **Path Parameters**:
  - `id` *(string)*: 规则唯一 ID。
- **Request Body**:
  ```json
  {
    "disabled": true
  }
  ```
- **Response `200 OK`**: `{"saved": true}`

---

### 3.8 DNS 管理

#### GET `/api/dns`
获取 DNS 全量配置，包括 DNS 服务器列表、DNS 规则列表及全局 DNS 基础选项。
- **Response `200 OK`**:
  ```json
  {
    "servers": [
      {
        "tag": "local-dns",
        "address": "local",
        "detour": "direct"
      },
      {
        "tag": "remote-dns",
        "address": "tls://1.1.1.1",
        "detour": "Proxy"
      }
    ],
    "rules": [
      {
        "id": "d4e5f6a7-89bc-def0-1234-56789abcdef0",
        "rule": {
          "domain_suffix": [".cn"],
          "server": "local-dns"
        },
        "disabled": false
      }
    ],
    "options": {
      "final": "remote-dns",
      "strategy": "prefer_ipv4",
      "disable_cache": false
    }
  }
  ```

#### POST `/api/dns/servers`
新建 DNS 服务器配置。
- **Request Body**:
  ```json
  {
    "tag": "google-dns",
    "address": "https://dns.google/dns-query",
    "detour": "Proxy"
  }
  ```
- **Response `200 OK`**: `{"saved": true}`

#### PUT `/api/dns/servers/{tag}`
替换指定 DNS 服务器配置。
- **Path Parameters**:
  - `tag` *(string)*: DNS Server tag。
- **Request Body**: DNS Server JSON 对象。
- **Response `200 OK`**: `{"saved": true}`

#### DELETE `/api/dns/servers/{tag}`
删除指定 DNS 服务器。
- **Path Parameters**:
  - `tag` *(string)*: DNS Server tag。
- **Query Parameters**:
  - `force` *(boolean, 可选, 默认 false)*: 若为 `true`，当被 `dns.final` 或 DNS 规则引用时，自动从规则中移除该引用后删除。
- **Response `200 OK`**: `{"saved": true}`
- **Response `409 Conflict`**: 返回被引用的具体规则位置。

#### POST `/api/dns/rules`
新建 DNS 规则（注意：sing-box 1.14 规范中动作字段为 `server`）。
- **Request Body**:
  ```json
  {
    "rule_set": ["geosite-cn"],
    "server": "local-dns"
  }
  ```
- **Response `200 OK`**: `{"saved": true}`

#### PUT `/api/dns/rules/{id}`
替换指定 DNS 规则。
- **Path Parameters**:
  - `id` *(string)*: DNS 规则 ID。
- **Request Body**: DNS 规则 JSON 对象。
- **Response `200 OK`**: `{"saved": true}`

#### DELETE `/api/dns/rules/{id}`
删除指定 DNS 规则。
- **Path Parameters**:
  - `id` *(string)*: DNS 规则 ID。
- **Response `200 OK`**: `{"saved": true}`

#### PUT `/api/dns/options`
部分更新 DNS 基础选项参数。
- **Request Body**:
  ```json
  {
    "final": "remote-dns",
    "strategy": "prefer_ipv4",
    "disable_cache": false,
    "independent_cache": false,
    "reverse_mapping": false,
    "client_subnet": "1.2.3.4/24"
  }
  ```
- **Response `200 OK`**: `{"saved": true}`

---

### 3.9 规则集 (Rule-Sets)

#### GET `/api/rule-sets`
获取所有在 `route.rule_set` 中定义的规则集。
- **Response `200 OK`**:
  ```json
  {
    "rule_sets": [
      {
        "id": "e5f6a7b8-9cde-f012-3456-789abcdef012",
        "rule_set": {
          "type": "remote",
          "tag": "geosite-geolocation-cn",
          "format": "binary",
          "url": "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-geolocation-!cn.srs",
          "download_detour": "Proxy"
        }
      }
    ]
  }
  ```

#### POST `/api/rule-sets`
新建规则集。支持 inline（内联）、local（本地文件）和 remote（远程 URL 下载）三种类型。
- **Request Body**:
  ```json
  {
    "type": "remote",
    "tag": "geoip-cn",
    "format": "binary",
    "url": "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs",
    "download_detour": "Proxy",
    "update_interval": "24h"
  }
  ```
- **Response `200 OK`**: `{"saved": true}`

#### PUT `/api/rule-sets/{id}`
替换指定规则集配置。
- **Path Parameters**:
  - `id` *(string)*: 规则集 ID。
- **Request Body**: 规则集 JSON 对象。
- **Response `200 OK`**: `{"saved": true}`

#### DELETE `/api/rule-sets/{id}`
删除指定规则集。
- **Path Parameters**:
  - `id` *(string)*: 规则集 ID。
- **Query Parameters**:
  - `force` *(boolean, 可选, 默认 false)*: 若为 `true`，当被 route 或 dns 规则引用时，自动从所有规则的 `rule_set` 列表中剔除后删除。
- **Response `200 OK`**: `{"saved": true}`
- **Response `409 Conflict`**: 返回被引用的路由/DNS 规则编号列表。

---

### 3.10 证书管理 (Certificate)

#### GET `/api/certificate`
获取顶层全局证书配置 (`certificate`) 以及多态证书 Provider 列表 (`certificate_providers`)。
- **Response `200 OK`**:
  ```json
  {
    "certificate": {
      "store": "system",
      "certificate": [],
      "certificate_path": ["/etc/ssl/certs/custom.pem"],
      "certificate_directory_path": []
    },
    "providers": [
      {
        "id": "f6a7b8c9-def0-1234-5678-9abcdef01234",
        "provider": {
          "type": "acme",
          "tag": "letsencrypt",
          "domain": ["example.com", "*.example.com"],
          "email": "admin@example.com",
          "key_type": "p256",
          "dns01_challenge": {
            "provider": "cloudflare",
            "api_token": "your_cloudflare_api_token"
          }
        }
      }
    ]
  }
  ```

#### PUT `/api/certificate`
整体替换全局 `certificate` 配置段（传入 `null` 可清空配置）。
- **Request Body**:
  ```json
  {
    "store": "system",
    "certificate_path": ["/etc/ssl/certs/custom.pem"]
  }
  ```
  *或清空配置：*
  ```json
  null
  ```
- **Response `200 OK`**: `{"saved": true}`

#### POST `/api/certificate/providers`
新建证书 Provider（如 ACME）。
- **Request Body**:
  ```json
  {
    "type": "acme",
    "tag": "acme-cf",
    "domain": ["domain.com"],
    "email": "user@mail.com",
    "dns01_challenge": {
      "provider": "cloudflare",
      "api_token": "secret_token"
    }
  }
  ```
- **Response `200 OK`**: `{"saved": true}`

#### PUT `/api/certificate/providers/{id}`
替换指定证书 Provider 配置。
- **Path Parameters**:
  - `id` *(string)*: Provider ID。
- **Request Body**: Provider JSON 对象。
- **Response `200 OK`**: `{"saved": true}`

#### DELETE `/api/certificate/providers/{id}`
删除指定证书 Provider。
- **Path Parameters**:
  - `id` *(string)*: Provider ID。
- **Query Parameters**:
  - `force` *(boolean, 可选, 默认 false)*: 若为 `true`，当被入站或出站的 `tls.certificate_provider` 引用时，强制删除。
- **Response `200 OK`**: `{"saved": true}`
- **Response `409 Conflict`**: 返回引用了该 Provider 的入站/出站列表。

---

### 3.11 辅助工具与进程控制 (Tools & Reload)

#### POST `/api/tools/uuid`
生成符合 UUID v4 规范的随机 UUID 字符串。
- **Request Body**: 空。
- **Response `200 OK`**:
  ```json
  {
    "uuid": "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
  }
  ```

#### POST `/api/tools/password`
生成 16 字节的安全随机密码（标准 Base64 编码，适合 Shadowsocks 等入站）。
- **Request Body**: 空。
- **Response `200 OK`**:
  ```json
  {
    "password": "8JCsPssfgS8tiRwiMlhARg=="
  }
  ```

#### POST `/api/tools/reality-keypair`
生成 Reality X25519 密钥对（URL-Safe Base64 编码，与 `sing-box generate reality-keypair` 格式一致）。
- **Request Body**: 空。
- **Response `200 OK`**:
  ```json
  {
    "private_key": "yAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
    "public_key": "wBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
  }
  ```

#### POST `/api/tools/parse-json`
校验并解析客户端传入的 JSON 字符串文本（常用于表单粘贴 JSON 解析参数）。
- **Request Body**:
  ```json
  {
    "json": "{\"server\": \"1.1.1.1\", \"port\": 53}"
  }
  ```
- **Response `200 OK`**:
  ```json
  {
    "ok": true,
    "data": {
      "server": "1.1.1.1",
      "port": 53
    }
  }
  ```

#### POST `/api/tools/parse-node`
解析客户端传入的节点配置文本或代理分享链接（支持 `vless://`, `vmess://`, `ss://`, `trojan://`, `tuic://`, `hy2://` 以及完整 JSON 节点）。
- **Request Body**:
  ```json
  {
    "text": "vless://4a3a3118-2bb1-4cd4-8848-f8605c3c1234@198.51.100.1:443?security=reality&sni=example.com&pbk=wBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB&sid=12345678&fp=chrome#US-Node"
  }
  ```
- **Response `200 OK`**:
  ```json
  {
    "ok": true,
    "data": {
      "type": "vless",
      "tag": "US-Node",
      "server": "198.51.100.1",
      "server_port": 443,
      "uuid": "4a3a3118-2bb1-4cd4-8848-f8605c3c1234",
      "tls": {
        "enabled": true,
        "server_name": "example.com",
        "reality": {
          "enabled": true,
          "public_key": "wBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
          "short_id": "12345678"
        },
        "utls": {
          "enabled": true,
          "fingerprint": "chrome"
        }
      }
    }
  }
  ```

#### POST `/api/reload`
手动向当前运行中的 sing-box 发送重载信号 (SIGHUP)。执行模式由 `settings.reload` 配置决定（支持 `auto`, `systemd`, `pidfile`, `hook`）。
- **Request Body**: 空。
- **Response `200 OK`**:
  ```json
  {
    "reloaded": true,
    "mode": "auto"
  }
  ```
- **Response `400 Bad Request`** (当重载失败或重载模式为 `none` 时):
  ```json
  {
    "error": "systemctl reload sing-box 失败: ..."
  }
  ```
