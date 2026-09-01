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

## 四、 Config URL 远程配置与订阅管理

```
[Remote URL / Subscription] ──► [HTTP Fetch (ETag 校验)]
                                          │
                                          ▼
                               [sing-box check (Dry Run)]
                                    │             │
                             (校验失败)         (校验成功)
                                    │             │
                                    ▼             ▼
                           [保留旧配置+报警]    [备份当前配置 + 原子覆写 + Reload]
```

1. **拉取与缓存策略**：
   - 支持手动触发与定时自动轮询更新。
   - 请求头携带 `If-None-Match` (ETag) 与 `If-Modified-Since`，无变更时跳过下载节省流量。
2. **安全预检 (Dry-Run Check)**：
   - 配置写入前，Controller 调用 `sing-box check -c <temp_path>` 进行语法和规则校验。
   - 校验通过才执行替换；校验失败则保留旧配置，并将详细报错信息返回给 WebUI。
3. **原子替换与版本回滚**：
   - 采用临时文件 + `os.Rename` 原子覆盖目标配置文件。
   - 自动备份最近 N 份历史配置（`config.backup.<timestamp>.json`），支持 WebUI 一键回滚。
4. **平滑重载**：
   - 优先通过 SIGHUP 信号或 sing-box 内部热重载接口触发配置刷新；不支持热重载的模块则执行进程重启。

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
- **Monaco Editor 智能代码视图**：
  - WebUI 内置的 JSON / YAML 配置编辑器直接挂载 `/api/schema`。
  - 用户在编辑配置时获得 IDE 级别的字段自动补全（IntelliSense）、Hover 字段文档说明和实时语法/类型标红校验。

### 3. UI 表现层：“80/20 混合架构”
- **核心协议 (80% 常用场景)**：针对 VLESS、Hysteria2、Shadowsocks、VMess、Direct、Selector 等常用出入站协议，前端手绘精致易用的专属卡片与开关表单。
- **长尾/新增特性 (20% 扩展场景)**：
  - 通过 Schema 元数据动态渲染“高级设置 (Advanced)”折叠区。
  - 结合 JSON 源码视图一键切换，确保即使用户使用最新实验性字段，也能即时配置并安全落盘。
- **字段透传保护**：前端与后端交互时保证未知字段不被过滤抹除 (Open Struct Passthrough)。

---

## 六、 实施路线图 (Milestones)

### Phase 1: Schema 驱动与前端类型全自动流水线 (Schema & CodeGen)
- [ ] 接入 `json-schema-to-typescript` 工具链，建立 `pnpm gen:types` 自动化构建脚本。
- [ ] 将 `/api/schema` 挂载到 Monaco Editor，实现配置代码视图的智能补全与语法高亮校验。
- [ ] 确保前端表单提交保留未建模字段（透传保护）。

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
