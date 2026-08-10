# sing-box-webui

基于 [sing-box](https://github.com/sagernet/sing-box) 源码（复用其 `option` 字段定义 / 校验管线 / JSON Schema 生成）
的可视化配置系统。**单二进制交付：webui 嵌入 controller，前后端同端口运行。**

## 架构

| 模块 | 位置 | 职责 |
|---|---|---|
| **sing-box** | 外部依赖（fork） | 代理核心。本仓库不包含、不修改，仅通过 `go.mod replace` 复用其 Go 包做配置校验 |
| **sing-box-controller** | `controller/` | 只负责 sing-box 主配置 `config.json` 的**读取、校验、生成**，不运行代理实例；`go:embed` 内嵌前端 |
| **webui** | `controller/web/` | Vue 3 + Element Plus 前端，独立工程（独立 package.json / Vite）；构建产物 `web/dist` 被 controller embed |

## 环境要求

- **Go ≥ 1.25.5**（依赖 sing-box fork 的版本要求）
- **Node.js ≥ 20**（构建前端；CI 用 22）
- 建议终端使用 PowerShell 7 / Git Bash（见「开发注意事项」）

## 快速开始

```bash
# 1. 构建前端（产物被 controller embed）
cd controller/web
npm install && npm run build

# 2. 启动 controller（页面 + API 同端口，默认 127.0.0.1:8080）
cd ..
go run -tags "with_quic with_utls" . -config config.json
# 浏览器打开 http://127.0.0.1:8080 即 webui
```

前端开发模式（HMR，`/api` 代理到 8080，controller 需同时运行）：

```bash
cd controller/web
npm run dev     # http://localhost:5173
```

> **构建必须带 `-tags "with_quic with_utls"`**：vless+reality 依赖 uTLS、tuic 依赖 QUIC，
> 否则校验管线直接报 `uTLS is not included in this build`。
>
> `web/dist` 未构建时 controller 自动退化为 **API-only 模式**（根路径返回提示），API 不受影响。

## Controller 配置（`config.json`，首次启动自动生成）

```jsonc
{
  "config": "./sing-box-config.json",   // sing-box 主配置文件路径（相对 controller 工作目录）
  "listen": "127.0.0.1:8080",           // HTTP 监听地址（webui 访问地址；改后需重启服务）
  "log": { "level": "info" },           // 日志级别：trace/debug/info/warn/error/fatal/panic（与 sing-box 枚举一致）
  "min_port": 8000,                     // 自动分配端口的最小起点
  "defaults": {                         // 前端新建 inbound/outbound 时的默认值
    "inbound_type": "mixed",
    "outbound_type": "vless",
    "listen": "127.0.0.1",
    "listen_port": 2080
  }
}
```

- `config` 指向的主配置文件不存在时，自动生成骨架（默认 inbound + direct/block outbound + route.final）
- 命令行 `-listen` 优先于配置文件 `listen`
- 前端 Settings 页可在线修改；`config` 路径变更立即生效，`listen` 变更需重启
- `listen` 使用特权端口（<1024）时保存接口返回 warning：deb 部署已带 `CAP_NET_BIND_SERVICE`，本机直跑需 root 或加 capabilities

## 功能

- **Outbound CRUD**：vless+reality（utls 指纹、Reality 密钥对生成）、tuic v5 专属表单；其他类型原始 JSON 兜底
- **Inbound CRUD**：mixed 专属表单（users 动态行）；预填默认 type/listen/端口，支持**自动分配最小可用端口**（min_port 起）
- **Route CRUD**：简单规则 `{"inbound": [...], "outbound": "tag"}`；稳定 id 存旁车 meta（`config.json.meta`）；引用保护（删除被路由引用的对象会被拦截）
- **粘贴 JSON 解析**：表单粘贴 JSON → 后端解析校验 → 填充字段
- **完整校验管线**：所有写操作 = 严格解码（未知字段/多态/重复 tag 检查）→ `box.New` 干跑预检 → 原子写盘（.bak 备份）；失败不落盘、内存回滚
- **JSON Schema 自动生成**（`GET /api/schema`，与代码同步）

## 复用 sing-box 的关键点

- `controller/go.mod`：`require github.com/sagernet/sing-box v1.14.0-beta.13` + **`replace => ../../sing-box`**
  （复用 [qualvey/sing-box](https://github.com/qualvey/sing-box) fork 源码，本地目录与 repo 平级）
- 校验管线：`json.UnmarshalExtendedContext[option.Options](include.Context(ctx), content)` + `box.New(...)` 干跑
  （复用 `daemon.CheckConfig` 模式；只实例化不 Start，不监听端口）
- **必须用 `replace` 而非 `go.work`**：go.work 不屏蔽 require 版本的依赖图，MVS 会把 sing-quic/quic-go
  抬升到与 fork 源码不兼容的版本（`DialEarly` 参数不匹配）导致编译失败
- `store.Parse` 内部强制注入 `include.Context`——所有入口（启动/API/校验）都能解码
  inbounds/outbounds/endpoints/services 等多态类型（生产踩过：启动路径漏 registry 导致
  `services[0]: missing service fields registry`）
- JSON Schema：`schema.Generate(include.Context(ctx), reflect.TypeFor[option.Options]())`

## 构建与发布（CI/CD，tag 触发）

```bash
# 本地交叉编译
cd controller
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags "with_quic with_utls" -o sing-controller .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags "with_quic with_utls" -o sing-controller .
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -tags "with_quic with_utls" -o sing-controller .

# 发布（推送 tag 触发 .github/workflows/release.yml → GoReleaser → GitHub Release）
git tag v0.1.5 && git push origin v0.1.5

# 本地预览产物（不发布）
goreleaser release --snapshot --clean
```

CI 流程：checkout 主仓库 → clone `qualvey/sing-box` fork 到 `<workspace>/../sing-box`（与本地布局一致）
→ setup-go(1.26) + setup-node(22) → `npm ci && npm run build` → goreleaser（自动 embed dist）。

自动产物：linux amd64 / arm64 / armv7 的 `tar.gz` + **deb**（含 systemd 单元，安装后自动建用户、启停服务）。

## Linux 部署（deb）

```bash
sudo dpkg -i sing-controller_<version>_linux_<arch>.deb
sudo systemctl status sing-controller
# 浏览器访问 http://<host>:8080（listen 可改 /etc/sing-controller/config.json 后 restart）
```

安装后：
- 用户/组 `sing-controller`（无特权、nologin），配置目录 `/etc/sing-controller`
- systemd 服务 `sing-controller`（journald 日志：`journalctl -u sing-controller -f`），带 `CAP_NET_BIND_SERVICE`（可监听 80/443）
- 默认主配置 `/etc/sing-controller/sing-box-config.json`（不存在自动生成骨架）
- controller 配置 `/etc/sing-controller/config.json` 为 `noreplace`（升级不覆盖）

## 开发注意事项（踩过的坑）

1. **Windows 下禁用 PowerShell 5.1 的 `Get-Content`/`Set-Content` 处理含中文文件**（默认 ANSI/GBK 编解码，
   会把 UTF-8 中文双重编码损坏并固化进 git）。一律用编辑器/`write` 工具或 .NET `File.WriteAllText(UTF8 无 BOM)`。
   建议直接用 PowerShell 7。
2. **`go run` 的编译产物子进程不会随终端关闭退出**，会持续占用端口——排查"代码改了没生效"先查
   `netstat -ano | findstr :<端口>` 找 PID 清理。
3. **curl 传 JSON 引号会被 PowerShell 吞**：用 `-d @file.json`（文件 UTF-8 无 BOM）。
4. `json.MarshalContext(ctx, value)` 传**值**会丢协议字段（指针接收者 `MarshalJSONContext` 不生效），必须传指针 `&item`。
5. Windows Smart App Control 可能拦截本地编译的未签名 exe（瞬时，重建后放行）；开发期用 `go run`。
6. gitignore 里 `config.json` 会误伤 `packaging/config.json`——已用 `!packaging/config.json` 白名单，新增同类文件注意。

## API 摘要（完整契约见 API.md）

```
GET  /                      webui 页面（嵌入；未构建时返回服务信息）
GET  /healthz               健康检查 {"status":"ok"}
GET  /api/status | /api/config | /api/schema | /api/types | /api/settings
PUT  /api/config | /api/settings
GET  /api/ports/available?start=N
POST /api/tools/uuid | /api/tools/reality-keypair | /api/tools/parse-json
GET/POST        /api/outbounds          /api/inbounds
GET/PUT/DELETE  /api/outbounds/{tag}    /api/inbounds/{tag}
GET/POST        /api/routes
PUT/DELETE      /api/routes/{id}
```

## 已知限制

- 反复 PUT 会规范化手写格式（注释丢失）；规则 id 是旁车元数据，外部手工改配置后会重新生成
- 可用端口探测基于本机 TCP bind 测试，跨机部署时以目标机为准
- controller 不运行代理实例：主配置保存即生效，由 sing-box 进程自行 watch/重启

## 项目结构

```
controller/              # sing-box-controller（Go module，嵌入 webui）
├── main.go              # 入口：-listen / -config(controller配置) / -secret / version 注入
├── webui.go             # go:embed web/dist + SPA fallback + 缓存头（可选嵌入）
├── web/                 # webui 前端（Vue3 + Vite + TS + Element Plus）
│   └── dist/            # 构建产物（被 embed，gitignore；未构建时 API-only）
└── internal/
    ├── settings/        # controller 自身配置（config/listen/log/min_port/defaults）
    ├── store/           # sing-box 主配置：加载 / 校验管线 / 原子写 / 回滚 / meta
    └── api/             # REST handlers（web 页面 / outbound / inbound / route / config / settings / ports / tools）
API.md                   # API 契约
.github/workflows/       # release.yml：tag 触发 CI/CD
packaging/               # deb 打包资源（systemd unit、默认配置、postinstall/postremove）
.goreleaser.yaml         # 交叉编译 + deb 打包配置
```
