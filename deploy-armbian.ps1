# deploy-armbian.ps1 — 本地 CD：构建 arm64 deb → 推送到 armbian → 安装 → 重启服务
# 前置：ssh armbian 可免密登录（~/.ssh/config 别名）
# 用法：pwsh ./deploy-armbian.ps1

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
Set-Location $root

$hostAlias = 'armbian'

# 1. 构建 linux/arm64 deb（产物在 dist/）
# Write-Host '[*] 构建 arm64 deb ...'
# & "$root/build-arm64.ps1"
# if ($LASTEXITCODE -ne 0) { throw '构建失败' }

# 2. 取最新 deb
$deb = Get-ChildItem dist -Filter '*linux_arm64.deb' | Sort-Object LastWriteTime -Descending | Select-Object -First 1
if (-not $deb) { throw 'dist 下没有 linux_arm64.deb' }
Write-Host "[*] 产物: $($deb.Name) ($([math]::Round($deb.Length/1MB,1)) MB)"

# 3. 推送至远端 /tmp/
Write-Host "[*] scp -> ${hostAlias}:/tmp/ ..."
scp "$($deb.FullName)" "${hostAlias}:/tmp/$($deb.Name)"
if ($LASTEXITCODE -ne 0) { throw 'scp 失败' }

# 4. 远程执行：强制覆盖安装 + 重启服务 + 查看状态 + 清理临时包
Write-Host '[*] 正在安装包并重启 sing-controller 服务 ...'
$remoteCmd = "sudo dpkg -i /tmp/$($deb.Name) && sudo systemctl restart sing-controller && sudo systemctl status sing-controller --no-pager -n 20; rm -f /tmp/$($deb.Name)"

ssh -t $hostAlias $remoteCmd
if ($LASTEXITCODE -ne 0) { throw '远程部署过程出错' }

ssh $hostAlias  "rm -rf /tmp/$($deb.Name)"  # 清理远程临时包
Write-Host "[OK] 已成功部署: $($deb.Name)"
Write-Host "     日志查看命令: ssh $hostAlias 'journalctl -u sing-controller -n 20'"