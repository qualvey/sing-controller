# packaging

跨平台部署资源。controller 的 reload.mode=auto 会自动探测各平台的重载机制
（systemd → openrc/rc-service → OpenWrt/procd → SysV service），无需手动改配置。

| 目录/文件 | 平台 | 说明 |
|---|---|---|
| `sing-controller.service` | systemd（deb 安装） | sing-controller 服务单元，`postinstall.sh` 负责安装 |
| `openrc/sing-box` | Alpine Linux（openrc） | sing-box init 脚本（参考）：`rc-service sing-box reload` 发 SIGHUP |
| `openwrt/sing-box.init` | OpenWrt（procd） | sing-box init 脚本（参考）：`service sing-box reload` 发 SIGHUP |

## Alpine（openrc）

```sh
cp packaging/openrc/sing-box /etc/init.d/sing-box && chmod +x /etc/init.d/sing-box
rc-update add sing-box default
rc-service sing-box start
# controller 装好后 reload.mode 保持 auto 即可，保存配置自动 rc-service sing-box reload
```

## OpenWrt（procd）

```sh
cp packaging/openwrt/sing-box.init /etc/init.d/sing-box && chmod +x /etc/init.d/sing-box
/etc/init.d/sing-box enable
/etc/init.d/sing-box start
```

> 脚本为参考实现：默认二进制路径 `/usr/bin/sing-box`、配置 `/etc/sing-box/config.json`，按实际布局调整。
> sing-box 自身不写 pidfile（openrc 脚本由 start-stop-daemon 托管；procd 由实例 pid 管理）。
