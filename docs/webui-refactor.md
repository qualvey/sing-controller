# WebUI 重构记录（对齐 zashboard 技术栈）

> 决策：2026-08-12 老板拍板 —— 渐进式重构，弃用 Element Plus，改用 daisyUI（zashboard 同款）。迁移期间新功能冻结。

## 目标架构

- 无重型 JS 组件库：Tailwind 4 + **daisyUI 5**（纯 CSS 组件）+ **@heroicons/vue** 图标
- 表格：@tanstack/vue-table（headless，自渲染）
- 表单校验：zod
- 消息/弹窗/提示：自写 toast（已建 `helper/toast.ts`）/ tippy.js
- 实时通信（ConnectRPC / REST / clash WS）与状态管理（Pinia）保持不变

## 阶段进度

| 阶段 | 内容 | 状态 |
|---|---|---|
| 0 | 修复侧边激活指示条错位 BUG（CSS 静态偏移与 transform 双重叠加） | ✅ 2026-08-12 |
| 1 | 布局+导航重写 + Tailwind 化：App.vue 脱离 element（原生 flex + router-link + 干净版滑动指示条） | ✅ 2026-08-12 |
| 2 | 通用组件迁移 + daisyUI 启用 + zashboard 基础设施复用 | ✅ 2026-08-12 |
| 3a | 简单 view 迁移：Settings / Users / Proxies / Config（表单+表格+弹窗样板） | ✅ 2026-08-12 |
| 3b | **技术方向修正**：老板拍板表单用最先进方案 → shadcn-vue（Reka UI + Tailwind + vee-validate + zod v3），daisyUI 展示层保留 | ✅ 2026-08-12 |
| 3c | **全部 view 迁移完成**：13/13 脱离 element（7 个复杂 view 迁移 shadcn-vue 表单体系）| ✅ 2026-08-12 |
| 4 | 收尾：element-plus 依赖已移除（主包 1.24MB→315KB）；tooltip/ESLint/测试待办 | ⏳ |
| 4 | 表格/弹窗/消息收尾：el-table → tanstack table；tooltip → tippy.js | ⏳ |
| 5 | 收尾：删除 element-plus 依赖、图标替换、补 ESLint+Prettier、产物瘦身验证 | ⏳ |

## 关键实现笔记

### 阶段 0：指示条错位根因
`sidebar-tab-indicator` CSS 静态定位 `left: 6px; top: 2px` 与 `transform: translate3d(tabRect - navRect, ...)` 叠加——transform 差值已包含菜单项 `margin: 2px 6px` 偏移，导致双重偏移（右偏 6px、下偏 2px）。修复：静态定位归零。

### 阶段 1：导航重写要点
- `el-menu`/`el-container` 全家 → 原生 `<aside>/<nav>/<router-link>`，激活态用 `route.path === item.path` 精确控制
- 指示条定位基准为 `.nav-item.active` rect - 容器 rect；导航项用 `gap` 而非 `margin`，无偏移歧义
- 图标：@element-plus/icons-vue → @heroicons/vue/24/outline（注：heroicons 无 RouteIcon，Routes 用 MapIcon）
- ElMessage → 自写 `helper/toast.ts`（全局单例，success/error/info/warning，样式在 style.css）
- 移动端折叠/遮罩、主题切换、重载 FAB、header 状态栏行为保持不变

### 阶段 1 收尾：App.vue 样式 Tailwind 化（老板反馈「为什么这么多 CSS」，2026-08-12）
- scoped CSS ~300 行 → 20 行（仅保留指示条弹性过渡曲线，其余全 Tailwind 工具类）；App.vue 504 → 262 行
- style.css 新增 `@custom-variant dark (&:where(.dark, .dark *))`（Tailwind class 暗色模式，跟随 html.dark）
- **修复旧 bug**：原 `.app-aside:not(.aside-expanded)` 折叠态选择器在桌面也命中 → 桌面菜单误居中；Tailwind 化后用 `menuCollapsed` 动态类正确控制（桌面左对齐/移动折叠居中）
- **踩坑记录**：Tailwind 化时误删 `nav-item` 结构性类名（syncTabIndicator 的 querySelector hook）→ 指示条不定位；排查一度误判为编译器/运行时版本问题（vite 预构建缓存 + HMR 失效 + 旧 dev server 占用端口三重干扰）。教训：**JS 依赖的类名是结构性契约，重构时必须保留**
- 回归：scripts/measure-indicator.mjs 扩至 10 场景全绿（新增桌面左对齐/移动折叠/移动展开验证）

