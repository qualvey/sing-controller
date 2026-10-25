# WebUI 组件设计规范与页面快速拼装指南

> **适用范围**：`sing-box-webui` 管理面板所有功能页（Inbounds、DNS、Outbounds、Routes、Rules 等）。  
> **核心目标**：统一视觉语言与交互体验，提供高复用、开箱即用的“页面拼装积木”，杜绝重复造轮子与风格割裂。

---

## 1. 页面标准解剖结构 (Page Anatomy)

一个标准配置管理页面由以下四层结构拼装而成：

```
┌─────────────────────────────────────────────────────────────┐
│ 1. 顶栏控制器 (*Ctrl.vue): SegmentedControl + Search + Plus │
├─────────────────────────────────────────────────────────────┤
│ 2. 页面容器: p-3 + base-container + usePaddingForViews      │
│   ┌─────────────────────────────────────────────────────┐   │
│   │ 3. 核心列表/表格 (*Table.vue): 卡片 Grid 布局       │   │
│   │    - 列头: text-xs uppercase text-base-content/60   │   │
│   │    - 行: 整行 Hover 高亮 + 整行点击编辑              │   │
│   │    - 胶囊/徽章: ProtocolTag + PortBadge             │   │
│   │    - 高亮: HighlightText 联动顶部搜索                │   │
│   │    - 操作: 右对齐删除按钮 + Tooltip + 二次确认      │   │
│   └─────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│ 4. 独立弹窗挂载 (*Dialog.vue): 表单/源码双 Tab + JSON 填充  │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. 列表表格规范 (Table Specification)

杜绝使用带硬边框、嵌套厚重的原生 HTML `<table>`。统一采用**响应式 Grid 卡片**。

### 核心样式与交互标准
1. **外层容器**：`<div class="w-full"><div class="overflow-x-auto bg-base-100"><div class="min-w-[700px]">`
2. **表头定义**：
   - 栅格：`grid grid-cols-[...] px-4 sm:px-5 h-11 items-center border-b border-base-content/10 bg-base-200/50 select-none`
   - 文字：`text-xs font-semibold text-base-content/60 tracking-wider uppercase`
   - 最后一列为操作，靠右对齐：`text-right pr-2`
3. **行数据与交互**：
   - 容器：`divide-y divide-base-content/5`
   - 单行样式：`group grid grid-cols-[...] px-4 sm:px-5 h-13 sm:h-14 items-center hover:bg-base-200/50 transition-colors duration-150 cursor-pointer`
   - **整行点击**：`@click="openEditModal(row)"`（提升移动端与桌面端可操作面积）
4. **单元格视觉规范**：
   - **协议/类型**：使用 `<ProtocolTag :protocol="row.type" />`，大写彩色半透明药丸徽章。
   - **名称/标识**：`text-sm font-semibold text-base-content overflow-hidden text-ellipsis whitespace-nowrap pr-2`，搭配 `<HighlightText :text="row.tag" :filter="filter" />`。
   - **端口**：使用 `<PortBadge :port="row.port" />`，统一端口胶囊。
   - **代理/出站/所属**：使用主色半透明徽章 `bg-primary/10 text-primary border border-primary/20 px-2 py-0.5 rounded-md`。
   - **无数据占位**：统一用浅色短横线 `<span class="text-base-content/30 font-mono select-none">—</span>`。
5. **操作列**：
   - 使用 `@click.stop` 阻止整行点击事件冒泡。
   - 幽灵圆形按钮：`btn btn-ghost btn-circle btn-sm text-base-content/30 hover:text-error hover:bg-error/10 transition-colors`。
   - 悬浮使用 `useTooltip().showTip($event, $t('delete'))`。
   - 点击调用 `showConfirmDialog` 二次确认与引用保护检测。

---

## 3. 编辑弹窗规范 (Dialog Specification)

弹窗遵循 **表单 / 源码 双 Tab 协同** 规范，具备“从 JSON 填充（粘贴解析）”与高度自适应能力。

### 核心要求
1. **外壳**：`DialogWrapper` 配置 `box-class="max-w-[680px]"`，`:body-class="activeTab === 'source' ? 'overflow-hidden' : 'overflow-y-auto'"`。
2. **Tab 切换**：使用 `TabsRoot` + `TabsList` + `TabsTrigger`。
3. **从 JSON 填充**：顶部常驻 `btn btn-ghost btn-xs` 配 `ClipboardPasteIcon`，弹窗支持一键将外部 sing-box json 解析并填充至表单。
4. **表单网格排版**：
   - 栅格：`grid gap-x-4 gap-y-5`，`style="grid-template-columns: 130px minmax(0, 1fr)"`。
   - 字段标签：`text-base-content/70 pt-2 text-sm`，必填项附带 `<span class="text-destructive">*</span>`。
   - 控件规范：下拉使用 `SelectField`，开关使用 `Switch`，标签数组使用 `ChipInput`，输入框使用 `input input-bordered input-sm w-full`。
5. **表单与源码双向同步**：
   - 使用 `useResizeObserver` 动态跟踪表单内容高度，使源码 Tab 的编辑器高度自适应等高，切换平滑不跳动。
   - 切换到 `source` Tab 时，若源码未手动更改，自动根据当前表单最新数据序列化。
6. **底部操作栏必须固定 (Fixed Footer)**：
   - **必须使用 `<template #footer>` 插槽**！保证表单内容再长、上下滚动时，底部的取消、保存与删除操作栏始终固定在弹窗底部可见。

---

## 4. 常用拼装积木清单 (Building Blocks)

| 积木组件 / Composable | 路径 | 用途说明 |
| :--- | :--- | :--- |
| **`ProtocolTag`** | `@/components/common/ProtocolTag.vue` | 统一协议/类型彩色徽章（支持 Inbound/Outbound/DNS 所有协议类型） |
| **`PortBadge`** | `@/components/common/PortBadge.vue` | 统一端口胶囊展示 |
| **`HighlightText`** | `@/components/common/HighlightText.vue` | 搜索关键词字符级高亮（自动支持正则过滤） |
| **`DialogWrapper`** | `@/components/common/DialogWrapper.vue` | 模态弹窗统一外壳 |
| **`SelectField`** | `@/components/ui/select` | 现代化无原生边框下拉选择器 |
| **`Switch`** | `@/components/ui/switch` | 精致扁平开关 |
| **`ChipInput`** | `@/components/common/ChipInput.vue` | 标签多值录入组件（如 network_type） |
| **`SegmentedControl`** | `@/components/common/SegmentedControl.vue` | 顶部视图分段切换器（带计数徽标） |
| **`CtrlsBar`** | `@/components/common/CtrlsBar.vue` | 响应式顶栏控制条外壳 |
| **`useTooltip`** | `@/helper/tooltip` | 按钮悬浮提示 `showTip($event, text)` |
| **`showConfirmDialog`** | `@/helper/confirmDialog` | 统一样式的异步确认对话框（Promise 式调用） |
| **`usePaddingForViews`**| `@/composables/paddingViews` | 视图安全边距自适应，防止与浮动状态栏重叠 |
