# build-arm64.ps1 — 本地快速构建 linux/arm64 的 deb（部署到 nanopi 验证用）
# 用法：pwsh ./build-arm64.ps1
# 产物：dist/sing-controller_<version>_linux_arm64.deb（snapshot 版本，不发布）
$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot  # 脚本位于项目根（build-arm64.ps1）
Set-Location $root

if (-not (Get-Command goreleaser -ErrorAction SilentlyContinue)) {
    Write-Host '[!] goreleaser 未安装，尝试 go install ...'
    go install github.com/goreleaser/goreleaser/v2@latest
    $env:PATH = "$env:USERPROFILE\go\bin;$env:PATH"
}

# 临时配置：仅 arm64（从 .goreleaser.yaml 精简，tags/打包逻辑保持一致）
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

Write-Host "[*] goreleaser snapshot build (linux/arm64 deb only)..."
goreleaser release --snapshot --skip=publish --clean -f $cfgFile
if ($LASTEXITCODE -ne 0) { throw 'goreleaser failed' }

$deb = Get-ChildItem dist -Filter '*linux_arm64.deb' | Select-Object -First 1
if (-not $deb) { throw 'deb 未生成' }
Write-Host "[OK] 产物: $($deb.FullName) ($([math]::Round($deb.Length/1MB,1)) MB)"
Write-Host "部署: scp $($deb.FullName) ryu@nanopi:/tmp/ && ssh ryu@nanopi 'sudo dpkg -i /tmp/$($deb.Name)'"
