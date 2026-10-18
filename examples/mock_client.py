#!/usr/bin/env python3
"""Pocket-Cluster Mock Client

模拟推理客户端：向 Master 请求推理计划，并演示张量切分请求在
各节点间的分发过程。不加载真实模型，纯演示。

用法:
    python examples/mock_client.py
    python examples/mock_client.py --model-size 1.5

前置: Master 已在运行，且至少 1 个节点在线（可用 mock_worker.py 模拟）。
"""
import argparse
import json
import sys
import time
import urllib.request

MASTER = "http://127.0.0.1:8080"


def api(path, method="GET", body=None):
    req = urllib.request.Request(MASTER + path, method=method)
    data = None
    if body is not None:
        data = json.dumps(body).encode()
        req.add_header("Content-Type", "application/json")
    with urllib.request.urlopen(req, timeout=5) as r:
        return json.loads(r.read())


def main():
    parser = argparse.ArgumentParser(description="Pocket-Cluster mock client")
    parser.add_argument("--model-size", type=float, default=1.5, help="模型大小 GB")
    args = parser.parse_args()

    try:
        health = api("/api/v1/health")
        print(f"== Cluster Health: {health['status']}, nodes={health['nodes']} ==")
    except Exception as e:
        print(f"Master 不可达，请先启动 pocketd: {e}")
        sys.exit(1)

    plan = api("/api/v1/plan", "POST", {"model_size_gb": args.model_size})
    if not plan.get("enough"):
        print(f"内存不足: {plan['reason']}")
        sys.exit(1)

    print(f"\n== 推理计划（模型 {plan['model_size_gb']}GB）==")
    for ep, ratio in zip(plan["rpc_endpoints"], plan["ts"]):
        print(f"  {ep:24s} ts={ratio}")
    print(f"llama-cli 参数: {plan['cli_args']}")

    print("\n== 模拟张量切分请求分发 ==")
    for ep, ratio in zip(plan["rpc_endpoints"], plan["ts"]):
        host, port = ep.replace("tcp://", "").split(":")
        print(f"  -> 连接 {host}:{port}，分配 {ratio * 100:.0f}% 层 ... OK")
        time.sleep(0.3)
    print("推理演示完成（Mock 模式，未加载真实模型）")


if __name__ == "__main__":
    main()
