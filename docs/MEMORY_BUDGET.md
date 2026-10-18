# 内存预算（16GB 宿主机 · 红线 90%）

## 硬性指标

- 物理内存：16GB（实际可用约 15.9GB）
- 红线：90% ≈ **14.4GB**（超过即可能卡死）
- 推荐目标：**13.5GB（86%）**，留 1GB 安全余量

## 预算表

| 组件 | 预算上限 | 说明 |
|---|---|---|
| Windows 基线（系统/桌面/驱动/输入法） | 4.0 GB | 不可压缩；启动集群前建议关闭浏览器多标签 |
| WSL2 总额（.wslconfig memory=5GB） | 5.0 GB | 双保险：配额 + 容器 mem_limit |
| └ Docker daemon | 0.4 GB | 由 WSL2 配额覆盖 |
| └ worker-1 (mem_limit 1536m) | 1.5 GB | 模型层权重 + KV cache |
| └ worker-2 (mem_limit 1536m) | 1.5 GB | 同上 |
| └ master 容器（可选） | 0.3 GB | 推荐改为 Windows 原生，更省 |
| MuMu 模拟器 | 2.0 GB | 设置 RAM=2048MB |
| └ Android 系统 + rpc-server | ~1.2 GB | MuMu 进程整体 |
| 监控 + 杂项 | 0.5 GB | watchdog / health_check / curl |
| **合计** | **13.5 GB（86%）** | 低于红线 4 个百分点 |

## 三层限制落地

| 层 | 手段 | 位置 |
|---|---|---|
| WSL2 | memory=5GB | wsl/.wslconfig |
| 容器 | mem_limit=1536m + memswap_limit=1536m | docker/docker-compose.yml |
| 宿主机 | 88%/93% 自动降级 | scripts/monitor/memory_watchdog.ps1 |

## 自动降级策略（watchdog 实现）

| 阈值 | 动作 |
|---|---|
| >= 88% | 记录日志 + 桌面通知；`docker stop pocket-worker-2`（保留 worker-1 与 MuMu） |
| >= 93% | 停 MuMu worker（adb shell kill）+ 通知；若仍超，停 worker-1 |
| < 85% | 恢复策略（可选：docker start pocket-worker-2） |

> 降级顺序设计：先停"可快速重启"的 Docker worker-2（恢复成本最低），最后才动 MuMu（启动慢）。

## 如何验证

```powershell
# Windows 总内存
Get-CimInstance Win32_OperatingSystem | Select TotalVisibleMemorySize, FreePhysicalMemory
# WSL2 内存
wsl.exe -- sh -c "free -m"
# Docker 容器内存
docker stats --no-stream
# MuMu 进程内存
Get-Process | Where-Object {$_.ProcessName -match 'MuMu|Nemu'} | Measure-Object WorkingSet64 -Sum
```
