#!/usr/bin/env bash
# Pocket-Cluster 推理演示：请求 Master 计划，再用 llama-cli 连接集群
set -euo pipefail

MODEL="${MODEL:-models/qwen2.5-1.5b-instruct-q4_k_m.gguf}"
PROMPT="${PROMPT:-Hello, distributed cluster!}"
MASTER="${MASTER:-http://127.0.0.1:8080}"

echo "[demo] 向 Master 请求推理计划..."
PLAN=$(curl -s -X POST "$MASTER/api/v1/plan" \
  -H 'Content-Type: application/json' -d '{"model_size_gb":1.0}')
echo "$PLAN" | python3 -m json.tool

CLI_ARGS=$(echo "$PLAN" | python3 -c \
  "import sys,json;print(json.load(sys.stdin).get('cli_args',''))")
if [ -z "$CLI_ARGS" ]; then
  echo "[demo] Master 判定资源不足，使用直连回退..."
  CLI_ARGS="--rpc tcp://127.0.0.1:50052 --rpc tcp://127.0.0.1:50053 --rpc tcp://127.0.0.1:50054 -ts 0.4,0.4,0.2"
fi

echo "[demo] 启动 llama-cli..."
llama-cli -m "$MODEL" $CLI_ARGS -p "$PROMPT"
