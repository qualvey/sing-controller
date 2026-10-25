# build-openwrt-apk.ps1 — 针对 OpenWrt / ImmortalWrt 构建官方规范的轻量 APK 安装包 (.apk)
# 双服务架构：sing-box 作为独立系统服务由官方 apk 安装管理，本包仅提供 WebUI 与配置控制中心
# 用法：pwsh ./build-openwrt-apk.ps1 [-Arch cortex-a53|arm64|amd64] [-Upx] [-SkipFrontend] [-HostAlias 192.168.1.4]
# 产物：dist/sing-controller_<version>_openwrt_<arch>.apk

[CmdletBinding()]
param(
    [ValidateSet('cortex-a53', 'arm64', 'amd64')]
    [string]$Arch = 'cortex-a53',

    [switch]$SkipFrontend,

    [string]$HostAlias = '192.168.1.4',

    [switch]$IncludeCore,

    [switch]$Upx
)

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
Set-Location $root

# 0. 准备工具
if (-not (Get-Command nfpm -ErrorAction SilentlyContinue)) {
    Write-Host '[*] 安装 nfpm 打包工具 ...' -ForegroundColor Yellow
    go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
    $env:PATH = "$env:USERPROFILE\go\bin;$env:PATH"
}

# 1. 构建前端
if (-not $SkipFrontend) {
    Write-Host '[*] 构建前端 (pnpm run build) ...' -ForegroundColor Green
    Push-Location "$root/controller/web"
    try {
        pnpm run build
        if ($LASTEXITCODE -ne 0) { throw '前端构建失败' }
    } finally {
        Pop-Location
    }
} else {
    Write-Host '[*] 跳过前端构建 (-SkipFrontend)' -ForegroundColor DarkGray
}

# 2. 获取版本信息
$gitVer = (git describe --tags --always 2>$null) -replace '^v', ''
if (-not $gitVer) { $gitVer = '0.11.0' }
$apkVer = ($gitVer -replace '-.*$', '')
if ($gitVer -match 'SNAPSHOT') {
    $apkVer = "$apkVer-r1"
}

# 3. 架构映射
$isArm64 = $Arch -in @('cortex-a53', 'arm64')
$goarch = if ($isArm64) { 'arm64' } else { 'amd64' }
$openwrtArch = if ($isArm64) { 'aarch64_cortex-a53' } else { 'x86_64' }
$suffix = if ($IncludeCore) { "_aio" } else { "" }
$fileName = "sing-controller_${gitVer}${suffix}_openwrt_${openwrtArch}.apk"

# 4. 编译 Go 二进制（纯配置与面板模式，去除不必要的重型底层代理驱动 tags）
$outDir = Join-Path $root 'dist'
if (-not (Test-Path $outDir)) { New-Item -ItemType Directory -Path $outDir | Out-Null }
$binPath = Join-Path $outDir "sing-controller-openwrt-$goarch"

Write-Host "[*] 编译 sing-controller (linux/$goarch, CGO_ENABLED=0, 精简模式) ..." -ForegroundColor Green
$env:CGO_ENABLED = "0"
$env:GOOS = "linux"
$env:GOARCH = $goarch

$ldflags = "-s -w -X main.version=$gitVer"

Push-Location "$root/controller"
try {
    go build -ldflags $ldflags -o $binPath .
    if ($LASTEXITCODE -ne 0) { throw 'Go 二进制编译失败' }
} finally {
    Pop-Location
}

# 4.1 可选 UPX 极致压缩
if ($Upx) {
    if (Get-Command upx -ErrorAction SilentlyContinue) {
        Write-Host '[*] 执行 UPX 极限二进制压缩 (--best --lzma) ...' -ForegroundColor Cyan
        upx --best --lzma -f $binPath
    } else {
        Write-Host '[!] 未检测到 upx 命令，跳过 UPX 压缩' -ForegroundColor Yellow
    }
}

# 5. 内核文件处理（仅在显式传入 -IncludeCore 时打包内核）
$coreEntry = ""
$pkgReplaces = ""
$pkgDepends = ""

