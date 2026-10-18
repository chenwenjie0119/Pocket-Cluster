#!/usr/bin/env python3
"""Pocket-Cluster 节点健康检查：探测 Master 与各 worker RPC 端口连通性"""
import json
import socket
import sys
import urllib.request

NODES = [
    ("worker-1", "127.0.0.1", 50052),
    ("worker-2", "127.0.0.1", 50053),
    ("worker-3 (MuMu)", "127.0.0.1", 50054),
]


def probe(host, port, timeout=3):
    try:
        with socket.create_connection((host, port), timeout=timeout):
            return True
    except OSError:
        return False


def check_master(master, timeout=3):
    """请求 Master /api/v1/health，返回 (ok, 描述信息)。"""
    url = master.rstrip("/") + "/api/v1/health"
    try:
        with urllib.request.urlopen(url, timeout=timeout) as resp:
            data = json.loads(resp.read().decode("utf-8"))
        return True, f"status={data.get('status', '?')}, online_nodes={data.get('nodes', 0)}"
    except Exception as e:
        return False, str(e)


def main():
    master = sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:8080"
    print(f"== Pocket-Cluster Health Check ==\nMaster: {master}")
    ok = True

    m_ok, m_info = check_master(master)
    ok &= m_ok
    print(f"[{'OK ' if m_ok else 'DOWN'}] master               {master} -> {m_info}")

    for name, host, port in NODES:
        alive = probe(host, port)
        ok &= alive
        print(f"[{'OK ' if alive else 'DOWN'}] {name:16s} {host}:{port}")

    print("\nALL OK" if ok else "\nSOME CHECKS FAILED")


if __name__ == "__main__":
    main()
