# Architecture & Design Guidelines (zashboard / sing-box-webui)

本文档定义了本项目（基于 zashboard 架构）的核心架构设计模式、分层约定与开发规范。
任何 AI 助手或开发者在新增功能、重构或修复问题时，**必须严格遵守此规范，保持高内聚、低耦合与既有抽象的纯洁性**。

---

## 1. 核心架构分层 (Layered Architecture)

项目严格划分为以下职责清晰的层级，依赖关系必须保持**单向流动**（从上至下）：
`Views` ➔ `Components` ➔ `Composables` ➔ `Assembly / Store` ➔ `API / Network`

```
┌────────────────────────────────────────────────────────────────────────┐
│                              Views (页面组合层)                         │
│   src/views/* (仅负责拼装 Controls, Headers, Panels 与 Modal 弹窗容器)     │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
┌───────────────────────────────────▼────────────────────────────────────┐
│                           Components (表现与交互层)                     │
│   src/components/<domain>/* (按业务域隔离的 UI 组件，严格无复杂业务副作用)      │
│   src/components/common/*   (通用业务容器: CollapseCard, DialogWrapper 等)│
│   src/components/ui/*       (基础原子组件: Radix / DaisyUI 封装)         │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
┌───────────────────────────────────▼────────────────────────────────────┐
│                        Composables (无头交互逻辑层)                     │
│   src/composables/* (视口计算/滚动虚拟化/拖拽/过渡/右键菜单状态等无头逻辑)      │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
┌───────────────────────────────────▼────────────────────────────────────┐
│                    Assembly & Store (业务装配与状态层)                   │
│   src/assembly/* (业务门面与多后端适配: 节点流/测速调度/Provider 聚合)       │
│   src/store/*    (响应式持久化配置: settings, smart, theme, proxyFolders)   │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
┌───────────────────────────────────▼────────────────────────────────────┐
│                          API & Helper (基础设施层)                      │
│   src/api/*, src/api.ts (REST 客户端契约、HTTP 拦截与反向代理请求)         │
│   src/helper/* (纯工具函数: tooltip, confirmDialog, toast, dayjs, indexeddb)│
└────────────────────────────────────────────────────────────────────────┘
```

---

## 2. 各层开发规范与约束

### 2.1 API 层 (`src/api/`, `src/api.ts`)
- **纯粹性**：仅声明 HTTP 请求方法、Request/Response TypeScript 接口契约。
- **禁止**：严禁在 API 层引入 UI 状态、DOM 操作或业务状态副作用。

### 2.2 Assembly 层 (`src/assembly/`) —— 业务装配与门面
- **门面模式 (Facade Pattern)**：为双后端（如 Clash REST API 与 sing-box 流驱动模式）提供统一的门面访问接口（如 `proxyMap`, `fetchProxies()`, `proxyLatencyTest()`）。
- **能力探针 (Capability Probe)**：通过 `can('...')` 判断后端能力，抹平差异，严禁将特定后端的私有逻辑散落在 UI 组件中。
- **禁止**：严禁在此层直接操作 DOM、绑定 UI 事件或引入 CSS 样式。

### 2.3 Composables 层 (`src/composables/`) —— 无头逻辑 (Headless Logic)
- **关注点分离**：所有复杂的计算、过滤、视口监听、手势拖拽、滚动分页（如 `useRenderProxyList`, `useCalculateMaxProxies`, `usePaddingForViews`）必须抽取为独立的 Composable。
- **单一职责**：一个 Composable 只专注于一种交互逻辑，保持纯粹，便于单元测试与多处复用。
- **禁止**：严禁在 Composable 中直接耦合特定的视图标签或内联 HTML 结构。

### 2.4 Store 层 (`src/store/`) —— 响应式状态与持久化
- **持久化契约**：使用 `@vueuse/core` 的 `useStorage` / `useSessionStorage` 进行偏好设置落盘。
- **状态归属**：
  - 用户配置、外观、排序方式、过滤器等跨组件共享的配置归入 `store/`。
  - 临时会话数据、单次操作状态（如当前的右键菜单坐标）优先归入 Composable，避免滥用全局 Store。

### 2.5 Components 层 (`src/components/`) —— 表现组件
- **按领域分包 (Domain Separation)**：
  - 业务组件必须放置于对应的领域子目录（如 `proxies/`, `connections/`, `rules/`, `outbounds/`, `settings/`）。
  - 通用复合组件放入 `common/`（如 `CollapseCard.vue`, `VirtualTable.vue`, `DialogWrapper.vue`, `SelectInput.vue`）。
  - 原子 UI 放入 `ui/`（基于 Radix / Tailwind 封装）。
