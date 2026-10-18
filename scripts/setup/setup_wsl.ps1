# 检查/安装 WSL2 并应用 .wslconfig 内存限制
$ErrorActionPreference = 'Stop'

Write-Host '[1/3] 检查 WSL 状态...'
wsl --status
if ($LASTEXITCODE -ne 0) {
  Write-Host 'WSL 未安装，开始安装（需要管理员权限，完成后重启）...'
  wsl --install -d Ubuntu-22.04
  Write-Host '安装完成后请重启电脑，再运行本脚本'
  exit 1
}

Write-Host '[2/3] 应用 .wslconfig 内存限制...'
$src = Join-Path $PSScriptRoot '..\..\wsl\.wslconfig'
$dst = Join-Path $env:USERPROFILE '.wslconfig'
if (-not (Test-Path $dst)) {
  Copy-Item $src $dst
  Write-Host "已复制 .wslconfig 到 $dst"
} else {
  Write-Host ".wslconfig 已存在，跳过（如需更新请手动覆盖）"
}

Write-Host '[3/3] 重启 WSL 使配置生效...'
wsl --shutdown
Write-Host '完成。重新打开 WSL 后运行: wsl -- sh -c "free -m" 验证内存约 5GB'
