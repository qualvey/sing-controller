#!/bin/sh
# sing-controller deb postremove
set -e

if command -v systemctl >/dev/null 2>&1; then
    systemctl stop sing-controller.service 2>/dev/null || true
    systemctl disable sing-controller.service 2>/dev/null || true
    systemctl daemon-reload 2>/dev/null || true
fi

# 保留 /etc/sing-controller 下的用户配置与数据（purge 时仅删 conffile）
exit 0
