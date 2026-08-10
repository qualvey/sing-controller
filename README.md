# sing-box-webui

三模块架构，职责完全分离：

| 模块 | 职责 | 说明 |
|---|---|---|
| **sing-box** | 代理核心 | 官方项目，本仓库不包含、不修改，仅通过 `go.mod replace` 复用其 Go 包做配置校验 |
| **sing-box-controller**（`controller/`） | 配置管理服务 | 只负责 sing-box 主配置 `config.json` 的**读取、校验、生成**，不运行代理实例 |
| **webui**（`web/`） | 可视化前端 | Vue 3 + Element Plus，纯浏览器应用，通过 RESTful API 操作 controller |

## 快速开始

```bash
# 1. 启动 controller（默认监听 127.0.0.1:8080）
cd controller
go run -tags "with_quic with_utls" . -listen 127.0.0.1:8080 -config config.json

# 2. 启动前端（开发模式，/api 代理到 8080）
cd web
npm install && npm run dev     # http://localhost:5173

# 构建后端二进制（无需前端，前端独立部署）
cd controller
go build -tags "with_quic with_utls" -o sing-box-controller.exe .
```

> 构建必须带 `-tags "with_quic with_utls"`（vless+reality 依赖 uTLS、tuic 依赖 QUIC），
> 否则校验管线会报 `uTLS is not included in this build`。

## Controller 配置（controller 自身的 config.json）

```jsonc
{
  "config": "./sing-box-config.json",   // sing-box 主配置文件路径（相对 controller 工作目录）
  "min_port": 8000,                     // 自动分配端口的最小起点
  "defaults": {                         // 前端新建 inbound/outbound 时的默认值
    "inbound_type": "mixed",
    "outbound_type": "vless",
    "listen": "127.0.0.1",
    "listen_port": 2080
  }
}
```

- 首次启动自动生成该文件；`config` 指向的主配置文件不存在时，自动生成骨架
  （默认 inbound + direct/block outbound + route.final）
- 前端 Settings 页可在线修改；修改 `config` 路径立即生效（新路径不存在则重新生成骨架）

## 功能

- **Outbound CRUD**：vless+reality（utls 指纹、密钥对生成）、tuic v5 专属表单；其他类型原始 JSON 兜底
- **Inbound CRUD**：mixed 专属表单（users 动态行）；新建时预填默认 type/listen/端口，支持**自动分配最小可用端口**
- **Route CRUD**：简单规则 `{"inbound": [...], "outbound": "tag"}`（稳定 id 存旁车 meta，引用保护）
- **粘贴 JSON 解析**：任意类型表单均可粘贴 JSON → 后端解析校验 → 填充表单字段
- **完整校验管线**：所有写操作 解码（严格/多态/重复tag检查）→ `box.New` 干跑 → 原子写盘(.bak)；失败不落盘、内存回滚
- **JSON Schema 自动生成**（`GET /api/schema`，443KB，与代码同步）

## 复用 sing-box 的关键点

- `controller/go.mod`：`replace github.com/sagernet/sing-box => <本地 checkout 路径>`
- 校验管线：`json.UnmarshalExtendedContext[option.Options](include.Context(ctx), content)` + `box.New(...)` 干跑
  （复用 `daemon.CheckConfig` 模式；只实例化不 Start，不监听端口）
- JSON Schema：`schema.Generate(include.Context(ctx), reflect.TypeFor[option.Options]())`
- 重载语义：controller 不运行实例，无热重载概念；配置保存即生效（由 sing-box 进程自行 watch/重启）

## API 摘要（详见 API.md）

```
GET  /api/status | /api/config | /api/schema | /api/types | /api/settings
PUT  /api/config | /api/settings
GET  /api/ports/available?start=N       # 最小可用端口探测（默认 min_port 起）
POST /api/tools/uuid | /api/tools/reality-keypair | /api/tools/parse-json
GET/POST        /api/outbounds          /api/inbounds
GET/PUT/DELETE  /api/outbounds/{tag}    /api/inbounds/{tag}
GET/POST        /api/routes
PUT/DELETE      /api/routes/{id}
```

## 已知限制

- 反复 PUT 会规范化手写格式（注释丢失）；规则 id 是旁车元数据，外部改配置后会重新生成
- 可用端口探测基于本机 TCP bind 测试，跨机部署时以目标机为准
- Windows Smart App Control 可能拦截本地编译的未签名 exe；开发期用 `go run`，正式使用可关 SAC 或代码签名

## 项目结构

```
controller/              # sing-box-controller（Go）
├── main.go              # 入口：-listen / -config(controller配置) / -secret
└── internal/
    ├── settings/        # controller 自身配置（config 路径 / min_port / defaults）
    ├── store/           # sing-box 主配置：加载 / 校验管线 / 原子写 / 回滚 / meta
    └── api/             # REST handlers（outbound/inbound/route/config/settings/ports/tools）
web/                     # webui（Vue3 + Vite + TS + Element Plus）
API.md                   # API 契约
```
