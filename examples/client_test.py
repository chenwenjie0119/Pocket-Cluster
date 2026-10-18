#!/usr/bin/env python3
"""Pocket-Cluster 客户端演示：查询 Master 推理计划并打印 llama-cli 命令"""
import json
import sys
import urllib.request

MASTER = "http://127.0.0.1:8080"
MODEL_SIZE_GB = 1.0  # 你的 GGUF 模型文件大小（GB）


def api(path, method="GET", body=None):
    req = urllib.request.Request(MASTER + path, method=method)
    data = None
    if body is not None:
        data = json.dumps(body).encode()
        req.add_header("Content-Type", "application/json")
    with urllib.request.urlopen(req, data=data, timeout=5) as r:
        return json.loads(r.read())


def main():
    try:
        health = api("/api/v1/health")
        print(f"== Cluster Health: {health['status']}, nodes={health['nodes']} ==")
    except Exception as e:
        print(f"Master 不可达: {e}")
        sys.exit(1)

    nodes = api("/api/v1/nodes").get("nodes", [])
    print("\n在线节点:")
    for n in nodes:
        print(f"  - {n['id']:10s} {n['addr']:22s} mem_free={n['mem_free']}MB")

    plan = api("/api/v1/plan", "POST", {"model_size_gb": MODEL_SIZE_GB})
    print("\n== 推理计划 ==")
    print(json.dumps(plan, ensure_ascii=False, indent=2))

    if plan.get("enough"):
        print(f"\n推荐命令:\n  llama-cli -m models/YOUR_MODEL.gguf {plan['cli_args']} -p \"Hi\"")
    else:
        print(f"\n资源不足: {plan.get('reason')}")


if __name__ == "__main__":
    main()
