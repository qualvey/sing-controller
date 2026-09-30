# RESTful 核心接口速查

以下为 `sing-controller` 核心 RESTful 端点速查表。完整参数与响应模型请参考交互式文档。

---

## 系统与状态

| 方法 | 路径 | 描述 |
| :--- | :--- | :--- |
| `GET` | `/healthz` | 基础探针健康检查（无需鉴权） |
| `GET` | `/api/status` | 获取系统总览、内存、内核状态与配置路径 |
| `GET` | `/api/node-health` | 获取后台节点可用性探测状态 |
| `GET` | `/api/node-health/config` | 获取探测与 webhook 配置（不回传 webhook 凭据） |
| `PUT` | `/api/node-health/config` | 更新探测与 webhook 配置 |
| `GET` | `/api/ports/available` | 检测指定端口是否可用 |
| `POST` | `/api/system/elevate` | 请求系统提权（Windows UAC 重启） |
| `POST` | `/api/system/disable-tun` | 一键紧急禁用 TUN 虚拟网卡 |

---

## 代理配置管理

| 方法 | 路径 | 描述 |
| :--- | :--- | :--- |
| `GET` | `/api/config` | 获取已解析的主配置对象 |
| `PUT` | `/api/config` | 更新主配置（经校验管线，原子写盘） |
| `GET` | `/api/config/raw` | 读取原始 JSON 文本内容 |
| `PUT` | `/api/config/raw` | 写入原始 JSON（自动执行校验） |
| `GET` | `/api/schema` | 获取 sing-box JSON Schema 定义 |
| `GET` | `/api/types` | 获取系统支持的协议与模块类型枚举 |

---

## 内核生命周期 (Supervisor)

| 方法 | 路径 | 描述 |
| :--- | :--- | :--- |
| `GET` | `/api/core/status` | 获取 sing-box 子进程状态与 PID |
| `POST` | `/api/core/start` | 启动 sing-box 内核 |
| `POST` | `/api/core/stop` | 停止 sing-box 内核 |
| `POST` | `/api/core/restart` | 重启 sing-box 内核 |
| `GET` | `/api/core/logs` | 获取内核近期运行日志（分页） |
| `GET` | `/api/core/logs/stream` | 实时日志流（SSE EventStream） |

---

## 订阅源 (Providers) 与配置组 (Profiles)

| 方法 | 路径 | 描述 |
| :--- | :--- | :--- |
| `GET` | `/api/providers` | 获取所有订阅源列表 |
| `POST` | `/api/providers` | 新增订阅源（支持 HTTP / 本地文件） |
| `POST` | `/api/providers/{id}/refresh` | 触发远程订阅拉取与节点更新 |
| `GET` | `/api/providers/{tag}/nodes` | 查看指定订阅源解析出的节点详情 |
| `GET` | `/api/profiles` | 查看配置模板组 |
| `POST` | `/api/profiles/{id}/apply` | 应用指定配置组并触发热重载 |

---

## 用户池 (Users)

| 方法 | 路径 | 描述 |
| :--- | :--- | :--- |
| `GET` | `/api/users` | 获取所有代理用户列表 |
| `POST` | `/api/users` | 新增用户并分配入站标签 |
| `PUT` | `/api/users/{name}` | 更新用户信息与绑定的入站 |
| `DELETE` | `/api/users/{name}` | 删除用户 |

---

## 工具集 (Tools)

| 方法 | 路径 | 描述 |
| :--- | :--- | :--- |
| `POST` | `/api/tools/uuid` | 生成随机合规 UUID v4 |
| `POST` | `/api/tools/reality-keypair` | 生成 VLESS Reality 公私钥对与 ShortID |
| `POST` | `/api/tools/password` | 生成高强度随机密码/Token |
| `POST` | `/api/tools/parse-json` | 校验并格式化任意 JSON 字符串 |
