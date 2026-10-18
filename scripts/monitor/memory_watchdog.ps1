# Pocket-Cluster 宿主机内存监控与自动降级（PowerShell 版）
# 用法: powershell -ExecutionPolicy Bypass -File memory_watchdog.ps1 [-Interval 5] [-HighPct 88] [-CriticalPct 93]
param(
  [int]$Interval = 5,
  [int]$HighPct = 88,
  [int]$CriticalPct = 93
)
$ErrorActionPreference = 'SilentlyContinue'
$logDir = Join-Path $env:USERPROFILE '.pocket-cluster'
New-Item -ItemType Directory -Force -Path $logDir | Out-Null
$logFile = Join-Path $logDir 'memory.log'
$worker2 = 'pocket-worker-2'
$worker1 = 'pocket-worker-1'

function Write-Log($msg) {
  $line = "[{0}] {1}" -f (Get-Date -Format 'yyyy-MM-dd HH:mm:ss'), $msg
  Add-Content -Path $logFile -Value $line
  Write-Host $line
}

function Get-MemUsagePct {
  $os = Get-CimInstance Win32_OperatingSystem
  $total = $os.TotalVisibleMemorySize
  $free = $os.FreePhysicalMemory
  return [math]::Round(($total - $free) / $total * 100, 1), [math]::Round($free / 1MB, 2)
}

function Get-WslMemMB {
  $out = wsl.exe -- sh -c "free -m | awk '/Mem:/{print \$3}'" 2>$null
  if ($out -match '^\d+$') { return [int]$out } else { return -1 }
}

function Get-DockerMem {
  return docker stats --no-stream --format "{{.Name}}|{{.MemUsage}}" 2>$null
}

function Stop-Worker($name) {
  Write-Log "降级: 停止容器 $name (docker stop)"
  docker stop $name 2>$null | Out-Null
}

function Stop-MumuWorker {
  Write-Log "降级: 停止 MuMu Android worker"
  adb shell "pkill -f llama-rpc-server" 2>$null | Out-Null
}

Write-Log "内存看门狗启动 (interval=${Interval}s, high=${HighPct}%, critical=${CriticalPct}%)"
while ($true) {
  $pct, $freeGB = Get-MemUsagePct
  $wslMB = Get-WslMemMB
  $dockerLines = Get-DockerMem
  $mumuMB = [math]::Round(((Get-Process | Where-Object { $_.ProcessName -match 'MuMu|Nemu' } | Measure-Object WorkingSet64 -Sum).Sum / 1MB), 0)

  $summary = "内存使用 ${pct}% (free ${freeGB}GB) | WSL2 已用 ${wslMB}MB | MuMu ${mumuMB}MB"
  $dockerOneLine = ($dockerLines | Where-Object { $_ } | Select-Object -First 6) -join ' ; '
  if ($dockerOneLine) { $summary += " | Docker: $dockerOneLine" }
  Write-Log $summary

  if ($pct -ge $CriticalPct) {
    Write-Log "!!! 超过 ${CriticalPct}% 临界线，执行强降级"
    Stop-MumuWorker
    Stop-Worker $worker2
    Stop-Worker $worker1
    try {
      Add-Type -AssemblyName System.Windows.Forms
      [System.Windows.Forms.MessageBox]::Show("Pocket-Cluster 内存超过 ${CriticalPct}%，已停止全部 Worker！", 'Pocket-Cluster Watchdog')
    } catch {}
  } elseif ($pct -ge $HighPct) {
    Write-Log "!! 超过 ${HighPct}%，按优先级降级（先停 worker-2）"
    Stop-Worker $worker2
    if ($pct -ge ($HighPct + 3)) { Stop-MumuWorker }
  }
  Start-Sleep -Seconds $Interval
}
