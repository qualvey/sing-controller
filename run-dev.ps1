[CmdletBinding()]
param(
    [string]$SettingsPath = "",
    [switch]$SkipFrontend,
    [switch]$BuildOnly
)

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
$webDir = Join-Path $root 'controller/web'
$controllerDir = Join-Path $root 'controller'
$distDir = Join-Path $root 'dist'
$binaryPath = Join-Path $distDir 'sing-controller-dev.exe'

foreach ($tool in @('go', 'pnpm')) {
    if (-not (Get-Command $tool -ErrorAction SilentlyContinue)) {
        throw "$tool 未找到，请先安装并加入 PATH。"
    }
}

if ([string]::IsNullOrWhiteSpace($SettingsPath)) {
    $SettingsPath = Join-Path $distDir 'controller.json'
} elseif (-not [System.IO.Path]::IsPathRooted($SettingsPath)) {
    $SettingsPath = Join-Path $root $SettingsPath
}
$SettingsPath = [System.IO.Path]::GetFullPath($SettingsPath)
if (-not (Test-Path -LiteralPath $SettingsPath -PathType Leaf)) {
    throw "找不到 Controller 配置文件: $SettingsPath"
}

if (-not (Test-Path -LiteralPath $distDir -PathType Container)) {
    New-Item -ItemType Directory -Path $distDir -Force | Out-Null
}

if (-not $SkipFrontend) {
    Write-Host '[1/3] 构建 WebUI...' -ForegroundColor Cyan
    Push-Location $webDir
    try {
        if (-not (Test-Path -LiteralPath (Join-Path $webDir 'node_modules/.bin/vite.cmd'))) {
            Write-Host '安装前端依赖...' -ForegroundColor DarkCyan
            & pnpm install --frozen-lockfile
            if ($LASTEXITCODE -ne 0) { throw 'pnpm install 失败。' }
        }
        & pnpm run build
        if ($LASTEXITCODE -ne 0) { throw 'WebUI 构建失败。' }
    } finally {
        Pop-Location
    }
} else {
    Write-Host '[1/3] 跳过 WebUI 构建。' -ForegroundColor DarkGray
}

$version = (& git -C $root describe --tags --always --dirty 2>$null)
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($version)) {
    $version = 'dev'
} else {
    $version = $version.Trim()
}

Write-Host '[2/3] 编译 Windows Controller...' -ForegroundColor Cyan
$previousCgo = $env:CGO_ENABLED
try {
    $env:CGO_ENABLED = '0'
    Push-Location $controllerDir
    try {
        & go build -trimpath -ldflags "-s -w -X main.version=$version" -o $binaryPath .
        if ($LASTEXITCODE -ne 0) { throw 'Controller 编译失败。' }
    } finally {
        Pop-Location
    }
} finally {
    if ($null -eq $previousCgo) {
        Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
    } else {
        $env:CGO_ENABLED = $previousCgo
    }
}

if ($BuildOnly) {
    Write-Host "构建完成: $binaryPath" -ForegroundColor Green
    return
}

$settings = Get-Content -LiteralPath $SettingsPath -Raw | ConvertFrom-Json
$listenPort = 8080
if ($settings.server.listen_port) {
    $listenPort = [int]$settings.server.listen_port
}
$secret = [string]$settings.server.secret
$distPrefix = [System.IO.Path]::GetFullPath($distDir).TrimEnd('\') + '\'
$oldControllers = @(
    Get-CimInstance Win32_Process |
        Where-Object {
            $_.Name -like 'sing-controller*.exe' -and
            $_.ExecutablePath -and
            $_.ExecutablePath.StartsWith($distPrefix, [System.StringComparison]::OrdinalIgnoreCase)
        }
)
$oldPids = @($oldControllers | ForEach-Object { [int]$_.ProcessId })
$listeners = @(Get-NetTCPConnection -State Listen -LocalPort $listenPort -ErrorAction SilentlyContinue)
$foreignListeners = @($listeners | Where-Object { $oldPids -notcontains [int]$_.OwningProcess })
if ($foreignListeners.Count -gt 0) {
    $foreignPids = ($foreignListeners | ForEach-Object { $_.OwningProcess } | Sort-Object -Unique) -join ', '
    throw "端口 $listenPort 已被其他进程占用 (PID: $foreignPids)。请先关闭占用该端口的程序。"
}

if ($oldControllers.Count -gt 0) {
    Write-Host '[3/3] 停止旧版本项目 Controller...' -ForegroundColor Cyan
    $headers = @{}
    if (-not [string]::IsNullOrWhiteSpace($secret)) {
        $headers['X-Secret'] = $secret
    }
    if ($listeners.Count -gt 0) {
        try {
            Invoke-RestMethod -Uri "http://127.0.0.1:$listenPort/api/core/stop" -Method Post -Headers $headers -TimeoutSec 3 | Out-Null
        } catch {
            Write-Warning "无法通过 API 停止旧内核，将由旧 Controller 的进程托管机制结束它: $($_.Exception.Message)"
        }
    }
    foreach ($process in $oldControllers) {
        Stop-Process -Id $process.ProcessId -Force -ErrorAction SilentlyContinue
        Wait-Process -Id $process.ProcessId -Timeout 10 -ErrorAction SilentlyContinue
    }
} else {
    Write-Host '[3/3] 启动 Controller...' -ForegroundColor Cyan
}

Write-Host "启动: $binaryPath" -ForegroundColor Green
Write-Host "配置: $SettingsPath" -ForegroundColor DarkGray
Write-Host "监听端口: $listenPort (按 Ctrl+C 停止)" -ForegroundColor DarkGray
Push-Location $distDir
try {
    & $binaryPath -config $SettingsPath
    if ($LASTEXITCODE -ne 0) {
        throw "Controller 已退出，退出码: $LASTEXITCODE"
    }
} finally {
    Pop-Location
}
