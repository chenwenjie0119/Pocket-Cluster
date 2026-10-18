#!/usr/bin/env python3
"""Pocket-Cluster Mock Worker

无需编译 llama.cpp 即可模拟推理节点：向 Master 注册并周期上报心跳。
默认同时模拟 worker-1 / worker-2 / worker-3 三个节点，打开 Dashboard
即可看到三个绿色在线节点。

用法:
    python examples/mock_worker.py                      # 模拟 3 个节点
    python examples/mock_worker.py --single worker-1   # 只模拟 1 个节点

前置: Master (pocketd) 已在 http://127.0.0.1:8080 运行。
"""
import argparse
import json
import random
import sys
import threading
import time
import urllib.request

MASTER = "http://127.0.0.1:8080"
HEARTBEAT_SEC = 10  # 心跳间隔，必须小于 Master 的 --node-ttl（默认 30 秒）

DEFAULT_NODES = [
    {"id": "worker-1", "addr": "127.0.0.1:50052", "arch": "linux-x86_64",
     "mem_total": 1536, "mem_free": 1400, "cpus": 2, "capabilities": ["cpu"]},
    {"id": "worker-2", "addr": "127.0.0.1:50053", "arch": "linux-x86_64",
     "mem_total": 1536, "mem_free": 1400, "cpus": 2, "capabilities": ["cpu"]},
    {"id": "worker-3", "addr": "127.0.0.1:50054", "arch": "android-x86_64",
     "mem_total": 1200, "mem_free": 900, "cpus": 4, "capabilities": ["cpu"]},
]


def api(path, payload):
    req = urllib.request.Request(
        MASTER + path,
        data=json.dumps(payload).encode(),
        headers={"Content-Type": "application/json"},
    )
    with urllib.request.urlopen(req, timeout=5) as r:
        return json.loads(r.read())


def run_node(node):
    try:
        api("/api/v1/nodes/register", node)
        print(f"[{node['id']}] registered -> {node['addr']} ({node['arch']})")
    except Exception as e:
        print(f"[{node['id']}] 注册失败，确认 Master 已启动: {e}")
        return

    free = node["mem_free"]
    while True:
        time.sleep(HEARTBEAT_SEC)
        # 模拟真实负载：空闲内存小幅波动
        free = max(64, min(node["mem_total"], int(free + random.uniform(-120, 120))))
        try:
            api("/api/v1/nodes/heartbeat", {"id": node["id"], "mem_free": free})
            print(f"[{node['id']}] heartbeat mem_free={free}MB")
        except Exception as e:
            print(f"[{node['id']}] 心跳失败: {e}")
            time.sleep(HEARTBEAT_SEC)


def main():
    parser = argparse.ArgumentParser(description="Pocket-Cluster mock worker")
    parser.add_argument("--single", metavar="NODE_ID", help="只模拟指定节点（worker-1/2/3）")
    args = parser.parse_args()

    nodes = DEFAULT_NODES
    if args.single:
        nodes = [n for n in nodes if n["id"] == args.single]
        if not nodes:
            print(f"未知节点: {args.single}")
            sys.exit(1)

    threads = [threading.Thread(target=run_node, args=(n,), daemon=True) for n in nodes]
    for t in threads:
        t.start()

    print(f"Mock 集群已启动（{len(nodes)} 个节点），打开 http://127.0.0.1:8080/ 查看 Dashboard")
    print("按 Ctrl+C 退出")
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("\nMock workers 已退出")


if __name__ == "__main__":
    main()
