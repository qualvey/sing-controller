# build-arm64.ps1 — 本地快速构建 linux/arm64 的 deb（部署到 nanopi 验证用）
# 用法：pwsh ./build-arm64.ps1 [-SkipFrontend] [-HostAlias armbian] [-IncludeCore]
# 产物：dist/deb/sing-controller_<version>_linux_arm64.deb（snapshot 版本，不发布）
[CmdletBinding()]
param(
    [switch]$SkipFrontend,
    [string]$HostAlias = 'armbian',
    [switch]$IncludeCore
)

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot  # 脚本位于项目根（build-arm64.ps1）
Set-Location $root

# 0. 重新打包前端（go:embed 嵌入的是 web/dist，不重建会打包旧前端）
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

if (-not (Get-Command goreleaser -ErrorAction SilentlyContinue)) {
    Write-Host '[!] goreleaser 未安装，尝试 go install ...' -ForegroundColor Yellow
    go install github.com/goreleaser/goreleaser/v2@latest
    $env:PATH = "$env:USERPROFILE\go\bin;$env:PATH"
}

# 临时配置：仅 arm64（双服务模式：不包含内核，声明 recommends sing-box）
$distDir = "dist/deb"
$extraContents = ""
$debDepends = ""
$debReplaces = ""
if ($IncludeCore) {
    Write-Host "[*] 包含内核 (-IncludeCore) ..." -ForegroundColor Cyan
    $debReplaces = @"
    replaces:
      - sing-box
"@
    $extraContents = @"
      - src: ./core_bin/sing-box-arm64
        dst: /usr/bin/sing-box
        file_info:
          mode: 0755
      - src: ./core_bin/libcronet.so
        dst: /usr/lib/libcronet.so
        file_info:
          mode: 0755
      - src: /usr/lib/libcronet.so
        dst: /usr/bin/libcronet.so
        type: symlink
"@
} else {
    Write-Host "[*] 双服务解耦模式 (推荐): 不包含内核，声明 recommends sing-box" -ForegroundColor Green
    $debDepends = @"
    dependencies:
      - sing-box
"@
}

$cfgFile = Join-Path $env:TEMP 'goreleaser-arm64.yaml'
@"
version: 2

project_name: sing-controller

dist: dist/deb

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
      sing-box controller - RESTful configuration management service and WebUI for sing-box.
      Manages sing-box configuration, profiles, and service reload.
    license: GPL-3.0
    formats:
      - deb
    bindir: /usr/bin
$debDepends
$debReplaces
    contents:
      - src: ./packaging/sing-controller.service
        dst: /etc/systemd/system/sing-controller.service
      - src: ./packaging/config.json
        dst: /etc/sing-controller/config.json
        type: config|noreplace$extraContents
    scripts:
      postinstall: ./packaging/postinstall.sh
      postremove: ./packaging/postremove.sh
    deb:
      breaks:
        - sing-box-webui
"@ | Set-Content -Path $cfgFile -Encoding utf8NoBOM

Write-Host "[*] goreleaser snapshot build (linux/arm64 deb only)..." -ForegroundColor Green
goreleaser release --snapshot --skip=publish --clean -f $cfgFile
if ($LASTEXITCODE -ne 0) { throw 'goreleaser failed' }

$deb = Get-ChildItem $distDir -Filter '*linux_arm64.deb' | Select-Object -First 1
if (-not $deb) { throw 'deb 未生成' }
Write-Host "[OK] 产物: $($deb.FullName) ($([math]::Round($deb.Length/1MB,1)) MB)" -ForegroundColor Green
Write-Host "部署: scp $($deb.FullName) ${HostAlias}:/tmp/ && ssh ${HostAlias} 'sudo dpkg -i /tmp/$($deb.Name)'"
