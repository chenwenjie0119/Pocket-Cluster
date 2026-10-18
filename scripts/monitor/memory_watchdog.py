#!/usr/bin/env python3
"""Pocket-Cluster 宿主机内存监控（Python 版，逻辑同 memory_watchdog.ps1）
依赖: psutil (pip install psutil)
"""
import os
import subprocess
import time
from datetime import datetime

try:
    import psutil
except ImportError:
    raise SystemExit("缺少 psutil，请先 pip install psutil")

LOG_DIR = os.path.join(os.path.expanduser("~"), ".pocket-cluster")
LOG_FILE = os.path.join(LOG_DIR, "memory.log")
INTERVAL = 5
HIGH_PCT = 88
CRITICAL_PCT = 93
WORKER2 = "pocket-worker-2"
WORKER1 = "pocket-worker-1"


def log(msg):
    line = f"[{datetime.now():%Y-%m-%d %H:%M:%S}] {msg}"
    with open(LOG_FILE, "a", encoding="utf-8") as f:
        f.write(line + "\n")
    print(line, flush=True)


def mem_pct():
    vm = psutil.virtual_memory()
    return round(vm.percent, 1), round(vm.available / 1024**3, 2)


def wsl_mem_mb():
    try:
        out = subprocess.run(
            ["wsl.exe", "--", "sh", "-c", "free -m | awk '/Mem:/{print $3}'"],
            capture_output=True, text=True, timeout=5)
        return int(out.stdout.strip()) if out.stdout.strip().isdigit() else -1
    except Exception:
        return -1


def docker_stats():
    try:
        out = subprocess.run(
            ["docker", "stats", "--no-stream", "--format", "{{.Name}} {{.MemUsage}}"],
            capture_output=True, text=True, timeout=15)
        return [l for l in out.stdout.splitlines() if l.strip()]
    except Exception:
        return []


def mumu_mb():
    total = 0
    for p in psutil.process_iter(["name"]):
        name = (p.info.get("name") or "").lower()
        if "mumu" in name or "nemu" in name:
            total += p.memory_info().rss
    return round(total / 1024**2)


def stop_container(name):
    log(f"降级: 停止容器 {name}")
    subprocess.run(["docker", "stop", name], capture_output=True)


def stop_mumu():
    log("降级: 停止 MuMu Android worker")
    subprocess.run(["adb", "shell", "pkill", "-f", "llama-rpc-server"], capture_output=True)


def main():
    os.makedirs(LOG_DIR, exist_ok=True)
    log(f"看门狗启动 interval={INTERVAL}s high={HIGH_PCT}% critical={CRITICAL_PCT}%")
    while True:
        pct, free_gb = mem_pct()
        wsl_mb = wsl_mem_mb()
        dk = docker_stats()
        mmb = mumu_mb()
        msg = f"内存 {pct}% (free {free_gb}GB) | WSL2 {wsl_mb}MB | MuMu {mmb}MB"
        if dk:
            msg += " | Docker: " + "; ".join(dk[:6])
        log(msg)

        if pct >= CRITICAL_PCT:
            log(f"!!! 超过 {CRITICAL_PCT}% 临界线，强降级")
            stop_mumu()
            stop_container(WORKER2)
            stop_container(WORKER1)
        elif pct >= HIGH_PCT:
            log(f"!! 超过 {HIGH_PCT}%，先停 worker-2")
            stop_container(WORKER2)
            if pct >= HIGH_PCT + 3:
                stop_mumu()
        time.sleep(INTERVAL)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        pass
