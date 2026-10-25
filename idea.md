# Idea But Not Plan

- [x] OverView 概览页控制面板独立卡片：重载配置、重启内核与方案联动（已落地）
- [x] 3D 地球自由轨道旋转交互优化：上下左右 360° 顺畅旋转与阻尼惯性（已落地）
- [x] proxiesPage-> GroupTab ->右键也有删除和编辑（拉取所有节点作为候选）
- [x] proxiesGroup内的节点，右键菜单添加"设为默认", 结果->config.json中将该项目至于数组的第一个
- [x] BackendConnectionError 组件，要有编辑配置入口（不然只能去ssh vim改配置了）（已落地）
- [/] 在线更新功能：
  - [x] Phase 1: 基础清洗与契约对接（清理 zashboard 遗留，官方 Releases API，SemVer 比对，暴露控制器版本与平台架构）
  - [ ] Phase 2: 全局更新弹窗与 Changelog Markdown 渲染（带下载加速源选择、Toast Action 唤起）
  - [ ] Phase 3: 后端一键更新引擎（二进制原子自替换、SHA256 校验、systemd/自重启与自动回滚）
> 💡 **已评审并落地为规划方案**：完整架构与实施方案请查阅 [plan.md](file:///C:/Users/Ryu/Documents/sing-box-webui/plan.md)。

## 和内核一起打包。

作为内核的启停控制器
接入config url功能。
直接对接内核源码还是编译后的二进制呢

linux:
systemd用一个service 还是两个
如果内核崩溃，panic关键日志能不能保留。

windwos：
不用service，由控制器管理生命周期

## 上游内核跟进

上游代码更新频繁，我们的后端对配置校验和序列化都依赖上游源码；
前端怎么高效快速的把字段、表单对接呢。