# sing-box-webui 开发规范与最佳实践指南 (Development Guidelines)

> **版本**：v1.0  
> **适用范围**：`sing-box-webui`、`sing-box-controller` 及前端 WebUI 全体模块  
> **核心宗旨**：保持系统高内聚低耦合、保障配置绝对安全性、维持单二进制交付体验、实施高标准的工程化代码质量。

---

## 目录
- [1. 核心架构与设计原则](#1-核心架构与设计原则)
- [2. Git 版本控制与提交规范](#2-git-版本控制与提交规范)
- [3. 后端 Go 开发规范](#3-后端-go-开发规范)
- [4. 前端 Vue 3 + TypeScript 开发规范](#4-前端-vue-3--typescript-开发规范)
- [5. 接口设计与协议兼容规范](#5-接口设计与协议兼容规范)
- [6. 文档与维护规范](#6-文档与维护规范)
- [7. 开发与自测检查清单 (Checklist)](#7-开发与自测检查清单-checklist)

---

## 1. 核心架构与设计原则

1. **单二进制交付 (Single Binary Delivery)**：
   - 前端 WebUI 构建产物嵌入 Go 后端（`go:embed`），前后端同端口同进程运行。
   - 当前端未构建时，自动降级为 API-only 模式，保障核心能力不中断。
2. **配置安全第一 (Config Safety First)**：
   - 任何配置写入均需通过“严格解码 -> 静态诊断 -> 内存干跑预检 -> 原子落盘”四重管线。
   - 杜绝脏配置落盘导致 sing-box 崩溃。
3. **协议优先度原则 (Protocol Priority)**：
   - **sing-box gRPC API 为第一公民**，优先保证 sing-box 原生控制面的完整可用性与性能。
   - Clash API 定位为兼容层可选方案。
4. **高内聚、低耦合与关注点分离**：
   - 前后端分层明确，数据单向流动，杜绝跨层污染。

---

## 2. Git 版本控制与提交规范

### 2.1 原子化提交 (Atomic Commits - 强制要求)
- **最小可行粒度**：每完成一个独立的小功能、一次明确的重构、一个 bug 修复，必须**立即做一个独立的 commit**。
- **禁止大杂烩**：严禁将不相关的改动（如“修复节点编辑 bug” + “优化 DNS 样式” + “修改后端 API”）合并在一个 commit 中。
- **构建保证**：每一个 commit 必须是一个可编译、测试通过的状态。

### 2.2 Commit Message 格式
遵循 Conventional Commits 规范：
```text
<type>(<scope>): <subject>

[optional body]
```

- **Type 类型**：
  - `feat`: 新增业务功能
  - `fix`: 修复 bug
  - `refactor`: 代码重构（无功能变更）
  - `perf`: 性能优化
  - `style`: 代码格式/样式微调（不影响功能逻辑）
  - `test`: 新增或修改单元测试
  - `docs`: 文档变更
  - `chore`: 构建流程、依赖更新或配置辅助工具变更
- **Scope 范围**（可选）：
  - 后端：`api`, `store`, `supervisor`, `collector`, `models`, `realtime`
  - 前端：`proxies`, `outbounds`, `inbounds`, `routes`, `dns`, `i18n`, `ui`, `settings`
- **示例**：
  ```bash
  feat(outbounds): 支持 VLESS Reality 密钥对生成
  fix(store): 修复多态类型反序列化缺少 include.Context 问题
  refactor(views): 抽离 OutboundEditDialog 为独立组件
  test(api): 补充 route rule_set 引用保护测试用例
  ```

---

## 3. 后端 Go 开发规范

### 3.1 目录职责分层 (`controller/`)
```
controller/
├── main.go               # 应用入口：参数解析、配置加载、模块依赖注入与生命周期管理
├── webui.go              # go:embed 静态资源嵌入与 SPA 路由 fallback 适配
└── internal/
    ├── api/              # REST / HTTP 处理器：入参解析、状态码返回、路由分发
    ├── store/            # 配置核心存储：JSON 解析、校验管线、备份与落盘
    ├── supervisor/       # sing-box 进程监管、重载适配（systemd/openrc/procd）
    ├── collector/        # 监控指标、节点延迟与状态采集器
    ├── realtime/         # WebSocket / SSE 实时通信层
    ├── provider/         # 外部资源（如订阅/证书提供商）拉取与转换
    └── models/           # 领域数据结构与 DTO 定义
```

### 3.2 依赖管理与 sing-box 源码协同
- **必须通过 `replace` 引用 sing-box fork**：
  `controller/go.mod` 中必须使用 `replace github.com/sagernet/sing-box => ../../sing-box`（或对应本地路径），**禁止**使用 `go.work`（防止 MVS 抬升 quic-go 等依赖导致接口不兼容）。
- **强制注入 Context**：
  所有反序列化与配置校验入口（`store.Parse` 等）必须强制包装 `include.Context(ctx)`，确保 inbounds/outbounds/services 等多态类型的正确解码。
- **构建 Tags 对齐**：
  编译和测试必须带全标准 tags：
  ```bash
  -tags "with_quic with_utls with_gvisor with_dhcp with_wireguard with_acme with_clash_api with_tailscale with_ccm with_ocm with_cloudflared with_usbip"
  ```

### 3.3 配置安全与校验管线规范
任何配置写操作（Create / Update / Delete / Import）必须严格遵循以下四步：
1. **严格反序列化 (Strict Decode)**：拦截未知字段、语法错误与多态字段异常。
2. **静态规则诊断 (Static Diagnosis)**：检查重复 Tag、悬空引用（Route/DNS 引用不存在的 Outbound/Server）、端口监听冲突。
3. **干跑预检 (Dry-run Validation)**：调用 `box.New(...)` 在内存中完整构建实例以做深层校验，不启动监听。
4. **原子写盘与备份 (Atomic Write & Rollback)**：先写入 `.tmp` 校验，再重命名替换，生成 `.bak` 备份文件；任一步骤失败立即回滚内存状态并抛出错误。

### 3.4 API 错误处理与响应规范
- **RESTful 状态码**：
  - `200 OK` / `201 Created`：请求成功
  - `400 Bad Request`：参数解析失败、JSON 格式错误
  - `404 Not Found`：资源不存在（如指定的 tag 未找到）
  - `409 Conflict`：**引用保护拦截**（例如试图删除被 Route 规则引用的 Outbound）；支持附带 `?force=true` 参数进行级联清理。
  - `500 Internal Server Error`：服务端不可恢复错误
- **错误统一输出**：
  禁止直接向客户端暴露未封装的底层 raw error，统一输出规范的 JSON 错误信息：`{"error": "具体错误原因"}`。
- **并发控制**：配置读写必须受 `sync.RWMutex` 或状态机严格保护。

### 3.5 单元测试规范
- **覆盖核心逻辑**：所有 `internal/api` 和 `internal/store` 模块必须具备单元测试。
- **关键测试场景**：
  - 配置的序列化与反序列化双向往返（Round-trip）；
  - JSONC 注释保留机制与格式化验证；
  - 引用保护（409 拦截）与强制级联删除逻辑；
  - 自动端口分配（`min_port`）冲突与可用性计算。

---

## 4. 前端 Vue 3 + TypeScript 开发规范

### 4.1 架构分层（单向依赖流动）
严格遵守以下依赖流向，禁止反向调用：
`Views` ➔ `Components` ➔ `Composables` ➔ `Assembly / Store` ➔ `API / Helper`

```
┌────────────────────────────────────────────────────────┐
│ Views: src/views/* (轻量组合页面、顶层控制栏与弹窗挂载)  │
└───────────────────────────┬────────────────────────────┘
                            │
┌───────────────────────────▼────────────────────────────┐
│ Components: src/components/* (按领域隔离的 UI 组件)     │
└───────────────────────────┬────────────────────────────┘
                            │
┌───────────────────────────▼────────────────────────────┐
│ Composables: src/composables/* (无头逻辑、视口计算、拖拽)│
└───────────────────────────┬────────────────────────────┘
                            │
┌───────────────────────────▼────────────────────────────┐
│ Assembly & Store: src/assembly/*, src/store/* (门面状态)│
└───────────────────────────┬────────────────────────────┘
                            │
┌───────────────────────────▼────────────────────────────┐
│ API & Helper: src/api/*, src/helper/* (纯请求与工具)   │
└────────────────────────────────────────────────────────┘
```

### 4.2 组件解耦与单文件守则
1. **杜绝巨石组件**：单文件超过 500 行应考虑拆分，复杂表单与弹窗必须独立为 `*Dialog.vue` 或 `*Modal.vue`。
2. **Teleport 挂载规范**：
   所有浮层、下拉菜单、右键菜单必须使用 `<Teleport to="#app-content">` 或 `<Teleport to="body">` 挂载，杜绝父级 `overflow: hidden`、`z-index` 或 `transform` 导致的定位偏移与视口裁剪。
3. **统一选择器规范 (Select / Dropdown)**：
   - 统一遵循 `src/components/common/SelectInput.vue` 体验；
   - 动画统一使用 `<Transition name="floating-menu">`；
   - 浮层面板统一绑定 `class="floating-menu-panel"`；
   - 包含键盘无障碍属性（`role="listbox"` / `aria-selected` 等）。
4. **FLIP 动画容器约束**：
   在 `TransitionGroup` 下的卡片（如 `ProxyNodeCard.vue`），根元素必须保持为**单一真实 DOM 元素**，严禁多根节点或 Fragment，避免动画撕裂。

### 4.3 国际化 (i18n) 强规范
1. **严禁硬编码**：所有界面文字、按钮、Placeholder、Tooltip、Toast 提示、确认弹窗文案**严禁在代码中写死字符串**。
2. **英文优先 (`en.ts`)**：
   - **`src/i18n/en.ts` 为第一优先语言**，新增 key 必须首先在 `en.ts` 中声明；
   - 中文（`src/i18n/zh.ts` / `zh-tw.ts`）必须同步维护对应的词条翻译。
3. **标准调用**：
   - 模板内：`{{ $t('common.save') }}` 或 `:placeholder="$t('rules.search')"`
   - Script Setup：`const { t } = useI18n(); t('common.confirm')`
   - 工具/纯 TS 文件：`import { i18n } from '@/i18n'; i18n.global.t('...')`

### 4.4 TypeScript 强类型规范
- **禁止使用 `any`**：必须为所有的 Props、Emits、API 返回值、Form 状态显式声明类型接口。
- **未知类型处理**：第三方动态数据临时过渡必须使用 `unknown` 并配合类型守卫（Type Narrowing）。
- **统一导入路径**：使用 `@/` 别名引用 `src/` 下模块。

### 4.5 样式与主题规范
- **语义化类名**：使用 Tailwind CSS + DaisyUI 语义化 token（如 `bg-base-100`, `bg-base-200`, `text-primary`, `border-base-300`）。
- **禁止硬编码颜色**：不得写死 `#1f2937` 等 hex 颜色值，确保亮色 (Light) / 暗色 (Dark) 主题下自适应切换。

---

## 5. 接口设计与协议兼容规范

1. **sing-box gRPC API 为首要目标**：
   - 所有核心代理功能（出站控制、连接观测、规则测试、实时监控）优先对接 sing-box 官方 gRPC / Connect-Web 接口。
2. **双后端抹平 (Facade / Assembly 模式)**：
   - 通过 `src/assembly/` 提供统一业务抽象。
   - 使用能力探针（`can('...')`）进行特性检测，禁止在 UI 组件内直接写 `if (backend === 'clash')` 的耦合逻辑。
3. **接口扩展流程**：
   - 新增或调整后端 HTTP/gRPC API 时，必须同步更新根目录下的 `API.md`。

---

## 6. 文档与维护规范

1. **API 契约文档 (`API.md`)**：
   每次新增/修改 REST API 端点、请求载荷或响应结构，必须同步修订 `API.md`。
2. **版本变更记录 (`CHANGELOG.md`)**：
   有功能发布、重大 Bug 修复或破坏性变更时，必须在 `CHANGELOG.md` 中以清晰条目记录。
3. **代码注释与文档完整性**：
   - 复杂算法、配置映射转换与协议兼容降级处必须保留详尽的代码注释。
   - 重构时不得无故删除现存的有效注释。

---

## 7. 开发与自测检查清单 (Checklist)

在发起 PR 或完成开发阶段前，请逐一核对以下项目：

- [ ] **Git 提交**：是否每个最小变更均已独立原子化 commit？Commit message 是否规范？
- [ ] **国际化覆盖**：是否有新增硬编码字符串？`en.ts` 与 `zh.ts` 是否均已补齐对应 key？
- [ ] **TypeScript 类型检查**：运行 `vue-tsc --noEmit` 是否 0 报错？是否存在未处理的 `any`？
- [ ] **前端构建验证**：在 `controller/web` 目录下执行 `pnpm run build` 是否顺利通过？
- [ ] **后端单元测试**：在 `controller` 目录下执行 `go test -tags "..." ./...` 是否全绿？
- [ ] **配置安全回滚**：新配置修改操作是否具备校验与 `.bak` 备份机制？
- [ ] **主题与响应式**：暗色/亮色切换是否正常？移动端布局与 Teleport 弹窗定位是否正常？
