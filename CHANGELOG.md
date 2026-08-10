# Changelog

## v0.8.0（2026-08-10）

### 新功能
- 日/夜主题切换：右上角 ☀️/🌙 切换，localStorage 持久化（默认暗色）；Element Plus dark css-vars 全组件适配；CodeMirror 用 Compartment 动态切换 oneDark / 亮色主题（语法高亮 + 编辑器 UI 外壳同步）
- 移动端响应式：<768px 侧边栏自动折叠为 64px 图标栏（菜单图标化）；汉堡按钮让侧边栏自身伸展为完整菜单（遮罩点击收起，选中路由自动收起）；窄屏隐藏 header 状态项
- 新建 inbound 的 listen 默认值实时从后端 settings 拉取（不再依赖 statusStore 异步时序），切换类型也重新填充

### 修复
- **CodeMirror 6 `Unrecognized extension value`（生产阻断）**：根因是 rollup 把 `@codemirror/state` 的 ESM/CJS 双入口解析成两个模块，同 chunk 内两份 state 核心代码（双实例）→ basicSetup 元素 instanceof 全挂。修复：vite `resolve.alias` 强制 `@codemirror/state` 指向单一 ESM 文件；移除 manualChunks（vite 自动共享即可）。headless 二分复现确认：单实例后 `Unrecognized` 特征 4→2，basicSetup create OK

### 工程
- 前端包管理器切换 pnpm（AGENTS.md 规范）：pnpm-workspace.yaml allowBuilds(esbuild/vue-demi)，CI 用 pnpm/action-setup@v4 + cache:pnpm
- 后端单元测试（`go test ./...`）：internal/api 覆盖 24.3%，40+ 用例——config/raw JSONC 注释往返、outbound/dns/rule_set/certificate 引用保护与删除回写（含 json.Unmarshal 复用不清零回归）、logical 规则语义（sing-box 1.14 嵌套规则禁 action）、端口分配、诊断计数
- 版本信息：`-version` flag + `version` 子命令（sing-box 风格）

## v0.7.1（2026-08-10）

- 规则集页（route.rule_set CRUD：inline/local/remote，format 从 url/path 推断 .srs→binary，引用保护 409+force）
- 证书页（certificate 段 + acme provider，dns01 服务商枚举）
- DNS/route 规则 logical 类型适配（and/or 嵌套 + action 共用 + 摘要）
- 源码 tab 下沉到所有编辑 dialog 内（ResourceSourceTab，dirty 时源码覆盖表单）+ 页面级 SourcePane
- 移除全部 rawJson/extraJson 附加字段（未覆盖字段统一走源码 tab）
- DNS server 表单全字段化（domain_resolver 子字段 + DialerOptions）
- 重载功能：POST /api/reload 三模式（systemd/pidfile/hook）+ 保存后自动重载 + 左下角悬浮按钮
- CodeMirror 多实例修复（manualChunks 提取共享 chunk，后续被 v0.8.0 的 alias 方案取代）

## v0.6.1（2026-08-08）

- JSONC 编辑器：自建 Lezer grammar + jsonc-parser（VSCode 同款格式化），注释/尾逗号保留
- GET/PUT /api/config/raw 原样读写
- 诊断页：endpoint 并入 outbound 命名空间 + rule_set 真实检查
- DNS 规则摘要全字段

## dev（未发版）

- 前端接入 Tailwind CSS v4（@tailwindcss/vite 插件）
- 导航栏激活指示条动效（借鉴 zashboard）：绝对定位指示条随路由切换弹性滑动
  （transform 0.55s cubic-bezier(0.22,1.5,0.36,1) + width/height 0.32s），激活项文字平滑变色；
  prefers-reduced-motion 自动降级
- 修复：router.isReady() 后再挂载应用（整页刷新 deep-link 时 el-menu 高亮丢失）
