#!/bin/sh
# 卸载时停止并禁用服务
/etc/init.d/sing-controller stop 2>/dev/null || true
/etc/init.d/sing-controller disable 2>/dev/null || true
exit 0
