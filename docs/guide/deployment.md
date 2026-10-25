# Linux systemd 双服务部署

在生产 Linux 服务器（如 Debian / Ubuntu / Armbian / CentOS）上，强烈推荐采用 **双服务架构**：
- `sing-box.service`：负责核心网络代理转发（TUN 网卡、透明代理等）。
- `sing-controller.service`：负责 WebUI 提供、配置热加载、RESTful API、订阅与集群管理。

---

## 架构优势

1. **故障隔离**：WebUI 或 Controller 重启/更新时，sing-box 数据转发不受任何影响，网络连接不中断。
2. **权限最小化与沙箱保护**：Controller 通过调用系统服务机制（`systemctl reload/restart sing-box`）实现无感配置重载。
3. **日志分离**：通过 `journalctl -u sing-box` 与 `journalctl -u sing-controller` 清晰独立排查。

---

## 1. 安装与目录准备

推荐文件系统结构：
- **可执行文件**：`/usr/bin/sing-controller` 与 `/usr/bin/sing-box`
- **Controller 配置**：`/etc/sing-controller/config.json`
- **sing-box 主配置**：`/etc/sing-box/config.json`
- **状态与缓存目录**：`/var/lib/sing-box/`

```bash
sudo mkdir -p /etc/sing-controller /etc/sing-box /var/lib/sing-box
sudo chown -R root:root /etc/sing-controller /etc/sing-box /var/lib/sing-box
```

---

## 2. systemd 服务配置

在 `/etc/systemd/system/sing-controller.service` 写入以下内容：

```ini
[Unit]
Description=sing-box controller (WebUI & configuration management service)
Documentation=https://github.com/qualvey/sing-controller
After=network.target nss-lookup.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
SupplementaryGroups=sing-box

# 状态目录：存放 cache.db 与运行时状态
StateDirectory=sing-box
WorkingDirectory=/var/lib/sing-box

# 能力集：底层网络控制 + TUN 网卡嗅探 + 写盘覆盖
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_RAW CAP_NET_BIND_SERVICE CAP_SYS_PTRACE CAP_DAC_READ_SEARCH CAP_DAC_OVERRIDE CAP_FOWNER
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_RAW CAP_NET_BIND_SERVICE CAP_SYS_PTRACE CAP_DAC_READ_SEARCH CAP_DAC_OVERRIDE CAP_FOWNER

LimitNOFILE=1048576
ExecStart=/usr/bin/sing-controller -config /etc/sing-controller/config.json
Restart=on-failure
RestartSec=5s

# 允许读写路径
ReadWritePaths=/etc/sing-controller /etc/sing-box

[Install]
WantedBy=multi-user.target
```

---

## 3. 启用与启动服务

```bash
# 重载 systemd 守护进程
sudo systemctl daemon-reload

# 开机自启并启动 Controller
sudo systemctl enable --now sing-controller

# 查看运行状态
sudo systemctl status sing-controller

# 实时查看日志
sudo journalctl -u sing-controller -f
```

访问 `http://<服务器IP>:8080` 即可直接进入 Web 控制面板。