if ($IncludeCore) {
    Write-Host "[*] 全合一模式 (-IncludeCore): 检索嵌入 sing-box 内核 ..." -ForegroundColor Yellow
    $coreSrc = $null
    if ($isArm64) {
        if (Test-Path "$root/core_bin/sing-box-openwrt-aarch64") {
            $coreSrc = "./core_bin/sing-box-openwrt-aarch64"
        } elseif (Test-Path "$root/core_bin/sing-box-openwrt-cortex-a53") {
            $coreSrc = "./core_bin/sing-box-openwrt-cortex-a53"
        } elseif (Test-Path "$root/core_bin/sing-box") {
            $coreSrc = "./core_bin/sing-box"
        }
    } else {
        if (Test-Path "$root/core_bin/sing-box-openwrt-x86_64") {
            $coreSrc = "./core_bin/sing-box-openwrt-x86_64"
        } elseif (Test-Path "$root/core_bin/sing-box") {
            $coreSrc = "./core_bin/sing-box"
        }
    }

    if ($coreSrc) {
        Write-Host "  -> 绑定 OpenWrt 专属内核: $coreSrc" -ForegroundColor Cyan
        $coreEntry = @"
  - src: $coreSrc
    dst: /usr/bin/sing-box
    file_info:
      mode: 0755
"@
        $pkgReplaces = @"
replaces:
  - sing-box
"@
    }
} else {
    Write-Host "[*] 双服务解耦模式 (推荐): 不包含内核，声明依赖 sing-box" -ForegroundColor Green
    $pkgDepends = @"
depends:
  - sing-box
"@
}

$cleanBinPath = $binPath -replace '\\', '/'

# 6. 生成 nfpm.yaml
$nfpmConfig = Join-Path $env:TEMP 'nfpm-openwrt-apk.yaml'
@"
name: "sing-controller"
arch: "$openwrtArch"
platform: "linux"
version: "$apkVer"
section: "net"
priority: "optional"
$pkgDepends
$pkgReplaces
maintainer: "qualvey <wel@ryugo.org>"
description: |-
  sing-box controller - RESTful configuration management service and WebUI for sing-box.
  Manages sing-box configuration, profiles, and service reload.
vendor: "qualvey"
homepage: "https://github.com/qualvey/sing-controller"
license: "GPL-3.0"
contents:
  - src: "$cleanBinPath"
    dst: /usr/bin/sing-controller
    file_info:
      mode: 0755
  - src: ./packaging/openwrt/sing-controller.init
    dst: /etc/init.d/sing-controller
    file_info:
      mode: 0755
  - src: ./packaging/openwrt/sing-controller.config
    dst: /etc/config/sing-controller
    type: config|noreplace
  - src: ./packaging/config.json
    dst: /etc/sing-controller/config.json
    type: config|noreplace
$coreEntry
scripts:
  postinstall: ./packaging/openwrt/postinstall.sh
  postremove: ./packaging/openwrt/postremove.sh
"@ | Set-Content -Path $nfpmConfig -Encoding utf8NoBOM

# 7. 调用 nfpm 构建原生多流 APK
$targetApk = Join-Path $outDir $fileName
Write-Host "[*] 调用 nfpm 打包原生 OpenWrt APK ($openwrtArch) ..." -ForegroundColor Green
nfpm pkg --packager apk --config $nfpmConfig --target $targetApk
if ($LASTEXITCODE -ne 0) { throw 'nfpm 打包失败' }

$apkItem = Get-Item $targetApk
Write-Host "`n[✓] OpenWrt APK 原生打包完成！" -ForegroundColor Cyan
Write-Host "  -> 产物: $($apkItem.FullName) ($([math]::Round($apkItem.Length/1MB, 2)) MB)" -ForegroundColor Green
Write-Host "     安装命令: scp $($apkItem.FullName) root@${HostAlias}:/tmp/ && ssh root@${HostAlias} 'apk add --allow-untrusted /tmp/$($apkItem.Name)'" -ForegroundColor Gray
