# Windows 桌面与托管模式

在 Windows 环境下，`sing-controller` 默认启用 **托管模式 (Managed Mode)**。

---

## 托管模式工作原理

在 Linux 上通常使用 systemd 双服务，而在 Windows 或单机桌面客户端场景下，并没有统一的 systemd：
1. **子进程接管**：Controller 会自动寻找当前目录、`core_bin/` 或系统 PATH 中的 `sing-box.exe`。
2. **Supervisor 生命周期管理**：
   - 自动拉起 `sing-box.exe run -c <config_path>` 作为子进程。
   - 监听子进程健康状态。如果内核意外崩溃，Supervisor 按照指数退避策略自动拉起恢复。
   - 配置修改并保存后，自动触发重新启动或优雅热重载。
   - 进程退出时（例如窗口关闭或接收终止信号），优雅通知并同步杀死内核子进程，防止孤儿进程残留导致 TUN 网卡异常。

---

## 目录结构建议

```plaintext
sing-controller/
├── sing-controller.exe      # 控制器与 WebUI
├── sing-box.exe             # sing-box 核心内核
├── controller.json          # 控制器配置文件 (自动生成)
├── sing-box.json            # 代理主配置文件
└── providers/               # 订阅源与节点配置缓存目录
```

---

## 命令行参数控制

你可以通过 `-managed` 显式开启或关闭托管模式：

```powershell
# 强制以托管模式启动 (Windows 默认)
.\sing-controller.exe -managed=true

# 非托管模式（仅作为 WebUI 和配置管理，不拉起 sing-box 进程）
.\sing-controller.exe -managed=false
```

---

## 权限与 TUN 网卡

如果配置了 TUN 虚拟网卡入站，必须以 **管理员身份 (Administrator)** 运行 `sing-controller.exe`，以便其为内核创建网络适配器及配置系统路由表。
