# WebUI 组件设计规范与页面快速拼装指南

> **适用范围**：`sing-box-webui` 管理面板所有功能页（Inbounds、DNS、Outbounds、Routes、Rules 等）。  
> **核心目标**：统一视觉语言与交互体验，提供高复用、开箱即用的“页面拼装积木”，杜绝重复造轮子与风格割裂。

---

## 1. 页面标准解剖结构 (Page Anatomy)

一个标准配置管理页面（如 [InboundsPage.vue](file:///c:/Users/Ryu/Documents/sing-box-webui/controller/web/src/views/InboundsPage.vue)、[DnsPage.vue](file:///c:/Users/Ryu/Documents/sing-box-webui/controller/web/src/views/DnsPage.vue)）由以下四层结构拼装而成：

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

### 页面骨架代码模板 (`views/MyPage.vue`)
```vue
<template>
  <div class="relative size-full overflow-x-hidden">
    <!-- 1. 顶部控制栏 -->
    <MyCtrl />

    <!-- 2. 主视口容器（自动留白适配状态栏） -->
    <div class="p-3" :style="padding">
      <!-- 加载中状态 -->
      <div
        v-if="loading && !items.length"
        class="text-base-content/50 py-16 text-center text-sm"
      >
        <span class="loading loading-spinner loading-md text-primary mb-2 block mx-auto" />
        {{ $t('loading') || '加载中…' }}
      </div>

      <template v-else>
        <!-- 列表/表格视图 -->
        <template v-if="activeTab === 'form'">
          <div class="base-container">
            <MyTable :items="renderItems" />
          </div>
        </template>

        <!-- 源码视图 -->
        <template v-else-if="activeTab === 'source'">
          <div class="base-container">
            <SourcePane segment="my_segment" @saved="fetchItems" />
          </div>
        </template>
      </template>
    </div>

    <!-- 3. 弹窗组件独立挂载在页面底部 -->
    <MyEditDialog />
  </div>
</template>

<script setup lang="ts">
import { usePaddingForViews } from '@/composables/paddingViews'
import { onMounted } from 'vue'

const { padding } = usePaddingForViews({ offsetTop: 0, offsetBottom: 8 })

onMounted(async () => {
  await fetchItems()
})
</script>
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
6. **空状态**：
   - `<div v-if="!items.length" class="py-16 text-center text-sm text-base-content/50">{{ $t('noDataTip') }}</div>`

### 表格代码模板 (`components/.../MyTable.vue`)
```vue
<template>
  <div class="w-full">
    <div class="overflow-x-auto bg-base-100">
      <div class="min-w-[700px]">
        <!-- 表头 -->
        <div class="grid grid-cols-[100px_180px_1fr_140px_60px] px-4 sm:px-5 h-11 items-center border-b border-base-content/10 bg-base-200/50 select-none">
          <span class="text-xs font-semibold text-base-content/60 tracking-wider uppercase">{{ $t('protocol') }}</span>
          <span class="text-xs font-semibold text-base-content/60 tracking-wider uppercase">{{ $t('name') }}</span>
          <span class="text-xs font-semibold text-base-content/60 tracking-wider uppercase">{{ $t('address') }}</span>
          <span class="text-xs font-semibold text-base-content/60 tracking-wider uppercase">{{ $t('detour') }}</span>
          <span class="text-xs font-semibold text-base-content/60 tracking-wider uppercase text-right pr-2">{{ $t('actions') }}</span>
        </div>

        <!-- 空状态 -->
        <div v-if="!items.length" class="py-16 text-center text-sm text-base-content/50">
          {{ $t('noDataTip') || '暂无匹配数据' }}
        </div>

        <!-- 数据行 -->
        <div v-else class="divide-y divide-base-content/5">
          <div
            v-for="row in items"
            :key="row.tag"
            class="group grid grid-cols-[100px_180px_1fr_140px_60px] px-4 sm:px-5 h-13 sm:h-14 items-center hover:bg-base-200/50 transition-colors duration-150 cursor-pointer"
            @click="openEditModal(row)"
          >
            <!-- 协议标签 -->
            <div>
              <ProtocolTag :protocol="row.type" />
            </div>

            <!-- 名称 + 搜索高亮 -->
            <div class="text-sm font-semibold text-base-content overflow-hidden text-ellipsis whitespace-nowrap pr-2">
              <HighlightText :text="row.tag" :filter="filter" />
            </div>

            <!-- 核心数据 + 端口 -->
            <div class="text-xs overflow-hidden text-ellipsis whitespace-nowrap pr-2 flex items-center gap-1.5 font-mono">
              <HighlightText :text="row.address" :filter="filter" />
              <PortBadge v-if="row.port" :port="row.port" />
            </div>

            <!-- 关联标签 -->
            <div class="text-xs overflow-hidden text-ellipsis whitespace-nowrap pr-2">
              <span v-if="row.detour" class="inline-flex items-center text-xs font-medium bg-primary/10 text-primary border border-primary/20 px-2 py-0.5 rounded-md">
                {{ row.detour }}
              </span>
              <span v-else class="text-base-content/30 font-mono select-none">—</span>
            </div>

            <!-- 操作按钮 -->
            <div class="flex items-center justify-end pr-1" @click.stop>
              <button
                type="button"
                class="btn btn-ghost btn-circle btn-sm text-base-content/30 hover:text-error hover:bg-error/10 transition-colors"
                @mouseenter="showTip($event, $t('delete'))"
                @click.stop="handleRemove(row)"
              >
                <TrashIcon class="size-4.5" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
```

---

## 3. 编辑弹窗规范 (Dialog Specification)

弹窗遵循 **表单 / 源码 双 Tab 协同** 规范，具备“从 JSON 填充（粘贴解析）”与高度自适应能力。

### 核心要求
1. **外壳**：`DialogWrapper` 配置 `box-class="max-w-[680px]"`，`:body-class="activeTab === 'source' ? 'overflow-hidden' : 'overflow-y-auto'"`。
2. **Tab 切换**：使用 `TabsRoot` + `TabsList` + `TabsTrigger`（已全面原生化，禁用 reka-ui / radix-vue）。
3. **从 JSON 填充**：顶部常驻 `btn btn-ghost btn-xs` 配 `ClipboardPasteIcon`，弹窗支持一键将外部 sing-box json 解析并填充至表单。
4. **表单网格排版**：
   - 栅格：`grid gap-x-4 gap-y-5`，`style="grid-template-columns: 130px minmax(0, 1fr)"`。
   - 字段标签：`text-base-content/70 pt-2 text-sm`，必填项附带 `<span class="text-destructive">*</span>`。
   - 控件规范：下拉使用 `SelectField`，开关使用 `Switch`，标签数组使用 `ChipInput`，输入框使用 `input input-bordered input-sm w-full`。
5. **区块分割线**：逻辑分块时使用整行网格跨度分割线：
   ```html
   <div class="flex items-center gap-2 text-[13px] font-semibold text-base-content/90" style="grid-column: 1 / -1">
     <span class="border-base-content/10 h-px flex-1 border-t" />
     区块标题
     <span class="border-base-content/10 h-px flex-1 border-t" />
   </div>
   ```
6. **表单与源码双向同步**：
   - 使用 `useResizeObserver` 动态跟踪表单内容高度，使源码 Tab 的 Monaco 编辑器高度自适应等高，切换平滑不跳动。
   - 切换到 `source` Tab 时，若源码未手动更改，自动根据当前表单最新数据序列化 `JSON.stringify(buildData(), null, 2)`。
   - 保存时：如果当前在 `source` Tab 或源码被修改，优先保存源码 JSON 内容。
7. **底部操作栏必须固定 (Fixed Footer)**：
   - **必须使用 `<template #footer>` 插槽**！`DialogWrapper` 的主体内容区独立滚动（`overflow-y-auto`），而 `#footer` 插槽位于内容区外部下方（`relative shrink-0 border-t bg-base-100/90 px-4 py-3`），保证表单内容再长、上下滚动时，底部的取消、保存与删除操作栏始终固定在弹窗底部可见，绝不可将操作栏放在默认插槽滚动体内。
   - 布局规范：`<div class="flex items-center justify-between w-full">`
     - **左侧**：若处于编辑模式（`isEdit`），显示危险删除按钮（`btn btn-ghost btn-sm text-error hover:bg-error/10`），点击时调用 `showConfirmDialog` 二次确认。
     - **右侧**：`取消` 按钮（`btn btn-ghost btn-sm`）+ `保存` 按钮（`btn btn-primary btn-sm :disabled="saving"`）。

---

## 4. 常用拼装积木清单 (Building Blocks)

| 积木组件 / Composable | 路径 | 用途说明 |
| :--- | :--- | :--- |
| **`ProtocolTag`** | `@/components/common/ProtocolTag.vue` | 统一协议/类型彩色徽章（支持 Inbound/Outbound/DNS 所有协议类型） |
| **`PortBadge`** | `@/components/common/PortBadge.vue` | 统一端口胶囊展示 |
| **`HighlightText`** | `@/components/common/HighlightText.vue` | 搜索关键词字符级高亮（自动支持正则过滤） |
| **`DialogWrapper`** | `@/components/common/DialogWrapper.vue` | 模态弹窗统一外壳 |
| **`ResourceSourceTab`** | `@/components/ResourceSourceTab.vue` | Monaco JSON 代码编辑器 Tab |
| **`SelectField`** | `@/components/ui/select` | 现代化无原生边框下拉选择器 |
| **`Switch`** | `@/components/ui/switch` | 精致扁平开关 |
| **`ChipInput`** | `@/components/common/ChipInput.vue` | 标签多值录入组件（如 network_type） |
| **`SegmentedControl`** | `@/components/common/SegmentedControl.vue` | 顶部视图分段切换器（带计数徽标） |
| **`CtrlsBar`** | `@/components/common/CtrlsBar.vue` | 响应式顶栏控制条外壳 |
| **`useTooltip`** | `@/helper/tooltip` | 按钮悬浮提示 `showTip($event, text)` |
| **`showConfirmDialog`** | `@/helper/confirmDialog` | 统一样式的异步确认对话框（Promise 式调用） |
| **`usePaddingForViews`**| `@/composables/paddingViews` | 视图安全边距自适应，防止与浮动状态栏重叠 |

---

## 5. 开发自查清单 (Checklist)

在新建或重构任何模块时，对照本清单核验：
- [ ] **表格**：是否采用 `grid` 布局与 `base-container`？
- [ ] **交互**：表格行是否支持整行悬浮高亮与整行点击打开编辑？
- [ ] **搜索**：名称与关键字段是否已接入 `HighlightText` 联动顶部 Filter？
- [ ] **徽章**：协议类型是否使用了 `ProtocolTag`，端口是否使用了 `PortBadge`？
- [ ] **操作**：删除操作是否有 Tooltip 提示，并经过 `showConfirmDialog` 二次确认？
- [ ] **弹窗**：是否具备“从 JSON 填充（粘贴解析）”入口？
- [ ] **固定 Footer**：保存/取消/删除按钮是否**严格放置在 `<template #footer>` 插槽**内固定展示，避免随表单滚动？
- [ ] **Tab 联动**：表单切换到源码时，是否已实现高度平滑自适应和数据双向同步？
- [ ] **国际化**：所有文本均使用 `$t(...)` 或 `t(...)`，无硬编码中文/英文字符串。

