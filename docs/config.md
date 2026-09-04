# sing-controller 配置说明

`sing-controller` 的配置采用声明式模块化设计，语法和规范与 `sing-box` 保持高度同构。

默认配置文件路径：
- Linux / OpenWrt：`/etc/sing-controller/config.json`
- Windows：`controller.json`

---

## 完整配置规范示例

```json
{
  "$schema": "https://github.com/qualvey/sing-controller/schema.json",

  "server": {
    "listen": "::",
    "listen_port": 8080,
    "secret": ""
  },

  "log": {
    "level": "info",
    "output": "stderr",
    "outfile": ""
  },

  "core": {
    "config_path": "/etc/sing-box/config.json",
    "managed": false,
    "binary_path": "",
    "reload": {
      "mode": "auto",
      "service": "sing-box",
      "pid_file": "/var/run/sing-box.pid",
      "hook": "/usr/local/bin/reload-sing-box.sh",
      "after_save": false
    }
  },

  "api": {
    "clash": {
      "address": "http://127.0.0.1:9090",
      "secret": ""
    },
    "service": {
      "address": "http://127.0.0.1:9900",
      "secret": ""
    },
    "v2ray": {
      "address": "127.0.0.1:10085"
    }
  },

  "stats": {
    "poll_interval": "10s",
    "pattern": "",
    "reset_on_query": false,
    "real_time": {
      "enabled": false,
      "interval_ms": 1000,
      "online_threshold_sec": 60
    }
  }
}
```

---

## 模块字段详解

### 1. `server`（服务自身网络与安全）
Controller 自身 WebUI 与 REST API 监听端点。

| 字段 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `listen` | string | `"::"` | 监听 IP 地址。`"::"` 监听双栈所有网卡，`"127.0.0.1"` 仅本机访问 |
| `listen_port` | uint16 | `8080` | HTTP 监听端口 |
| `secret` | string | `""` | 可选 API 鉴权 Token。若设置，客户端需在请求头携带 `X-Secret` |

---

### 2. `log`（Controller 自身日志）
配置风格与 `sing-box` 的 `log` 段保持一致。

| 字段 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `level` | string | `"info"` | 日志级别：`trace` \| `debug` \| `info` \| `warn` \| `error` |
| `output` | string | `"stderr"` | 日志输出目标：`stderr` \| `stdout` \| `file` |
| `outfile` | string | `""` | 日志文件路径。当 `output` 为 `file` 或指定了文件路径时落盘写入 |

---

### 3. `core`（sing-box 内核与生命周期管理）
负责管理受控的 sing-box 配置文件及服务启停/热重载。

| 字段 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `config_path` | string | `/etc/sing-box/config.json`（Windows 为 `sing-box.json`） | sing-box 主配置文件路径 |
| `managed` | bool | Windows 为 `true`，Linux 为 `false` | 是否由 Controller Supervisor 托管启动 sing-box 子进程 |
| `binary_path` | string | `""` | `managed` 为 true 时指定 sing-box 二进制路径（留空则自动查找） |
| `reload.mode` | string | `"auto"` | 重载模式：`auto` \| `systemd` \| `procd` \| `rc-service` \| `pidfile` \| `hook` \| `none` |
| `reload.service` | string | `"sing-box"` | 系统服务名称（用于 systemd / procd / openrc） |
| `reload.pid_file` | string | `""` | PID 文件路径（用于 pidfile 模式） |
| `reload.hook` | string | `""` | 自定义重载命令（hook 模式） |
| `reload.after_save` | bool | `false` | WebUI 保存配置后是否自动触发重载 |

---

### 4. `api`（内核上游 API 通信与反向代理）
WebUI 页面及 Controller 内部反向代理到 sing-box 的通信端点。

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `clash.address` | string | sing-box 的 `experimental.clash_api.external_controller` 地址 |
| `clash.secret` | string | clash API 访问密钥（若配置，代理时自动注入 Bearer Token） |
| `service.address` | string | sing-box 的 `services[type=api]` gRPC-Web 端点 |
| `service.secret` | string | service API 访问密钥 |
| `v2ray.address` | string | sing-box 的 v2ray stats gRPC 统计端口（通常为 `127.0.0.1:10085`） |

> **自动推断机制**：若 `api` 段对应端点留空，Controller 启动时会自动尝试从 `core.config_path` 的 sing-box 配置中推断并自适应连接，无需手动重复配置。

---

### 5. `stats`（流量统计与实时监控）
控制后台 Collector 对流量和用户连接的数据统计行为。

| 字段 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `poll_interval` | string | `"10s"` | 从 sing-box v2ray stats 采集数据的轮询周期（Go duration 格式） |
| `pattern` | string | `""` | 流量统计规则匹配过滤正则 |
| `reset_on_query` | bool | `false` | 查询时是否通知内核重置计数器 |
| `real_time.enabled` | bool | `false` | 是否开启实时 WebSocket 流量推送广播 |
| `real_time.interval_ms` | int | `1000` | 实时广播频率（毫秒） |
| `real_time.online_threshold_sec` | int | `60` | 判定用户/节点在线的活跃超时时间（秒） |