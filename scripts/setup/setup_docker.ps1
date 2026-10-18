# 检查 Docker Desktop 并构建启动 worker 容器
$ErrorActionPreference = 'Stop'
docker info *> $null
if ($LASTEXITCODE -ne 0) {
  Write-Error 'Docker 不可用。请确认 Docker Desktop 已安装并切换到 WSL2 后端后重试。'
}
Write-Host '[1/2] 构建并启动 worker 容器...'
Push-Location (Join-Path $PSScriptRoot '..\..\docker')
docker compose up -d --build
Pop-Location
Write-Host '[2/2] 验证...'
docker ps --filter "name=pocket-worker"
docker stats --no-stream --format "table {{.Name}}\t{{.MemUsage}}\t{{.MemPerc}}"
