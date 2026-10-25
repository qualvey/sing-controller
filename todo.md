# Sing-Box WebUI & Controller 任务清单 (TODO List)

> 本清单基于项目全面复盘与动静分离架构演进方案整理，跟踪核心缺陷修复、系统架构演进、前后端交互优化及跨平台分发生态。

---

## 📌 优先级看板概览

```
[P0: 核心缺陷修复] ────────► [P1: 动静分离架构] ────────► [P2: 前端与交互增强] ────────► [P3: 跨平台生态与解耦]
   (100% 已落地)                (核心已落地)                 (规划推进中)                   (长期演进)
```

---

## 一、 P0：核心缺陷修复与稳定性基石 (Core Stability & Bugs) - [x] 已全部落地

- [x] **修复并发读写数据竞争 (Data Race)**：
  - 将 [`store.Store.mu`](file:///c:/Users/Ryu/Documents/sing-box-webui/controller/internal/store/store.go) 升级为 `sync.RWMutex`；
  - 在 API 层所有 GET Handlers（`handleStatus`, `handleListInbounds`, `handleListOutbounds`, `handleListRoutes`, `handleGetCertificate`, `handleDiagnostics`, `handleGetDNS`, `handleListRuleSets`, `proxy` 等）加入 `RLock` 保护。
- [x] **修复流量统计虚增爆炸 (Traffic Counting)**：
  - 在 [`collector/collector.go`](file:///c:/Users/Ryu/Documents/sing-box-webui/controller/internal/collector/collector.go) 中维护 `lastStats` 状态；
  - 彻底解决 `reset_on_query: false` 默认配置下每次轮询将全量累计值重复累加至数据库的问题，改为基于差值计算真实周期增量（`delta = current - last`）。
- [x] **修复元数据快照丢失 (Snapshot Rollback Bug)**：
  - 重构 [`store.go`](file:///c:/Users/Ryu/Documents/sing-box-webui/controller/internal/store/store.go) 中的 `snapshot()` 与 `cloneMeta()`，使用 JSON 完整深拷贝 `Meta` 所有字段（`DNSRuleIDs`, `Users`, `DisabledRuleIDs` 等），防止配置校验失败回滚时重置数据。
- [x] **修复空指针崩溃隐患 (Nil Pointer Panic)**：
  - 修复 [`store.AlignMeta()`](file:///c:/Users/Ryu/Documents/sing-box-webui/controller/internal/store/store.go) 在主配置缺少 `route` 块时引发的空指针解引用崩溃。
- [x] **修复备份机制失效 (.bak Backup Logic)**：
  - 修复 `saveLocked` / `saveRawLocked`，保存前将磁盘现存旧文件真正备份为 `.bak`，而非覆盖写入新内容。
- [x] **修复 DisableTun 并发与校验死锁**：
  - 将 [`DisableTun`](file:///c:/Users/Ryu/Documents/sing-box-webui/controller/internal/provider/manager.go) 改写为统一走 `store.Update` 校验管线，具备原子写盘与快照回滚能力。
- [x] **增强 Supervisor 并发防护**：
  - 在 [`supervisor.Manager.Restart`](file:///c:/Users/Ryu/Documents/sing-box-webui/controller/internal/supervisor/manager.go) 中引入互斥防抖锁 `restartMu`，防止并发调用引起的进程生命周期混乱。

---

## 二、 P1：架构演进：动静分离与配置管道 (Clean Base + Dynamic Outbound)

- [x] **Base 配置与 Outbounds 职责剥离**：
  - `config.json`（或 `00-base.json`）永远只保留用户自建的基础配置（Inbound、Route 分流策略、DNS、Services），**绝不硬写入外部机场节点**；
  - 启动时自动执行 `cleanBaseConfig()`，对存量配置中的历史外部注入节点进行无缝“脱水净化”。
- [x] **独立动态出站池生成 (`10-outbounds.json`)**：
  - 由 [`provider/manager.go`](file:///c:/Users/Ryu/Documents/sing-box-webui/controller/internal/provider/manager.go) 中的 `CompileOutbounds()` 独立计算并原子生成；
  - 自动构建包含 `Proxy` (selector) 选择器、`Auto` (urltest) 测速组与启用的全部外部节点；
  - 切换方案或删除订阅源时幂等重绘，**彻底根除僵尸节点与残留问题**。
- [x] **Linux 标准服务与 Supervisor 托管运行模式适配**：
  - 原生适配 Linux 标准服务命令：`ExecStart=/usr/bin/sing-box -D /var/lib/sing-box -C /etc/sing-box run`；
  - Supervisor 托管模式下自动支持目录挂载 (`-C <dir>`) 与双文件挂载 (`-c base.json -c 10-outbounds.json`)。
- [x] **联合预检管道 (Multi-File Validation)**：
  - 引入 `store.ValidateMulti()`，在临时沙箱调用 `sing-box check -C <tmpDir>` 进行 Base 与 Outbound 的联合权威预检。
- [x] **前端节点呈现动静分栏 (UI Separation)**：
  - 在节点管理页清晰标明“自建出站 (Custom Outbounds)”与“外部节点池 (Provider Nodes)”，避免混淆。

---

## 三、 P2：交互体验与前端功能增强 (Frontend & UX)

- [x] **代码编辑器快捷键拦截 (CodeEditor Save Shortcut)**：
  - 监听 `Ctrl+S` / `Cmd+S` 快捷键，触发配置原子保存并提示，阻止浏览器默认的“保存网页为 HTML”行为。
- [x] **节点组右键上下文菜单增强 (Proxies Group Context Menu)**：
  - ProxiesPage -> Group 标签页右键支持“编辑分组”与“删除分组”，弹窗拉取当前所有节点与自建出站作为候选。
- [x] **出站组默认节点排序 (Set as Default Node)**：
  - 在 ProxiesGroup 内部节点卡片右键添加“设为默认”，确认后将该节点在配置数组中移至第一位（Default 选中项）。
- [x] **3D 地球视角记忆与性能优化**：
  - 记忆用户在 3D 地球上的旋转角度与缩放偏好，减少后台标签页切换时的 WebGL 渲染负载。

---

## 四、 P3：上游内核解耦与微内核架构 (Decoupling & Upstream Sync)

- [x] **外部内核优先校验模式 (External Check First)**：
  - 尽可能将配置校验权下沉至外挂的 `sing-box check` 进程，降低 Controller 对完整上游 Go Module 源码的强依赖。
- [ ] **动态 Schema 驱动表单 (JSON Schema Form)**：
  - 深度利用 `/api/schema` 接口，前端部分高级配置表单由 Schema 动态生成，当上游新增协议或字段时无需频繁手写 TS 类型与表单。
- [x] **时序流量日志定期聚合与清理**：
  - 引入每小时/每日时序流量聚合归档任务，定期归档/清理历史明细单条记录，防止 SQLite 数据库无限膨胀。

---

## 五、 P4：桌面端与跨平台分发生态 (Desktop & Packaging)

- [ ] **Wails 桌面端系统托盘深度集成 (Full System Tray)**：
  - 托盘右键菜单直接展示核心代理组与快速切换（如 Proxy 组快速选节点）；
  - 托盘一键切换代理模式（Rule / Global / Direct）；
  - 托盘图标悬停展示当前上下行实时速率与出网 IP。
- [ ] **系统虚拟网卡驱动一键排障 (TUN Helper)**：
  - Windows 桌面端检测到 TUN 权限不足或缺网卡驱动时，提供一键引导或安装向导脚本。
- [ ] **轻量 Linux 嵌入式版构建 (OpenWrt / Alpine)**：
  - 支持通过编译 tags（剔除 WebView / CGO）构建数十兆以内的超轻静态二进制，无缝适配路由器与小内存设备。