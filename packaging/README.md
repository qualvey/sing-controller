# 跨平台部署与服务编排指南 (Packaging & Deployment)

本项目根据不同操作系统采用最优的运行架构：

1. **Linux / OpenWrt（推荐双服务解耦架构）**：
   - **`sing-box`** 作为独立系统服务运行（由官方 `apt` / `apk` / `opkg` 安装和守护，享有独立的 TUN 网卡与底层网络能力）。
   - **`sing-controller`** 作为独立的 WebUI 与 RESTful 配置中心运行。配置变更后通过 `systemctl reload sing-box` 或 `/etc/init.d/sing-box reload` 触发内核原地 SIGHUP 热重载。
   - **优点**：控制器崩溃或重启**绝不会导致局域网断网**；内核可自由通过系统包管理器单独无感升级；包体积极小（~18MB）。

2. **Windows 桌面（单服务/绿色托管架构）**：
   - Windows 缺少 systemd/procd 守护机制，Controller 自动启用内置 `Supervisor` 托管子进程 `sing-box.exe`（通过 Windows Job Object 作业对象确保父退子亡，安全退出）。

---

## 各系统服务部署说明

### 1. OpenWrt / ImmortalWrt (procd + UCI 双服务)
```sh
# 1. 安装官方内核（如果未安装）
apk add sing-box

# 2. 安装 sing-controller 面板（体积仅 18~24MB）
apk add --allow-untrusted sing-controller_*_openwrt_aarch64_cortex-a53.apk

# 3. 启动并启用两个服务
/etc/init.d/sing-box enable && /etc/init.d/sing-box start
/etc/init.d/sing-controller enable && /etc/init.d/sing-controller start

# 4. 常用 UCI 管理命令（兼容标准 OpenWrt 运维体系）
# uci set sing-controller.main.enabled='1'
# uci commit sing-controller
# /etc/init.d/sing-controller reload   # 通过 SIGHUP 触发无断流原地配置重载
```

### 4. OpenWrt 官方 APK 包一键安装（适用于 OpenWrt 24+ / ImmortalWrt）
现代 OpenWrt 已经全面转向 Alpine 的 APK 包管理器：
```sh
# 本地构建 OpenWrt APK 包（默认 arm64，自动自包含内核）
pwsh ./build-openwrt-apk.ps1 -Arch arm64

# 推送并一键免信任证书安装
scp dist/sing-controller_*_aarch64.apk root@openwrt:/tmp/
ssh root@openwrt 'apk add --allow-untrusted /tmp/sing-controller_*_aarch64.apk'
```

---

## 资产清单

| 目录/文件 | 平台体系 | 说明 |
|---|---|---|
| `sing-controller.service` | systemd（Debian/Ubuntu 等） | sing-controller 服务单元（含 StateDirectory 与全部网络/进程能力） |
| `openrc/sing-controller` | OpenRC（Alpine Linux 等） | sing-controller 专有 OpenRC 服务脚本（推荐模式 B） |
| `openwrt/sing-controller.init` | procd（OpenWrt 软路由） | sing-controller 专有 procd 服务脚本（支持 UCI、权限自愈与热重载） |
| `openwrt/sing-controller.config` | UCI（OpenWrt 软路由） | sing-controller 标准 OpenWrt UCI 配置文件 |
| `openwrt/postinstall.sh` | APK（OpenWrt 软路由） | APK 安装包触发的自动自愈赋权与开机自启脚本 |
| `build-openwrt-apk.ps1` | 构建工具 | 一键构建 OpenWrt aarch64 / x86_64 APK 安装包脚本 |
| `openrc/sing-box` | OpenRC（Alpine 传统参考） | 传统双服务模式参考脚本 |
| `openwrt/sing-box.init` | procd（OpenWrt 传统参考） | 传统双服务模式参考脚本 |
