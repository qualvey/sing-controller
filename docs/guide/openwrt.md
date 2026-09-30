# OpenWrt 部署

在 OpenWrt 路由系统（如搭载 musl libc 的 23.05+ 或支持 APK 软件包的 OpenWrt 快照版）中，`sing-controller` 可以非常轻量地充当全功能旁路由/主路由管控面板。

---

## 编译 OpenWrt 软件包

项目根目录下提供了用于构建 OpenWrt APK 包的脚本 `build-openwrt-apk.ps1`：

```powershell
# 在 Windows 环境下一键交叉编译 OpenWrt APK
.\build-openwrt-apk.ps1 -Arch aarch64_generic
```

构建产物将输出在 `dist/` 目录下，命名规则如 `sing-controller_<version>_<arch>.apk`。

---

## procd 服务托管

在 OpenWrt 系统中，通过 `procd` 进行统一进程管理。`/etc/init.d/sing-controller` 服务脚本示例：

```sh
#!/bin/sh /etc/rc.common

START=99
USE_PROCD=1

PROG=/usr/bin/sing-controller
CONF=/etc/sing-controller/config.json

start_service() {
    procd_open_instance
    procd_set_param command "$PROG" -config "$CONF"
    procd_set_param respawn 3600 5 0
    procd_set_param stdout 1
    procd_set_param stderr 1
    procd_close_instance
}

stop_service() {
    killall sing-controller
}
```

赋予执行权限并启动：
```sh
chmod +x /etc/init.d/sing-controller
/etc/init.d/sing-controller enable
/etc/init.d/sing-controller start
```
