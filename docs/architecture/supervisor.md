# 内核托管与重载机制

在不同的操作系统与部署拓扑中，`sing-controller` 支持多种内核生命周期管理模式。

---

## 重载模式 (Reload Modes)

在 `controller.json` 的 `core.reload` 节点中，可配置如下重载模式：

| 模式 (`mode`) | 说明 | 适用场景 |
| :--- | :--- | :--- |
| `auto` | 自动探测模式：Linux 优先使用 systemd / rc-service，Windows 使用内置 Supervisor 重启 | 大多数通用环境 |
| `systemd` | 调用 `systemctl reload-or-restart sing-box` | Linux 生产服务器双服务 |
| `procd` | 调用 `/etc/init.d/sing-box reload` | OpenWrt 路由器环境 |
| `rc-service` | 调用 `rc-service sing-box reload` | Alpine Linux / OpenRC |
| `pidfile` | 向 `pid_file` 记录的进程号发送 `SIGHUP` 信号 | 经典 Unix 守护进程 |
| `hook` | 执行自定义命令（如脚本执行） | 特殊编排与容器环境 |
| `none` | 不主动触发重载，仅落盘保存文件 | 外部由文件监听器（fsnotify）托管场景 |

---

## 自动保存后重载 (after_save)

当 `core.reload.after_save` 设为 `true` 时，任何通过 WebUI 或 REST API 成功修改代理配置的操作，在校验通过并原子落盘后，会自动立即触发重载。无需用户二次点击“重启”或“重载”按钮。

```json
{
  "core": {
    "reload": {
      "mode": "auto",
      "after_save": true
    }
  }
}
```
