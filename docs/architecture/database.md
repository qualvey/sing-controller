# 状态数据库与迁移注意事项

对于涉及持久化存储（如 SQLite / PostgreSQL / cache.db）的状态迁移，需要注意嵌入式数据库的并发和日志机制。

---

## SQLite WAL (Write-Ahead Logging) 机制注意事项

在 SQLite 开启 WAL 模式下运行时，工作目录不仅会生成主数据库文件 `data.db`，还会伴生：
- `data.db-wal`（预写日志文件）
- `data.db-shm`（共享内存索引文件）

> [!WARNING] 数据库手工备份与替换防坑准则
> 如果在手工替换或迁移数据库文件时**没有先停止正在运行的服务**，或者**遗留了旧的 `data.db-wal` 文件**：
> SQLite 进程启动时会优先从内存和旧的 WAL 文件重放并覆盖新拷入的 `data.db`，导致数据出现回滚或读取到初始化的默认数据。

### 正确迁移流程
1. 彻底停止相关服务进程：`systemctl stop sing-controller`
2. 同步清理伴生的 WAL/SHM 临时文件：
   ```bash
   rm -f /var/lib/sing-box/data.db-wal /var/lib/sing-box/data.db-shm
   ```
3. 拷贝新的 `data.db` 文件到位并确保文件属组正确。
4. 重启服务：`systemctl start sing-controller`
