<#
.SYNOPSIS
    sing-controller 统一构建脚本

.DESCRIPTION
    支持构建前端静态资源 (WebUI) 及各平台的 Go 二进制产物或 Debian 安装包 (.deb)。

.PARAMETER Target
    构建目标平台：
    - local (默认): 当前系统原生二进制 (Windows 下生成 sing-controller.exe)
    - windows / windows-amd64: Windows 64位二进制
    - windows-arm64: Windows ARM64 二进制
    - linux-amd64: Linux 64位二进制
    - linux-arm64: Linux ARM64 二进制 (NanoPi / 树莓派等)
    - linux-armv7: Linux ARMv7 32位二进制
    - deb-arm64: Linux ARM64 Debian 安装包 (.deb)
    - deb-all: 全平台 Debian 安装包及压缩包 (.deb / .tar.gz)
    - all: 常用二进制全量构建 (Windows x64, Linux x64, Linux ARM64, Linux ARMv7)

.PARAMETER SkipFrontend
    跳过 WebUI 前端构建 (pnpm run build)

.PARAMETER Version
    指定构建版本号 (默认自动从 Git tag 读取)

.PARAMETER Tags
    指定 Go build tags

.EXAMPLE
    .\build.ps1                         # 编译前端并构建本地二进制
    .\build.ps1 -SkipFrontend           # 仅重新编译本地 Go 二进制
    .\build.ps1 -Target linux-arm64     # 交叉编译 Linux ARM64 二进制
    .\build.ps1 -Target deb-arm64       # 打包 Linux ARM64 deb 安装包
#>

[CmdletBinding()]
param(
    [ValidateSet('local', 'windows', 'windows-amd64', 'windows-arm64', 'linux-amd64', 'linux-arm64', 'linux-armv7', 'deb-arm64', 'deb-all', 'all')]
    [string]$Target = 'local',

    [switch]$SkipFrontend,

    [string]$Version,

    [string]$Tags = "with_quic,with_dhcp,with_wireguard,with_utls,with_clash_api,with_gvisor,with_tailscale,with_v2ray_api",

    [string]$OutDir = ""
)

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
Set-Location $root

if ([string]::IsNullOrWhiteSpace($OutDir)) {
    $OutDir = Join-Path $root "dist"
}

# 确保输出目录存在
if (-not (Test-Path $OutDir)) {
    New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
}

# 获取版本号
if (-not $Version) {
    $gitVer = git describe --tags --always --dirty 2>$null
    if ($gitVer) {
        $Version = $gitVer.Trim()
    }
    else {
        $Version = "dev"
    }
}
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " sing-controller 构建工具" -ForegroundColor Cyan
Write-Host " 目标: $Target" -ForegroundColor Yellow
Write-Host " 版本: $Version" -ForegroundColor Yellow
Write-Host " Tags: $Tags" -ForegroundColor DarkGray
Write-Host "==========================================" -ForegroundColor Cyan

# 1. 构建前端
if (-not $SkipFrontend) {
    Write-Host "`n[*] [1/2] 构建 WebUI 前端 (pnpm run build) ..." -ForegroundColor Green
    Push-Location (Join-Path $root "controller/web")
    try {
        if (-not (Get-Command pnpm -ErrorAction SilentlyContinue)) {
            throw "pnpm 未找到，请先安装 pnpm (npm i -g pnpm)"
        }
        pnpm run build
        if ($LASTEXITCODE -ne 0) { throw "前端构建失败" }
    }
    finally {
        Pop-Location
    }
}
else {
    Write-Host "`n[*] [1/2] 跳过前端构建 (-SkipFrontend)" -ForegroundColor DarkGray
}

# 2. 编译函数
function Build-Binary {
    param(
        [string]$Goos,
        [string]$Goarch,
        [string]$Goarm = "",
        [string]$OutputName
    )

    $outputPath = Join-Path $OutDir $OutputName
    Write-Host "  -> 编译 $Goos/$Goarch$(if($Goarm){"v$Goarm"}) => $OutputName ..." -ForegroundColor White

    $env:CGO_ENABLED = "0"
    $env:GOOS = $Goos
    $env:GOARCH = $Goarch
    if ($Goarm) {
        $env:GOARM = $Goarm
    }
    else {
        Remove-Item env:GOARM -ErrorAction SilentlyContinue
    }

    $ldflags = "-s -w -X main.version=$Version"

    Push-Location (Join-Path $root "controller")
    try {
        go build -tags $Tags -ldflags $ldflags -o $outputPath .
        if ($LASTEXITCODE -ne 0) { throw "Go 编译失败: $Goos/$Goarch" }
    }
    finally {
        Pop-Location
    }

    $file = Get-Item $outputPath
    $sizeMB = [math]::Round($file.Length / 1MB, 2)
    Write-Host "     [OK] $OutputName ($sizeMB MB)" -ForegroundColor Green
}

# 3. 检查 Goreleaser
function Ensure-Goreleaser {
    if (-not (Get-Command goreleaser -ErrorAction SilentlyContinue)) {
        Write-Host "[!] goreleaser 未安装，尝试 go install ..." -ForegroundColor Yellow
        go install github.com/goreleaser/goreleaser/v2@latest
        $env:PATH = "$env:USERPROFILE\go\bin;$env:PATH"
        if (-not (Get-Command goreleaser -ErrorAction SilentlyContinue)) {
            throw "goreleaser 安装失败或未在 PATH 中找到"
        }
    }
}

