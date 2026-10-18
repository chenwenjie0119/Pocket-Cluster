#!/bin/bash
set -e
# 端口可由环境变量覆盖（compose 中按需设置）
HOST="${RPC_HOST:-0.0.0.0}"
PORT="${RPC_PORT:-50052}"
echo "[pocket-worker] llama-rpc-server on ${HOST}:${PORT}"
exec llama-rpc-server --host "$HOST" --port "$PORT" "$@"
