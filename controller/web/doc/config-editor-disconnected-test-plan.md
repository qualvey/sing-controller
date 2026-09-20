# 配置编辑器：sing-box 核心断开测试设计

## 测试目标

验证 sing-box 核心停止、重启或恢复期间，设置页的 `config.json` 编辑器：

- 页面和 CodeMirror 编辑区始终渲染，不出现空白页或未捕获异常；
- 能加载已有配置，或至少显示可编辑的 `{}` 文档；
- 用户可以编辑、格式化和放弃修改；
- 核心断开时保存失败有明确提示，不能误报保存成功；
- 核心恢复后可以重新加载并保存配置。

## 前置条件

1. 启动 sing-controller，准备一个合法的 sing-box `config.json`。
2. 打开前端 **设置 → 配置编辑器**。
3. 浏览器开启 Console，并记录未捕获异常、网络请求和页面截图。
4. 测试过程中只停止 sing-box 核心，不停止 sing-controller；另执行一次 controller 不可达场景以区分两类故障。

## 核心测试用例

| 编号 | 场景与步骤 | 预期结果 |
| --- | --- | --- |
| CE-01 | 核心在线，进入配置编辑器 | 显示编辑器容器、CodeMirror、配置文本和 `JSONC / 语法正常`；无 Console error |
| CE-02 | 进入编辑器后停止核心，点击“刷新” | 页面仍可见；接口失败只显示错误提示，编辑器不被卸载；原文本或 `{}` 仍可编辑 |
| CE-03 | 先停止核心，再进入设置页配置编辑器 | 编辑器骨架正常渲染；`config/raw` 返回空值/错误时显示 `{}` 或 controller 回退配置，不出现白屏 |
| CE-04 | 核心断开时在编辑器追加合法字段，点击“保存配置” | 编辑内容保留；显示保存/校验失败提示；不显示“保存成功”；保存按钮最终恢复可用 |
| CE-05 | 核心断开时输入非法 JSONC，点击保存 | 显示格式错误；不发起保存请求；文本和错误标记仍可见 |
| CE-06 | 核心断开时点击“放弃修改” | 重新加载失败也不导致白屏；保留原文或回退到 `{}`，dirty 状态清除 |
| CE-07 | 核心恢复，点击“刷新” | 加载恢复后的配置，路径和内容更新，dirty 清除；无重复编辑器实例 |
| CE-08 | 核心恢复后修改配置并保存 | 发出一次 `PUT /api/config/raw`；显示成功提示；随后状态刷新成功 |
| CE-09 | 配置页与其他设置分类来回切换，再重复 CE-02/CE-07 | 每次只有一个 CodeMirror 实例；无重复请求、内存泄漏或渲染异常 |
| CE-10 | controller 整体不可达（关闭 controller）后进入页面 | 与核心断开区分：显示后端连接错误；页面壳和编辑器仍可见，恢复 controller 后刷新可重试 |

## 自动化建议

使用浏览器 E2E（Puppeteer/Playwright）拦截 `/api/config/raw` 和 `/api/config`：

```ts
await page.route('**/api/config/raw', route => route.abort('failed'))
await page.goto(`${baseUrl}/#/settings?section=config`)

await expect(page.locator('.cm-editor')).toBeVisible()
await expect(page.locator('.cm-content')).toContainText('{}')
await expect(page.locator('text=保存配置')).toBeVisible()
```

随后将 `config/raw` 从失败切换为合法 JSON，点击“刷新”，断言编辑器文本、请求次数和 `dirty` 状态。保存场景应分别模拟：

- `PUT` 网络失败/HTTP 500：错误提示，不能出现成功提示；
- `PUT` 返回 `{ load_error: ... }`：警告提示，文本仍保留；
- `PUT` 返回成功：成功提示且调用一次状态刷新。

## 通过标准

- CE-01～CE-10 全部通过；
- 核心断开期间无 `TypeError`、`Uncaught (in promise)` 或 CodeMirror 初始化异常；
- 不以 sing-box 核心状态决定编辑器是否挂载，核心状态只影响 API 结果和提示；
- 恢复后无需刷新整个浏览器即可完成重新加载和保存。
