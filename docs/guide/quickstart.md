# 快速启动

本章节介绍如何在本地环境快速构建并运行 `sing-controller` 及 WebUI 面板。

---

## 环境准备

- **Go**: 1.22 或更高版本（推荐 1.24+）
- **Node.js**: 20+ / 22+
- **pnpm**: 9+ / 11+
- **sing-box**: 内核二进制文件（建议 1.10+）

---

## 1. 本地快速运行 (开发模式)

### 1.1 启动后端 Controller
进入 `controller` 目录，携带全部推荐 tag 启动：

::: code-group
```bash [Linux / macOS]
cd controller
go run -tags "with_quic,with_tailscale,with_gvisor,with_utls,with_clash_api,with_acme,with_tun,with_dhcp,with_wireguard,with_ccm,with_ocm,with_cloudflared,with_usbip" .
```

```powershell [Windows PowerShell]
cd controller
go run -tags "with_quic,with_tailscale,with_gvisor,with_utls,with_clash_api,with_acme,with_tun,with_dhcp,with_wireguard,with_ccm,with_ocm,with_cloudflared,with_usbip" .
```
:::

服务默认将在 `http://0.0.0.0:8080` 启动，并会在当前目录生成默认配置文件 `controller.json`。

### 1.2 启动 WebUI 开发服务器
在另一终端窗口启动前端：

```bash
cd controller/web
pnpm install
pnpm dev
```
WebUI 开发服务器默认运行在 `http://localhost:5173`，其自带 Vite 代理将自动把 `/api/*` 请求转发至后端的 `:8080`。

---

## 2. 生产打包构建

若要发布单可执行文件（静态资源嵌入二进制）：

### 2.1 编译 WebUI 静态资源
```bash
cd controller/web
pnpm install
pnpm build
```
前端产物将输出至 `controller/web/dist`。

### 2.2 编译后端二进制

::: code-group
```powershell [Windows (exe)]
cd controller
go build -ldflags="-s -w" -o sing-controller.exe .
```

```powershell [跨平台 Linux arm64]
cd controller
$env:CGO_ENABLED="0"
$env:GOOS="linux"
$env:GOARCH="arm64"
go build -ldflags="-s -w" -o sing-controller-linux-arm64 .
```

```bash [Linux amd64]
cd controller
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o sing-controller-linux-amd64 .
```
:::

编译生成的单文件内置了 WebUI，双击或以守护进程启动后直接访问 `http://<IP>:8080` 即可使用。
