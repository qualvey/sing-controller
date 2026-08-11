# Changelog

## v0.9.0（2026-08-10）

### 新功能
- **shadowsocks 入站控制**：Inbounds 页新增 shadowsocks 表单（第三类已实现表单，mixed/tuic 之后）——
  - 默认值：method `chacha20-ietf-poly1305`、listen `::`、端口 `23010`（与需求示例一致）
  - 密码自动生成：新建时自动调用后端 `POST /api/tools/password`（16 字节随机 → 标准 base64），失败浏览器端兜底；表单内可一键「重新生成」
  - method 九选一（none / aes-gcm 系 / chacha20 系 / 2022-blake3 系）；method 为 none 时密码字段禁用、校验跳过
  - 用户池绑定（Users 页统一管理，name+password 注入 users[]；非 2022 method 以用户密码为准，2022 method 以顶部 password 为主密钥）
  - 列表新增 method 列
- 后端新增 `POST /api/tools/password` 工具接口

### 修复
- **编辑弹窗「源码」tab 永远指向第一个项目的源码**：ResourceSourceTab 的 initial watch 在首次程序化替换时被自己的 updateListener 标记 dirty（docChanged），后续切换编辑对象时因 dirty=true 跳过刷新；且保存时 `isDirty()` 为真，会拿上一项的源码覆盖当前项（静默数据损坏）。修复：initial 变更时强制重置 dirty 并刷新内容，程序化 dispatch 期间抑制 dirty 标记（同时修复 inbound/outbound/dns/certificate 四个使用方）
- **handleDeleteInbound 无 route 段空指针崩溃**（配置缺少 route 段时删除入站 panic；handleDeleteOutbound 已有同样防护，补齐）
- handleStatus 无 route 段空指针（`rules` 计数防护）

### 工程
- 后端单测新增：密码工具格式/随机性、shadowsocks 入站 CRUD 全链路（创建/回读/更新/非法 method/缺密码/none/用户池绑定投影/删除）

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

- **clash API 反向代理**：controller 新增 `/api/clash/*` 代理 → sing-box
  `experimental.clash_api`（自动注入 secret，前端同源访问、secret 不落浏览器）
- settings 新增 `clash_api` 段（address/secret，可选；默认从 sing-box 配置推断）
- 前端移植 zashboard 的 clash API 客户端（`src/api/clash.ts`，MIT）：
  proxies/select/延迟/规则/configs/DNS 查询/fakeip flush + connections/logs/traffic WS 流
- 修复 store.Load 真实 bug：无 route 段的配置空指针崩溃

- **service API（gRPC-Web）反向代理**：`/api/grpc/*` → sing-box 顶层
  `services[type=api]`（自动推断 listen/listen_port/secret；settings 可显式覆盖）
  ——sing-box 1.14 原生支持 gRPC-Web + grpc-websockets 子协议（源码确认），
  纯 HTTP 反代即可，无需协议转换；事件流式推送（连接/日志/状态/代理组/出站）
- 前端 singbox 客户端（移植 zashboard）：@connectrpc/connect-web + protobuf 生成代码
  （src/gen/）+ src/api/singbox.ts（订阅流 + selectOutbound/closeConnection 等操作）
- clash/service 代理泛化为 proxyCache（前缀 + 签名重建）
