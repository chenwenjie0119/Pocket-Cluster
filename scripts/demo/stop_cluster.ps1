# Pocket-Cluster 一键停止
$ErrorActionPreference = 'Continue'
$root = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent

Write-Host '[1/4] 停 MuMu worker...'
adb shell "pkill -f llama-rpc-server" 2>$null

Write-Host '[2/4] 停 Docker worker...'
Push-Location (Join-Path $root 'docker')
docker compose down
Pop-Location

Write-Host '[3/4] 停 Master...'
Get-Process -Name 'pocketd' -ErrorAction SilentlyContinue | Stop-Process -Force

Write-Host '[4/4] 完成'
