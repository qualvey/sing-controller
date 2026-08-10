#!/bin/sh
# sing-controller deb postinstall
set -e

# 创建系统用户（若不存在）
if ! getent group sing-controller >/dev/null 2>&1; then
    groupadd --system sing-controller
fi
if ! getent passwd sing-controller >/dev/null 2>&1; then
    useradd --system --gid sing-controller --home-dir /etc/sing-controller \
        --shell /usr/sbin/nologin sing-controller
fi

# controller 自身配置目录（config.json 属 dpkg conffiles，升级不覆盖）
mkdir -p /etc/sing-controller
chown -R sing-controller:sing-controller /etc/sing-controller
chmod 750 /etc/sing-controller

# 主配置目录：服务要原子写盘（tmp+rename），目录必须归服务所有，
# 否则 systemd ProtectSystem=full 放行后仍会因目录 root:root 755 而 permission denied
mkdir -p /etc/sing-box
chown -R sing-controller:sing-controller /etc/sing-box

# 重启服务（容器/chroot 内无 systemd 则跳过）
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload
    systemctl enable sing-controller.service
    systemctl restart sing-controller.service || true
fi

exit 0
