#!/bin/sh
set -e

# 确保 init 脚本可执行
chmod +x /etc/init.d/sing-controller 2>/dev/null || true

# BusyBox / OpenWrt 兼容创建 sing-box 用户与组
if ! getent group sing-box >/dev/null 2>&1; then
    addgroup -S sing-box 2>/dev/null || groupadd --system sing-box 2>/dev/null || true
fi
if ! getent passwd sing-box >/dev/null 2>&1; then
    adduser -S -D -H -G sing-box -s /bin/false sing-box 2>/dev/null || \
    useradd --system --gid sing-box --home-dir /var/lib/sing-box --shell /bin/false sing-box 2>/dev/null || true
fi

# 确保状态与配置目录存在并赋权
mkdir -p /var/lib/sing-box /etc/sing-controller /etc/sing-box
chown -R root:sing-box /var/lib/sing-box /etc/sing-controller /etc/sing-box 2>/dev/null || true
chmod 775 /var/lib/sing-box /etc/sing-controller /etc/sing-box 2>/dev/null || true

# 确保内核二进制权限
if [ -f /usr/bin/sing-box ]; then
    chmod +x /usr/bin/sing-box 2>/dev/null || true
fi

# 自动开机自启
/etc/init.d/sing-controller enable 2>/dev/null || true

exit 0
