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
pnpm install && pnpm run build
# 子路径反代部署时指定挂载路径（资源/路由/API 均带此前缀）：
# VITE_BASE_URL=/webui/ pnpm run build   # 默认 '/' 为根路径部署，无需设置

# 2. 启动 controller（页面 + API 同端口，默认 127.0.0.1:8080）
cd ..
go run -tags "with_quic with_utls with_gvisor with_dhcp with_wireguard with_acme with_clash_api with_tailscale with_ccm with_ocm with_cloudflared with_usbip" . -config config.json
# 浏览器打开 http://127.0.0.1:8080 即 webui
```

前端开发模式（HMR，`/api` 代理到 8080，controller 需同时运行）：

```bash
cd controller/web
pnpm run dev     # http://localhost:5173
```

> **构建 tags 对齐生产 sing-box 二进制**（with_quic/with_utls/with_gvisor/with_dhcp/with_wireguard/with_acme/with_clash_api/with_tailscale/with_ccm/with_ocm/with_cloudflared/with_usbip），
> 缺失时校验管线会报 `not included in this build`（如 DHCP）。naive outbound（cronet-go）需 cgo，交叉编译不支持，未启用。
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

- **Outbound CRUD**：vless+reality（utls 指纹、Reality 密钥对生成）、tuic v5 专属表单；**selector/urltest 组表单**（成员多选、default/url/interval/tolerance）；其他类型原始 JSON 兜底
- **Inbound CRUD**：mixed / tuic / shadowsocks 专属表单——mixed（users 动态行）、tuic（TLS 强制 + 证书 Provider + 用户池绑定）、shadowsocks（默认 method chacha20-ietf-poly1305 / listen :: / 端口 23010，密码自动生成 + 一键重新生成，用户池绑定）；预填默认 type/listen/端口，支持**自动分配最小可用端口**（min_port 起）
- **用户池（Users 页）**：全局用户多对多绑定入站（vless/vmess/trojan/tuic/hysteria2/hysteria/shadowsocks/anytls/shadowtls 共 9 类），保存时按入站类型自动投影注入各入站 users[]（用户池为唯一真相）；新建默认自动生成密码，uuid/密码均支持一键生成；解绑/删除自动同步所有关联入站
- **Route CRUD（rule→action 模型）**：action 支持出站 route（默认，outbound 字段）/ direct / bypass / reject / hijack-dns / sniff / resolve / route-options；**匹配字段完整覆盖 sing-box RawDefaultRule**（inbound/ip_version/network/protocol/domain 系列/geosite/rule_set/geoip/ip_cidr/port 系列/process/package/user/网络环境/invert 等 38 字段，从 sing-box 源码自动整理，前端元数据驱动渲染；复杂 map 字段走 JSON 兜底）；稳定 id 存旁车 meta；引用保护（删除被路由/组引用的对象会被拦截）
- **新建 outbound 自动并入 Proxy**（settings 默认开，可配目标 selector tag）
- **粘贴 JSON 解析**：表单粘贴 JSON → 后端解析校验 → 填充字段
- **Config 页使用 CodeMirror 6 编辑器（JSONC）**：支持 `//` 与 `/* */` 注释、尾逗号（自建 Lezer grammar，语法树完整 → 折叠/缩进不丢）；实时 lint 用微软 jsonc-parser（VSCode 同款引擎）；一键格式化/Ctrl+Shift+F = VSCode 同款格式化（保留注释）；保存兼容 JSONC 原文（注释忽略、尾逗号容忍）
- **DNS 管理页**：servers CRUD（local/udp/tcp/tls/https/quic/h3/fakeip/hosts/dhcp/mdns 多态 transport + detour/TLS 表单）、DNS 规则 CRUD（server 字段模型 + logical 嵌套类型，常用匹配字段 + 附加 JSON）、基础选项（final/strategy/timeout/cache）；删除引用保护（409 + force 确认）
- **规则集页（route.rule_set）**：inline/local/remote 三类型 CRUD（format 按扩展名推断 + 保存时显式写回，sing-box Marshal 丢 format 的兼容）；引用保护（409 + force 清除规则引用）
- **证书页**：certificate 段 + acme providers CRUD；被 tls.certificate_provider 引用保护
- **每个配置页带「源码」tab**：手动编辑对应段 JSON（JSONC），保存时与主配置合并（整段替换/写 null 删除，其余配置不变）——通用 SourcePane 组件（inbounds/outbounds/route/dns/route.rule_set/certificate）
- **配置诊断页**：静态分析（重复 tag、route/dns 悬空引用、组引用、监听冲突、未使用 outbound），补 sing-box 校验盲区（悬空引用 box.New 不拦）
- **完整校验管线**：所有写操作 = 严格解码（未知字段/多态/重复 tag 检查）→ `box.New` 干跑预检 → 原子写盘（.bak 备份）；失败不落盘、内存回滚
- **sing-box 重载（SIGHUP）**:`POST /api/reload`，默认 **auto 自动适配**——systemd（Debian/Ubuntu 等）→ openrc `rc-service`（Alpine）→ OpenWrt `service`/procd → SysV service，有什么用什么；也可显式 systemd / pidfile / hook / none；保存配置后自动重载（after_save）；全页面左下角悬浮重载按钮；配套 Alpine/OpenWrt 参考 init 脚本见 `packaging/openrc`、`packaging/openwrt`
- **日/夜主题切换**:右上角 ☀️/🌙 切换,localStorage 持久化（默认暗色）;Element Plus dark css-vars 全组件适配;CodeMirror 用 Compartment 动态切换 oneDark/亮色（含编辑器 UI 外壳）
- **移动端响应式**:<768px 侧边栏自动折叠为 64px 图标栏,汉堡按钮自身伸展为完整菜单（遮罩点击收起）,选中路由自动收起
- **后端单元测试**:`go test ./...`（internal/api 36.0% 覆盖,36 个测试函数/111 个用例:配置往返/引用保护/删除回写/端口分配/诊断计数/shadowsocks 入站 CRUD/密码工具;httptest 全链路 config/raw JSONC 注释保留）
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
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags "with_quic with_utls with_gvisor with_dhcp with_wireguard with_acme with_clash_api with_tailscale with_ccm with_ocm with_cloudflared with_usbip" -o sing-controller .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags "with_quic with_utls with_gvisor with_dhcp with_wireguard with_acme with_clash_api with_tailscale with_ccm with_ocm with_cloudflared with_usbip" -o sing-controller .
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -tags "with_quic with_utls with_gvisor with_dhcp with_wireguard with_acme with_clash_api with_tailscale with_ccm with_ocm with_cloudflared with_usbip" -o sing-controller .