### 阶段 1 修复：滚动容器下指示条错位（用户报障，2026-08-12）
- **症状**：侧边栏视口高度不足触发滚动条后，指示条错位，错位量 = scrollTop
- **排查**：Edge headless + puppeteer-core 定量复现。syncTabIndicator 计算的 transform 值全部正确（computed/inline 均为目标值），但 getBoundingClientRect 渲染位置 = transform - scrollTop → 指示条被滚动容器的合成器滚动偏移带着走
- **根因**：指示条是滚动容器（nav，overflow-y:auto）的绝对定位子元素，与菜单项**不在同一滚动上下文**；真实滚轮（合成器滚动）后，浏览器对绝对定位元素与滚动内容的偏移处理不一致 → 结构性缺陷，任何 JS 修正都治标不治本
- **修复（结构性）**：滚动容器改为外层 `.nav-scroll`，指示条与菜单项同处 `.nav-inner`（position:relative，滚动内容流内）。transform 差值在内容流内**恒定**（两者同步偏移），滚动无需监听重算，天然对齐。顺带删除了滚动监听/过渡启停/ will-change 等全部补丁（will-change:transform 也会触发同类合成层问题）
- **回归验证**：`scripts/measure-indicator.mjs`（puppeteer-core，devDep）覆盖 8 场景：初始/真实滚轮/点击切换/回滚/程序化滚动/深链刷新/无滚动，全部 dy=0

### 阶段 3b：表单换最先进方案 → shadcn-vue（老板拍板，2026-08-12）
- **选型**：shadcn-vue 模式（Reka UI/radix-vue 行为 + Tailwind 样式 + vee-validate + zod v3）；lucide 图标；CVA/clsx/tailwind-merge
- **主题统一**：style.css 重构为「shadcn 变量为主（--background/--primary/--destructive 等）+ daisyUI 映射同一色板（--color-base-100: var(--card) 等）」→ 两套组件视觉一致
- **已建组件**：ui/button、input、label、form（FormItem/Label/Control/Message + vee-validate 导出）、select（radix SelectTrigger/Content/Item）、switch、checkbox
- **踩坑记录**：
  1. zod v4 与 @vee-validate/zod 不兼容（peerDep 是 ^3.24）→ 降级 zod v3（3.25）
  2. **DialogWrapper Teleport 崩溃**：Teleport to="#app-content"（自身渲染子树内）在 vue 3.5.41 下打开即崩（emitsOptions null / nextSibling null）→ 改 to="body" + v-show 改 v-if（Teleport 内禁用 v-show）
  3. Input defineProps 重复声明 modelValue（defineModel 已自动声明）→ 组件实例异常
  4. radix SelectItem 不允许空字符串 value（「（无）」选项用 "none" 哨兵值）
  5. FormMessage 无错误时空渲染 → v-if="!!$slots.default"
- **样板**：UsersView 表单 → vee-validate + zod（name 必填 / uuid v4 格式 / uuid+password 至少一个 superRefine），实测：弹窗 ✓ 自动密码 ✓ zod 错误显示 ✓ 提交链路 ✓
- 验证：build ✓；指示条回归 ✓；/users 弹窗全链路无页面错误 ✓