# 4. 执行构建
Write-Host "`n[*] [2/2] 正在构建 Go 产物 ..." -ForegroundColor Green

switch ($Target) {
    'local' {
        if ($IsWindows -or $env:OS -eq 'Windows_NT') {
            Build-Binary -Goos 'windows' -Goarch 'amd64' -OutputName 'sing-controller.exe'
            # 同时拷贝到 controller/sing-controller.exe 便于根目录直接调试
            Copy-Item (Join-Path $OutDir 'sing-controller.exe') (Join-Path $root 'pkg_test/sing-controller.exe') -Force
        }
        elseif ($IsMacOS) {
            Build-Binary -Goos 'darwin' -Goarch 'arm64' -OutputName 'sing-controller-darwin-arm64'
        }
        else {
            Build-Binary -Goos 'linux' -Goarch 'amd64' -OutputName 'sing-controller-linux-amd64'
        }
    }
    { $_ -in @('windows', 'windows-amd64') } {
        Build-Binary -Goos 'windows' -Goarch 'amd64' -OutputName 'sing-controller-windows-amd64.exe'
    }
    'windows-arm64' {
        Build-Binary -Goos 'windows' -Goarch 'arm64' -OutputName 'sing-controller-windows-arm64.exe'
    }
    'linux-amd64' {
        Build-Binary -Goos 'linux' -Goarch 'amd64' -OutputName 'sing-controller-linux-amd64'
    }
    'linux-arm64' {
        Build-Binary -Goos 'linux' -Goarch 'arm64' -OutputName 'sing-controller-linux-arm64'
    }
    'linux-armv7' {
        Build-Binary -Goos 'linux' -Goarch 'arm' -Goarm '7' -OutputName 'sing-controller-linux-armv7'
    }
    'deb-arm64' {
        Ensure-Goreleaser
        $cfgFile = Join-Path $env:TEMP 'goreleaser-arm64.yaml'
        @'
version: 2
project_name: sing-controller
builds:
  - id: sing-controller
    dir: controller
    binary: sing-controller
    env:
      - CGO_ENABLED=0
    goos:
      - linux
    goarch:
      - arm64
    tags:
      - with_quic
      - with_utls
      - with_gvisor
      - with_dhcp
      - with_wireguard
      - with_acme
      - with_clash_api
      - with_tailscale
      - with_ccm
      - with_ocm
      - with_cloudflared
      - with_usbip
    ldflags:
      - -s -w -X main.version={{ .Version }}
archives:
  - formats: [tar.gz]
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    files:
      - none*
nfpms:
  - id: deb
    package_name: sing-controller
    vendor: qualvey
    homepage: https://github.com/qualvey/sing-controller
    maintainer: qualvey <wel@ryugo.org>
    description: |-
      sing-box controller - RESTful configuration management service for sing-box.
      Generates and validates sing-box config.json; does not run the proxy itself.
    license: GPL-3.0
    formats:
      - deb
    bindir: /usr/bin
    contents:
      - src: ./packaging/sing-controller.service
        dst: /etc/systemd/system/sing-controller.service
      - src: ./packaging/config.json
        dst: /etc/sing-controller/config.json
        type: config|noreplace
    scripts:
      postinstall: ./packaging/postinstall.sh
      postremove: ./packaging/postremove.sh
    deb:
      breaks:
        - sing-box-webui
'@ | Set-Content -Path $cfgFile -Encoding utf8NoBOM

        Write-Host "  -> 调用 goreleaser 构建 linux/arm64 deb ..." -ForegroundColor White
        goreleaser release --snapshot --skip=publish --clean -f $cfgFile
        if ($LASTEXITCODE -ne 0) { throw "goreleaser failed" }

        $deb = Get-ChildItem $OutDir -Filter '*linux_arm64.deb' | Select-Object -First 1
        if ($deb) {
            Write-Host "     [OK] deb 产物: $($deb.FullName) ($([math]::Round($deb.Length/1MB,2)) MB)" -ForegroundColor Green
        }
    }
    'deb-all' {
        Ensure-Goreleaser
        Write-Host "  -> 调用 goreleaser 全量构建 deb/tar.gz ..." -ForegroundColor White
        goreleaser release --snapshot --skip=publish --clean -f (Join-Path $root ".goreleaser.yaml")
        if ($LASTEXITCODE -ne 0) { throw "goreleaser failed" }
    }
    'all' {
        Build-Binary -Goos 'windows' -Goarch 'amd64' -OutputName 'sing-controller-windows-amd64.exe'
        Build-Binary -Goos 'linux' -Goarch 'amd64' -OutputName 'sing-controller-linux-amd64'
        Build-Binary -Goos 'linux' -Goarch 'arm64' -OutputName 'sing-controller-linux-arm64'
        Build-Binary -Goos 'linux' -Goarch 'arm' -Goarm '7' -OutputName 'sing-controller-linux-armv7'
    }
}

Write-Host "`n[✓] 构建完成！产物位于: $OutDir" -ForegroundColor Cyan
