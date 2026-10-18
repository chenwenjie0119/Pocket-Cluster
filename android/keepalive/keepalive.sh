#!/system/bin/sh
# Pocket-Cluster Android Worker 保活（模拟器内双进程守护）
# 每 10 秒检查 llama-rpc-server，被杀立即重启。
# 用法: adb shell "nohup sh /data/local/tmp/keepalive.sh >/dev/null 2>&1 &"
BIN=/data/local/tmp/llama-rpc-server
PORT=${RPC_PORT:-50052}

while true; do
  if ! pidof llama-rpc-server >/dev/null 2>&1; then
    echo "[keepalive] $(date) restart llama-rpc-server" >> /data/local/tmp/keepalive.log
    nohup "$BIN" --host 0.0.0.0 --port "$PORT" >/data/local/tmp/rpc.log 2>&1 &
  fi
  sleep 10
done
