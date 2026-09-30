# 原子保存与校验管线

在网络代理服务中，错误的 JSON 配置可能导致主网络瘫痪，甚至导致远程 SSH 连接断开失联。
`sing-controller` 内部通过 `store.Store` 提供了强保障的防损校验管线。

---

## 校验管线步骤

在任何写操作（如 `PUT /api/config`、`PUT /api/config/raw`、`POST /api/profiles/:id/apply`）触发时：

```mermaid
sequenceDiagram
    participant C as 客户端 (WebUI/API)
    participant S as Store (内存快照)
    participant V as sing-box 校验引擎
    participant D as 文件系统 (Disk)
    participant R as Reload 重载器

    C->>S: 提交配置变更变更
    S->>S: 生成内存快照 (Snapshot)
    S->>S: 执行 Mutate (结构变更)
    S->>V: 序列化为 JSON 并调用 Validate(content)
    alt 校验失败 (语法/端口/依赖错误)
        V-->>S: 返回 Error
        S->>S: 内存状态自动回滚 (Restore Snapshot)
        S-->>C: 400 Bad Request (返回详细错误原因，绝不碰磁盘)
    else 校验成功
        V-->>S: 校验通过 (Valid)
        S->>D: 写入临时文件 (config.json.tmp)
        S->>D: 触发 POSIX 原子重命名 (Rename to config.json)
        S->>R: 根据 reload 配置触发内核热重载
        S-->>C: 200 OK {"saved": true, "reloaded": true}
    end
```

---

## 关键安全机制

1. **真实内核干跑**：不仅仅校验 JSON 语法，而是直接调用官方 `box.New` 校验逻辑，验证 outbound tag 引用完整性、端口冲突、路由规则语法等深度语义。
2. **原子覆盖**：避免进程写盘中断写出一半残缺的文件，通过 `.tmp` 临时文件 + 原子重命名保证要么完整覆盖，要么原封不动。
3. **备份文件保留**：保存时自动保留 `.bak` 历史快照，便于灾备与回滚。
