# sing-box-webui

基于 sing-box 源码（复用其 `option` 字段定义 / 校验管线 / JSON Schema 生成）的
可视化配置系统。后端 Go + RESTful API，前端 Vue 3 + Element Plus，单二进制交付。

## 技术栈

| 层 | 选型 | 理由 |
|---|---|---|
| 后端 | Go 1.24+ / 标准库 net/http | 直接 import `sing-box` 包复用全部配置能力，零额外 Web 框架依赖 |
| 配置复用 | `option` / `include` / `schema` / `box` 包 | 严格解码、多态类型、校验、schema 生成全部来自 sing-box 自身 |
| 前端 | Vue 3 + Vite + TS + Element Plus + Pinia | 表单密集场景，Element Plus 生态成熟 |
| 交付 | go:embed 内嵌前端 dist | 单二进制，无部署依赖 |

## 复用 sing-box 的关键点

- `go.mod` 中 `replace github.com/sagernet/sing-box => <本地 checkout 路径>`
- 校验管线：`json.UnmarshalExtendedContext[option.Options](include.Context(ctx), content)`
  （严格解码 + 多态 + 重复 tag 检查）→ `box.New(...)` 干跑预检（复用 daemon.CheckConfig 模式）
- JSON Schema：`schema.Generate(include.Context(ctx), reflect.TypeFor[option.Options]())`
  → `/api/schema`（前端表单/编辑器可驱动）
- 重载语义与 sing-box SIGHUP 一致：close + recreate 整个实例；启动失败自动回滚旧配置

> 注意：vless+reality / tuic 需要 build tag `with_quic,with_utls`，
> 否则校验管线会报 `uTLS is not included in this build`（这正是管线该做的）。

## 功能

- **Outbound CRUD**：重点支持 vless+reality（含 utls 指纹、密钥对生成）、tuic v5；其他类型可用原始 JSON
- **Inbound CRUD**：重点支持 mixed（users 动态行）；其他类型原始 JSON
- **Route CRUD**：简单规则 `{"inbound": [...], "outbound": "tag"}`，规则 id 存于旁车 meta（`config.json.meta`），
  支持 route.final 修改；删除被引用的 outbound/inbound 会被引用保护拦截
- 实例管理：start / stop / reload（配置保存后自动热重载实例）
- 工具：uuid 生成、Reality X25519 密钥对生成（URL-safe base64，与 `sing-box generate reality-keypair` 一致）

## 运行

```bash
# 开发
go run -tags "with_quic with_utls" . -listen 127.0.0.1:8080 -config config.json

# 构建（Windows）
go build -tags "with_quic with_utls" -o sing-box-webui.exe .

# 仅配置管理、不跑实例
sing-box-webui.exe -no-run

# 可选鉴权
sing-box-webui.exe -secret your-token
```

前端开发（热更新）：

```bash
cd web && npm install && npm run dev   # http://localhost:5173，/api 代理到 8080
```

## API（详见 API.md）

```
GET  /api/status | /api/config | /api/schema | /api/types
PUT  /api/config
POST /api/reload | /api/instance/start | /api/instance/stop
GET/POST        /api/outbounds          /api/inbounds
GET/PUT/DELETE  /api/outbounds/{tag}    /api/inbounds/{tag}
GET/POST        /api/routes
PUT/DELETE      /api/routes/{id}
POST /api/tools/uuid | /api/tools/reality-keypair
```

所有写操作走统一管线：**修改内存 → 全量校验（解码+box.New 干跑）→ 原子写盘（.bak 备份）→ 实例热重载**；
校验失败不落盘、内存回滚；reload 失败保留旧实例。

## 已知限制

- 重载 = 实例重建（sing-box 原生语义），在途连接会断，非无缝热更新
- 反复 PUT 会规范化手写格式（注释丢失）
- 规则 id 是旁车元数据，外部手工改配置后 id 会重新生成
- Windows Smart App Control 可能拦截本地编译的未签名 exe；开发期用 `go run`，
  正式使用可在 Windows 安全中心关闭 SAC 或对二进制做代码签名

## 项目结构

```
main.go                  # 入口：flag + embed 前端 + 组装
internal/store/          # 配置存储：加载/校验管线/原子写/回滚/meta
internal/api/            # REST handlers（outbound/inbound/route/config/tools/instance）
internal/runner/         # sing-box 实例生命周期（close+recreate，失败回滚）
web/                     # Vue3 前端（构建产物 web/dist 被 embed）
API.md                   # API 契约
```
