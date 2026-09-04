#!/bin/sh
# sing-controller deb postinstall
set -e

# 创建系统用户（若不存在）
if ! getent group sing-box >/dev/null 2>&1; then
    groupadd --system sing-box
fi
if ! getent passwd sing-box >/dev/null 2>&1; then
    useradd --system --gid sing-box --home-dir /var/lib/sing-box \
        --shell /usr/sbin/nologin sing-box
fi

# controller 自身配置目录（config.json 属 dpkg conffiles，升级不覆盖）
mkdir -p /etc/sing-controller
chown -R sing-box:sing-box /etc/sing-controller
chmod 750 /etc/sing-controller

# 主配置目录（原子写盘 tmp+rename 需可写）
mkdir -p /etc/sing-box
chown -R sing-box:sing-box /etc/sing-box
chmod 750 /etc/sing-box

# 状态与持久化缓存目录（存放 cache.db 与 GeoIP/GeoSite 缓存）
mkdir -p /var/lib/sing-box
chown -R sing-box:sing-box /var/lib/sing-box
chmod 750 /var/lib/sing-box

# 确保内核二进制与动态库权限及缓存刷新
if [ -f /usr/bin/sing-box ]; then
    chmod +x /usr/bin/sing-box
fi
if [ -f /usr/lib/libcronet.so ] || [ -f /usr/bin/libcronet.so ]; then
    ldconfig >/dev/null 2>&1 || true
fi

# 重启服务（容器/chroot 内无 systemd 则跳过）
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload
    systemctl enable sing-controller.service
    systemctl restart sing-controller.service || true
fi

exit 0