# 发布（推送 tag 触发 .github/workflows/release.yml → GoReleaser → GitHub Release）
git tag v0.1.5 && git push origin v0.1.5

# 本地预览产物（不发布）
goreleaser release --snapshot --clean
```

CI 流程：checkout 主仓库 → clone `qualvey/sing-box` fork 到 `<workspace>/../sing-box`（与本地布局一致）
→ setup-go(1.26) + setup-node(22) → `pnpm install --frozen-lockfile && pnpm run build` → goreleaser（自动 embed dist）。

自动产物：linux amd64 / arm64 / armv7 的 `tar.gz` + **deb**（含 systemd 单元，安装后自动建用户、启停服务）。

**Alpine / OpenWrt**（无 deb 包）：参考 init 脚本在 `packaging/`——Alpine 用 `openrc/sing-box`（`rc-service sing-box reload`），
OpenWrt 用 `openwrt/sing-box.init`（`service sing-box reload`，procd）；装好后 controller 的 reload.mode 保持默认 `auto` 即可自动适配。

## Linux 部署（deb）

```bash
sudo dpkg -i sing-controller_<version>_linux_<arch>.deb
sudo systemctl status sing-controller
# 浏览器访问 http://<host>:8080（listen 可改 /etc/sing-controller/config.json 后 restart）
```

重载权限

```shell
sudo tee /etc/polkit-1/rules.d/50-sing-box.rules <<'EOF'
polkit.addRule(function(action, subject) {
  if (action.id == "org.freedesktop.systemd1.manage-units" &&
      subject.user == "sing-controller" &&
      action.lookup("unit") == "sing-box.service") {
    return polkit.Result.YES;
  }
});
EOF
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
*   /api/clash/* /api/grpc/*   clash API / service API（gRPC-Web）反向代理
GET  /api/status | /api/config | /api/config/raw | /api/schema | /api/types | /api/settings | /api/diagnostics
PUT  /api/config | /api/config/raw | /api/settings
GET  /api/ports/available?start=N
POST /api/tools/uuid | /api/tools/password | /api/tools/reality-keypair | /api/tools/parse-json | /api/reload
GET/POST        /api/outbounds          /api/inbounds          /api/routes          /api/users
GET/PUT/DELETE  /api/outbounds/{tag}    /api/inbounds/{tag}    /api/routes/{id}     /api/users/{name}
GET/POST        /api/dns/servers       /api/dns/rules        /api/rule-sets        /api/certificate/providers
PUT/DELETE      /api/dns/servers/{tag}  /api/dns/rules/{id}   /api/rule-sets/{id}   /api/certificate/providers/{id}
GET/PUT         /api/certificate       PUT /api/dns/options
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
packaging/               # 部署资源（systemd unit + postinstall/postremove；openrc/OpenWrt 参考 init 脚本）
.goreleaser.yaml         # 交叉编译 + deb 打包配置
```

## 未来展望

集群系统，一主多从。
主节点有面板，从节点只有后端。
从节点自动向主节点注册。
主节点通过api控制从节点。
通信方式参考komari

> 设计文档（评审稿，未实现）：
> - [需求评审](docs/requirement-review.md)
> - [技术栈与协议选型](docs/tech-stack-protocol.md)
> - [集群架构设计](docs/cluster-architecture.md)

## 增强

路由规则支持选择插入位置
所有的只读选择下拉菜单，可以用键盘模糊搜索

