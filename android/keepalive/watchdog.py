#!/usr/bin/env python3
"""Pocket-Cluster MuMu Worker 看门狗（宿主侧）

每 15 秒通过 adb 检查模拟器内 llama-rpc-server 是否存活，被杀则自动拉起。
配合 android/keepalive/keepalive.sh 形成"宿主侧 + 设备侧"双保险。

依赖: adb 在 PATH 中；MuMu 12 默认已开启 adb（端口 16384）。
"""
import subprocess
import sys
import time

ADB = "adb"
SERIAL = None                     # 多设备时指定，如 "127.0.0.1:16384"
REMOTE_BIN = "/data/local/tmp/llama-rpc-server"
RPC_PORT = 50052
CHECK_INTERVAL = 15


def adb(*args, serial=None):
    cmd = [ADB]
    if serial:
        cmd += ["-s", serial]
    cmd += list(args)
    try:
        r = subprocess.run(cmd, capture_output=True, text=True, timeout=10)
        return r.stdout.strip()
    except Exception as e:
        return f"ERR:{e}"


def main():
    serial = SERIAL or adb("get-serialno") or None
    print(f"[watchdog] serial={serial}, interval={CHECK_INTERVAL}s")
    while True:
        pid = adb("shell", "pidof", "llama-rpc-server", serial=serial)
        if not pid or "ERR" in pid or "error" in pid.lower():
            print("[watchdog] worker down, restarting...")
            adb("shell", "nohup", f"{REMOTE_BIN} --host 0.0.0.0 --port {RPC_PORT}",
                ">/data/local/tmp/rpc.log 2>&1 &", serial=serial)
            time.sleep(3)
            pid = adb("shell", "pidof", "llama-rpc-server", serial=serial)
            print(f"[watchdog] worker pid={pid or 'STILL DOWN'}")
        else:
            print(f"[watchdog] worker alive pid={pid}")
        time.sleep(CHECK_INTERVAL)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        sys.exit(0)
