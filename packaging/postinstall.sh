#!/bin/sh
# sing-controller deb postinstall
set -e

# 创建系统用户（无特权、无登录 shell）
if ! getent group sing-controller >/dev/null 2>&1; then
    groupadd --system sing-controller
fi
if ! getent passwd sing-controller >/dev/null 2>&1; then
    useradd --system --gid sing-controller --home-dir /etc/sing-controller \
        --shell /usr/sbin/nologin sing-controller
fi

# 配置目录权限（config.json 本体由 dpkg conffiles 管理，这里只管目录）
mkdir -p /etc/sing-controller
chown -R sing-controller:sing-controller /etc/sing-controller
chmod 750 /etc/sing-controller

# 启动服务（无 systemd 的环境跳过，如容器/chroot）
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload
    systemctl enable sing-controller.service
    systemctl restart sing-controller.service || true
fi

exit 0
