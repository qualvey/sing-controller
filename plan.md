# Sing-Box 内核集成与生命周期管理架构规划 (Plan)

> 本文档基于 [idea.md](file:///C:/Users/Ryu/Documents/sing-box-webui/idea.md) 的头脑风暴与技术评审结论整理，明确 sing-controller 与 sing-box 内核打包、进程生命周期托管、崩溃捕获及远程配置拉取的落地实施方案。

---

## 一、 核心架构设计：Supervisor（守护进程）模式

采用 **独立二进制 + Controller 充当 Supervisor (进程管理器)** 的方案。

```
+-------------------------------------------------------------+
|                      sing-controller                        |
|                                                             |
|  +---------------+    +------------------+    +-----------+ |
|  | WebUI / API   |    | Process Manager  |    | Config    | |
|  | (Dashboard)   |<-->| (Supervisor)     |<-->| Manager   | |
|  +---------------+    +------------------+    +-----------+ |
|                               | (stdin/stdout/stderr)       |
|                               v                             |
|                    +--------------------+                   |
|                    | sing-box (子进程)  |                   |
|                    +--------------------+                   |
+-------------------------------------------------------------+
```

### 为什么选择 Supervisor 模式（而非 Go 源码内嵌）
1. **强故障隔离**：内核发生 Panic 或崩溃时，Controller 与 WebUI 保持在线，用户可在界面直观查看崩溃堆栈并修改配置，避免直接失联。
2. **解耦与独立升级**：sing-box 内核可独立下载/替换/热更，降低编译依赖与 CGO 复杂度。
3. **分发打包友好**：可通过安装包或发布归档将 `sing-controller` 与 `sing-box` 打包在同一目录下分发。

---

## 二、 跨平台生命周期管理

### 1. Linux 平台：单 Service 模式
- **架构决策**：systemd / OpenRC / procd 仅配置单个 `sing-controller.service`，由 Controller 负责启动并管理 `sing-box` 子进程。
- **核心优势**：
  - 免去 Controller 调用 `systemctl` 所需的高级权限与 D-Bus/Polkit 配置。
  - 抹平各发行版（Debian/Alpine/OpenWrt/群晖）的 init 系统差异。
  - Controller 直接拥有子进程的文件描述符（File Descriptor），接管所有日志输出。

### 2. Windows 平台：进程托管与 Job Object 防孤儿进程
- **架构决策**：不强制依赖 Windows Service，Controller 直接以应用/托盘模式运行并管理内核子进程。
- **孤儿进程防护 (Job Object)**：
  - 创建 Windows **Job Object**，配置 `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` 标志。
  - 将 `sing-box.exe` 进程绑定到该 Job Object。当 Controller 异常退出、被任务管理器杀死或断电重启时，Windows 内核自动级联终止 `sing-box.exe`，彻底避免端口占用与 TUN 虚拟网卡残留。
- **优雅退出流程**：
  - Controller 停止时，先向 `sing-box` 发送 `SIGINT` / 控制台事件，给予超时等待（如 3 秒）让其平滑清理 TUN 路由和网络连接，超时后再强制 Terminate。

---

## 三、 异常监控与 Panic 关键日志持久化 (Crash Guard)

```
[sing-box stdout/stderr] 
       │
       ▼
[Pipe Reader (Controller)]
       ├────────► [Memory Ring Buffer (最新 2000 行)] ──► WebUI 实时查看
       ├────────► [sing-box.log (轮转文件)]
       │
[Process Exit (ExitCode != 0 / Panic)]
       ▼
[Crash Dump Generator] ──► 生成 logs/crash-<timestamp>.log ──► WebUI 崩溃弹窗提醒
```

1. **双写缓冲机制**：
   - **内存环形缓冲区 (Ring Buffer)**：在内存维护最近 2000 行日志，提供低延迟的 WebSocket 实时日志流推送。
   - **文件持久化**：日志异步写入磁盘文件，支持大小限制与轮转。
2. **Panic / 崩溃关键日志捕获**：
   - 监听 `cmd.Wait()` 退出状态。若退出码非零或检测到 Go Panic 标志字符（如 `panic:`、`runtime error:`、`SIGSEGV`）：
     - 将崩溃点前后完整日志截取并单独保存为 `logs/crash-<YYYYMMDD-HHmmss>.log`。
     - 记录崩溃事件与时间戳到内存状态。
3. **防雪崩与自愈限流 (Backoff Restart)**：
   - 连续崩溃保护（例如 10 秒内连续退出超过 3 次），暂停自动重启，状态置为 `Error` 并向前端推送错误告警，防止死循环占用 CPU/磁盘。

---

## 四、 配置组、多配置片段与远程订阅管理 (Config Profiles & Multi-Config Merging)

### 1. 架构定位：全量基础配置 + 节点片段模块化

sing-box 原生支持多配置文件合并（`-c file1.json -c file2.json` 或 `-C <conf_dir>`）。利用该能力，我们将传统的单一大配置解耦为 **“基础框架 (Base)”** 与 **“节点/规则片段 (Fragments / Providers)”**：

```
+-------------------------------------------------------------------------------+
|                             配置组 (Active Profile)                           |
|                                                                               |
|  [00-base.json]               [10-sub-airport.json]    [20-nodes-vps.json]   |
|  (Inbound/DNS/Route/Services) (机场 A 订阅节点池)       (自建 VPS 节点池)      |
|           │                            │                        │             |
|           └────────────────────────────┼────────────────────────┘             |
|                                        ▼                                      |
|                       Controller 编译合并 / 多文件挂载                         |
|                                        │                                      |
|                                        ▼                                      |
|                          [sing-box check (Dry Run 预检)]                      |
|                                   │             │                             |
|                            (校验失败)         (校验成功)                      |
|                                   │             │                             |
|                                   ▼             ▼                             |
|                          [保留旧方案+报警]   [原子切换 + Supervisor 平滑重载]  |
+-------------------------------------------------------------------------------+
```

### 2. 核心模块与概念设计
1. **基础全量配置 (Base Config)**：
   - 包含系统的基础框架：`log`, `dns`, `inbounds`（如 `mixed 2080`, `tun` 模式网卡）, `route` 基础分流骨架, `services` (gRPC API), `experimental` 等。
   - 基础配置稳定，用户可通过 WebUI 可视化设置或 JSONC 智能编辑器按需调整。
2. **节点配置片段与订阅源 (Node Providers & Subscriptions)**：
   - **远程订阅源**：支持机场订阅 URL（sing-box JSON / Clash YAML / Base64），具备 Cron 定时拉取与 ETag / Last-Modified 差量更新；
   - **自建节点片段**：用户在 WebUI 录入的私有节点池，独立落盘为 `nodes-custom.json`；
   - **动态策略组归并**：节点拉取/导入后，可根据正则筛选或标签规则，自动注入到 Base 配置中的 `selector`（如 `Proxy`、`Auto`）和 `urltest` 成员列表中。
3. **配置组 (Profile & Config Group)**：
   - 用户可创建和保存多个方案（如：`默认分流模式`、`游戏低延迟方案`、`出差备用模式`、`直连纯净模式`）；
   - 每个配置组记录：引用的 Base 配置文件 + 启用的若干订阅源/节点片段列表；
   - WebUI 顶部/侧边栏提供 **一键方案切换 (Profile Switcher)**，切换时自动完成预检与内核平滑重载。
4. **安全预检 (sing-box check) 与版本回滚**：
   - 任何配置合并或订阅更新后，均先调用 `sing-box check` 预检；
   - 预检通过才原子覆写并触发内核重载；失败则阻断并生成诊断日志；
   - 自动备份历史版本（`profiles/backup/<timestamp>/`），支持一键回滚。
---

## 五、 上游内核跟进与 Schema 驱动契约体系 (Zero-Maintenance Upstream Sync)

```
[sing-box 上游更新] ──► [go get -u sing-box]
                              │
            ┌─────────────────┴─────────────────┐
            ▼                                   ▼
    【后端 Go：零手工负担】             【前端 TS/WebUI：全自动生成】
   • option.Options 直接引用           • GET /api/schema (内置内核 Schema)
   • store.Validate 原生干跑校验                   │
   • schema.Generate 动态反射                      ▼
                                       • json-schema-to-typescript ──► src/types/gen/singbox.d.ts
                                       • Monaco Editor 挂载 Schema ──► 原生 IntelliSense / 悬浮文档 / 语法校验
                                       • 80/20 混合 UI 架构 (精选表单 + 动态高级插槽)
```

### 1. 后端：以内核源码为唯一信任源 (Single Source of Truth)
- **零重复维护**：完全直接引用 `github.com/sagernet/sing-box/option`，后端无需手写任何协议或字段的 struct 定义。
- **天然校验与序列化**：配置读写与干跑校验直接走 `store.Validate` / `box.New`，上游新增的任何协议与字段自动获得原生校验与存储支持。
- **Schema 动态提供**：后端内置 `schema.Generate`，通过 `GET /api/schema` 实时输出当前编译版本对应的权威 JSON Schema (Draft 2020-12)。

### 2. 前端：Schema 驱动的代码生成与开发流 (CodeGen Pipeline)
- **TypeScript 类型全自动生成**：
  - 配置 npm 脚本：`pnpm gen:types`。
  - 调用 `json-schema-to-typescript` 读取 `/api/schema`（或 `sing-box schema` 二进制输出），一键生成 `src/types/gen/singbox.d.ts`。
  - 内核升级后，一行命令全量刷新前端 interface、类型守卫与枚举，彻底消灭人工编写 TS 类型的失误与滞后。
- **CodeMirror / JSONC 智能编辑体系 (IntelliSense & Diagnostics)**：
  - **Schema 语义校验**：集成 Ajv 2020 与 AST 定位，实现不符合 Schema 规范字段的实时标红诊断。
  - **键名智能补全**：根据 AST 路径与兄弟节点（如 `type: "vless"`）动态联想当前协议专属的合法属性，并自动去重。
  - **枚举值联想**：在 `type`、`method`、`level`、`network` 等字段值位置，自动联想所有合法枚举项及说明。
  - **悬浮字段文档 (Hover Tooltips)**：鼠标悬停在属性名上时弹出悬浮窗，展示官方 `description` 文档、类型约束、默认值与必填标识。

### 3. UI 表现层：“80/20 混合架构”
- **核心协议 (80% 常用场景)**：针对 VLESS、Hysteria2、Shadowsocks、VMess、Direct、Selector 等常用出入站协议，前端手绘精致易用的专属卡片与开关表单。
- **长尾/新增特性 (20% 扩展场景)**：
  - 通过 Schema 元数据动态渲染“高级设置 (Advanced)”折叠区。
  - 结合 JSON 源码视图一键切换，确保即使用户使用最新实验性字段，也能即时配置并安全落盘。
- **字段透传保护**：前端与后端交互时保证未知字段不被过滤抹除 (Open Struct Passthrough)。

---

## 六、 实施路线图 (Milestones)

### Phase 1: Schema 驱动与前端类型全自动流水线 (Schema & CodeGen)
- [x] 接入 `json-schema-to-typescript` 工具链，建立 `pnpm gen:types` 自动化构建脚本并导出 `singbox.d.ts`。
- [x] 在 CodeMirror 中接入 `Ajv 2020` 与 AST 节点映射，实现配置代码视图的 Schema 语义校验。
- [x] 确保前端数据层全量重导出生成类型，并保证未知字段透传保护。
- [ ] 在 CodeMirror 中基于 `jsonc-parser` + `schema.json` 接入**键名/枚举值智能补全 (Autocomplete)** 与 **悬浮字段文档 (Hover Tooltips)**。

### Phase 2: 进程管理器与日志管道 (Supervisor Engine)
- [ ] 实现 Go `ProcessManager` 抽象，封装子进程启动、停止、重启与状态查询。
- [ ] 接入 stdout/stderr 管道，实现内存 `RingBuffer` 与日志广播。
- [ ] Windows 平台适配 Job Object 生命周期绑定（防孤儿进程）。

### Phase 3: 崩溃防护与自愈系统 (Crash Guard)
- [ ] 捕获非正常退出与 Panic 堆栈，自动生成独立 Crash 日志文件。
- [ ] 实现防频繁秒退的熔断/退避重试（Backoff）机制。
- [ ] WebUI 增加内核状态指示灯、崩溃诊断弹窗与 Crash 日志下载。

### Phase 4: Config URL 订阅与配置流转 (Subscription & Auto-Reload)
- [ ] 实现远程 URL 定时拉取器与 ETag 差量缓存。
- [ ] 集成 `sing-box check` 预检与配置原子更新流程。
- [ ] 配置文件历史版本备份与 WebUI 一键回滚。

### Phase 5: Wails 跨平台桌面端打包 (Desktop App with Wails)
- [ ] 引入 **Wails (Go + Webview)** 框架，将 Controller 作为应用主核心，直接嵌入 WebUI。
- [ ] 实现系统托盘 (System Tray) 支持：托盘菜单、快速切换代理模式/策略组、配置组一键切换。
- [ ] 封装系统级能力：开机自启、原生通知提示、系统代理一键设置 (System Proxy)。
- [ ] 桌面端跨平台打包流水线（Windows `.exe` / `.msi`、macOS `.app` / `.dmg`、Linux `.AppImage` / `.deb`）。