- **解耦与单文件精炼**：
  - **严禁**创建上千行的单文件巨大组件。复杂弹窗（如编辑表单）必须抽离为独立的 `*Dialog.vue` 或 `*Modal.vue`。
  - **Teleport 规范**：浮层、右键菜单、全局 Modal 统一使用 `<Teleport to="body">` 或 `<Teleport to="#app-content">` 渲染，防止受到父容器 `overflow: hidden`、`z-index` 或 CSS `transform`（如 `TransitionGroup`）的层级裁切与定位偏移。
  - **FLIP 动画友好**：在 `TransitionGroup` 容器下的卡片组件（如 `ProxyNodeCard.vue`），根元素必须保持为单一真实 DOM 元素，防止 `ResizeObserver` 报错或动画撕裂。
- **下拉菜单与选择器规范（强约束）**：
  - **所有下拉菜单/选择器类组件必须对齐 `src/components/common/SelectInput.vue` 的交互体验与视觉效果**。
  - 优先直接使用 `SelectInput.vue`，若需特殊扩展或定制，必须严格遵循以下实现：
    1. **Teleport 浮层**：必须通过 `<Teleport to="#app-content">` 挂载，杜绝父容器滚动/裁剪污染；
    2. **过渡动画**：统一使用 `<Transition name="floating-menu">` 动画；
    3. **面板样式**：浮层必须带有 `class="floating-menu-panel"` 与动态坐标样式；
    4. **分组渲染**：支持 `option.group` 区分分组，分组标题采用 `text-base-content/45 px-2 pt-2 pb-1 text-xs font-medium` 统一风格；
    5. **无障碍与交互**：具备 `role="listbox"` / `role="option"`、`:aria-selected`、`:aria-disabled`，支持键盘导航与外部点击自动关闭。

### 2.6 Views 层 (`src/views/`) —— 页面组合
- **轻量组合器**：View 仅负责组合顶层 Controls 栏、Header、Tab 切换以及挂载全局 Modal。
- **禁止**：严禁在 View 内部直接编写庞大的表单、冗长的表格列定义或深层业务逻辑。

---

## 3. UI 风格、交互与国际化规范

1. **样式与设计系统**：
   - 使用 Tailwind CSS + DaisyUI 语义化类名（如 `bg-base-200`, `text-primary`, `bg-base-300/50`），保持暗色/亮色主题与全局变量一致。
   - 避免使用非标准的硬编码颜色值（如 `#1a1a1a`），必须使用语义化色彩 token。
2. **国际化 (i18n)**：
   - 所有用户可见文本**严禁硬编码**，必须在 `src/i18n/` 中配置中英文对应翻译，并在组件中使用 `t('key')` 调用。
3. **反馈与提示机制**：
   - 文本截断使用 `src/helper/tooltip.ts` 的 `checkTruncation` / `showTip` 机制。
   - 用户操作提示统一走 `showToast()` 或 `showNotification()`，危险操作二次确认统一走 `showConfirmDialog()`。

---

## 4. 新功能开发指引（以 Proxies 页面功能扩展为例）

当需要在 `ProxiesPage` 或其他页面新增功能（如节点右键菜单、节点就地编辑）时，必须按以下步骤推进：

```
[1. 抽象通用组件/表单]
   将业务弹窗（如 Outbound 编辑表单）从 View 中剥离为 src/components/<domain>/<Feature>Dialog.vue

[2. 抽象交互 Composable]
   在 src/composables/use<Feature>.ts 中管理激活状态、上下文数据与坐标计算

[3. 组合通用浮层/菜单]
   在 src/components/<domain>/<Feature>ContextMenu.vue 中构建 Teleport 弹层

[4. 最小化侵入现有组件]
   在 ProxyNodeCard.vue 仅派发事件，在 ProxiesPage.vue 底部声明式挂载 Modal，保持主流程干净
```

---

## 5. 质量与代码守则 (Code Principles)

- **保持向后兼容**：不破坏既有 Clash API 与 sing-box 双后端兼容能力。
- **避免代码重复 (DRY)**：相同的数据转换与表单逻辑必须抽离复用，禁止多页面复制粘贴。
- **TypeScript 强类型**：所有 Props、Emits、API 返回值必须明确声明类型，避免使用 `any`。
- **小步验证**：每次修改后确保类型检查通过（`vue-tsc` 或 IDE 无报错），且不影响现有手势、动画与虚拟滚动性能。
