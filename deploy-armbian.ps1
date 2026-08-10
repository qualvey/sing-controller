# deploy-armbian.ps1 — 本地 CD：构建 arm64 deb → 推送到 armbian → 安装 → 重启服务
# 前置：ssh armbian 可免密登录（~/.ssh/config 别名）
# sudo：自动检测免密；若需密码，按提示先配置 NOPASSWD（一次性）或手动安装
# 用法：pwsh ./deploy-armbian.ps1
$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
Set-Location $root

$hostAlias = 'armbian'

# 1. 构建 linux/arm64 deb（复用 build-arm64.ps1，产物在 dist/）
Write-Host '[*] 构建 arm64 deb ...'
& "$root/build-arm64.ps1"
if ($LASTEXITCODE -ne 0) { throw '构建失败' }

# 2. 取最新 deb
$deb = Get-ChildItem dist -Filter '*linux_arm64.deb' | Sort-Object LastWriteTime -Descending | Select-Object -First 1
if (-not $deb) { throw 'dist 下没有 linux_arm64.deb' }
Write-Host "[*] 产物: $($deb.Name) ($([math]::Round($deb.Length/1MB,1)) MB)"

# 3. 推送
Write-Host "[*] scp -> ${hostAlias}:/tmp/ ..."
scp "$($deb.FullName)" "${hostAlias}:/tmp/"
if ($LASTEXITCODE -ne 0) { throw 'scp 失败' }

# 4. sudo 免密检测
# Write-Host '[*] 检查 sudo ...'
# $sudoStatus = ssh $hostAlias 'sudo -n true && echo SUDO_OK || echo SUDO_NEED_PASSWORD'
# if ($sudoStatus -match 'SUDO_NEED_PASSWORD') {
#     Write-Host ''
#     Write-Host '[!] armbian 上 sudo 需要密码（非交互无法输入）。二选一：'
#     Write-Host '    a) 配置免密（一次性，手动执行一次）：'
#     Write-Host "       ssh $hostAlias 'echo \"`$USER ALL=(ALL) NOPASSWD: ALL\" | sudo tee /etc/sudoers.d/`$USER'"
#     Write-Host '    b) 手动安装（跳过本脚本后续）：'
#     Write-Host "       ssh $hostAlias 'sudo dpkg -i /tmp/$($deb.Name) && sudo systemctl restart sing-controller'"
#     throw 'sudo 需要密码，请按上面提示处理后再跑本脚本'
# }

# 5. 安装 + 重启 + 状态
Write-Host '[*] dpkg -i + restart sing-controller ...'
ssh -t $hostAlias 'sudo apt install -y /tmp/sing-controller_0.7.0-beta.1-SNAPSHOT-b6c009a_linux_arm64.deb'
if ($LASTEXITCODE -ne 0) { throw '安装/重启失败' }
ssh -t $hostAlias 'sudo systemctl restart sing-controller && sudo systemctl status sing-controller --no-pager -n 20'

# 6. 清理远端临时包
ssh $hostAlias "rm -f /tmp/$($deb.Name)"

Write-Host "[OK] 已部署: $($deb.Name)"
Write-Host "     检查: ssh $hostAlias 'journalctl -u sing-controller -n 20'"
