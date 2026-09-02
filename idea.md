# Idea But Not Plan

地球页面，目前的旋转方向，并不能上下左右自由旋转，而是yi

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