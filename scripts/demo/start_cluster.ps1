# Pocket-Cluster 一键启动（Docker workers + MuMu worker + Master）
$ErrorActionPreference = 'Stop'
$root = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent

Write-Host '[1/5] 启动 Docker worker...'
Push-Location (Join-Path $root 'docker')
docker compose up -d --build
Pop-Location

Write-Host '[2/5] 检查 MuMu adb...'
adb devices

Write-Host '[3/5] 启动 Master (pocketd)...'
$masterExe = Join-Path $root 'master\pocketd.exe'
if (-not (Test-Path $masterExe)) {
  Write-Host '首次运行，编译 Master...'
  Push-Location (Join-Path $root 'master')
  go build -o pocketd.exe ./cmd/pocketd
  Pop-Location
}
Start-Process -FilePath $masterExe -WindowStyle Minimized

Write-Host '等待 Master 启动...'
$masterReady = $false
for ($i = 0; $i -lt 10; $i++) {
  Start-Sleep -Seconds 1
  try {
    Invoke-RestMethod -Uri 'http://127.0.0.1:8080/api/v1/health' -TimeoutSec 1 | Out-Null
    $masterReady = $true
    break
  } catch {}
}
if (-not $masterReady) {
  throw 'Master 未在 10 秒内就绪，请检查 pocketd.exe 是否正常启动'
}

Write-Host '[4/5] 注册节点（演示用，3 个 worker）...'
$nodes = @(
  @{id='worker-1'; addr='127.0.0.1:50052'; arch='linux-x86_64'; mem_total=1536; mem_free=1400; cpus=2; capabilities=@('cpu')},
  @{id='worker-2'; addr='127.0.0.1:50053'; arch='linux-x86_64'; mem_total=1536; mem_free=1400; cpus=2; capabilities=@('cpu')},
  @{id='worker-3'; addr='127.0.0.1:50054'; arch='android-x86_64'; mem_total=1200; mem_free=900; cpus=4; capabilities=@('cpu')}
)
foreach ($n in $nodes) {
  $json = $n | ConvertTo-Json -Compress
  Invoke-RestMethod -Uri 'http://127.0.0.1:8080/api/v1/nodes/register' -Method Post -Body $json -ContentType 'application/json' | Out-Null
}

Write-Host '[5/5] 健康检查...'
Start-Sleep -Seconds 2
Invoke-RestMethod -Uri 'http://127.0.0.1:8080/api/v1/health'