### 阶段 3c：全部 view 迁移完成（老板指示「全部迁移，写完提交」，2026-08-12）
- 7 个复杂 view 全部迁移：Connections / RuleSets / Certificate / Routes / Outbounds / Inbounds / Dns
- 新增组件：ChipInput（替代 el-select multiple+allow-create，5 个 view 复用）、ui/tabs（radix）
- 模式：el-table→原生 table、el-form→grid 两列布局、el-dialog→DialogWrapper、el-tabs→radix Tabs、el-select→radix Select、多选→ChipInput、校验→手写 validateForm（动态字段表单场景 vee-validate 不适用）、ElMessageBox→confirmDialog 队列
- Outbounds/Inbounds 的「从 JSON 填充」（原 ElMessageBox.prompt）→ DialogWrapper + textarea 弹窗；Reality 私钥展示 → confirmDialog（仅展示一次）
- **element-plus 彻底移除**：main.ts 全局注册删除、style.css 清理、pnpm remove
- **主包瘦身：1,241KB → 315KB（gzip 408→109KB，-75%）**
- 验证：build ✓；13/13 页面 el- 引用 = 0、零页面错误；指示条 10 场景回归 ✓
- git：2 个原子提交（a60eb23 feat 迁移 / 44d4cc8 chore 移除依赖）
- **SettingsView**：纯表单样板——grid 两列 label 布局（替代 el-form label-width）、daisyUI input/select/toggle、`input+datalist` 替代 el-select allow-create filterable（目标组 tag）、分组标题替代 el-divider
- **UsersView**：原生 table（简单列表不引 tanstack）+ DialogWrapper 弹窗 + checkbox 多选绑定入站 + showConfirmDialog 删除确认（zashboard 复用件首次实战）
- **ProxiesView**：el-select/input/button/icon/empty 全换 daisyUI + heroicons；element CSS 变量批量替换为 Tailwind 色值并补 dark: 变体
- **ConfigView**：CodeMirror 编辑器页工具栏 el-button → btn（文件为 CRLF，用 PS 正则替换处理）
- 已迁移 6/13 view：Logs / Diagnostics / Settings / Users / Proxies / Config，el- 引用全 0
- **暂缓**（复杂表单）：Dns(158) / Inbounds(113) / Outbounds(93) / Routes(57) / Certificate(63) / RuleSets(43) / Connections(34)
- 验证：build ✓；指示条 10 场景回归 ✓；6 页面 el-=0 无页面错误 ✓

### 阶段 2：daisyUI 启用 + zashboard 复用 + 通用组件迁移（2026-08-12）
- **daisyUI 5 启用**：`@plugin "daisyui" { themes: false }` + CSS 变量映射到 element 配色（--color-primary=#409eff 等，light/dark 双套）→ 组件视觉与旧版无缝衔接，避免主题突变
- **从 zashboard 复用（MIT）**：helper/confirmDialog.ts（确认队列，替代 ElMessageBox）、composables/dialog.ts（弹窗计数，剥离 viewport 依赖）、DialogWrapper.vue（daisyUI modal，剥离 blurIntensity）、ConfirmDialogHost.vue（去掉 $t i18n 依赖）、SegmentedControl.vue（iOS 分段控件，替代 el-radio-group）、VisibilityToggle.vue
- **未复用**（与 zashboard 状态耦合过深）：CollapseCard/CtrlsBar/TextInput（依赖 zashboard store/tippy，后续按需）
- App.vue：app-body 加 id="app-content"（DialogWrapper teleport 目标）+ 挂载 ConfirmDialogHost
- **迁移完成**：SourcePane（el-button→btn）、LogsCard（element CSS 变量→Tailwind）、LogsView（SegmentedControl+daisyUI input/btn）、DiagnosticsView（btn/badge/alert）
- 验证：build ✓；指示条 10 场景回归 ✓；/logs /diagnostics 页面 el- 引用 = 0，无页面错误

### 部署
- 子路径反代：`VITE_BASE_URL=/webui/ pnpm run build`（vite base + 路由 + 三个 API 客户端统一走 `import.meta.env.BASE_URL`）

## 验收清单（每阶段）
- [x] `pnpm run build`（vue-tsc 类型检查 + 构建）通过
- [x] 该阶段文件无 `element-plus` / `el-` 引用残留
- [ ] 浏览器人工回归：明暗主题、移动端折叠、深链刷新
