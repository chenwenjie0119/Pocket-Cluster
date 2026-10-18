# 真机 Android / 树莓派节点接入（Node Onboarding）

把真实设备（Android 手机、树莓派）变成 Pocket-Cluster 的推理 Worker。核心原则：设备上运行 llama.cpp 的 `rpc-server`（编译时开启 `-DLLAMA_RPC=ON`），然后按与 Docker Worker 相同的方式向 Master 注册并上报心跳。

## 1. 通用前置要求

- 设备与 Master 处于**同一局域网/网段**，能互相访问。
- Master 的 HTTP 端口（默认 `8080`）与各 Worker 的 RPC 端口（默认 `50052`）在防火墙上放行。
- 心跳脚本要求 Python 3.8+（Android 用 Termux 自带 Python，树莓派用系统 Python）。

## 2. Android 真机接入（Termux）

### 2.1 构建 rpc-server

方案一（源码构建，推荐）：

```bash
# Termux 内先安装依赖
pkg install git cmake ninja python
# 克隆 llama.cpp 并开启 RPC 构建
git clone https://github.com/ggerganov/llama.cpp.git && cd llama.cpp
cmake -B build -DLLAMA_RPC=ON -DCMAKE_BUILD_TYPE=Release
cmake --build build --config Release -j $(nproc)
# 产物：build/bin/llama-rpc-server
```

方案二（预编译）：若社区/官方提供对应架构（`arm64-v8a`）的 `llama-rpc-server` 二进制，直接下载并赋予可执行权限即可，跳过编译。

### 2.2 启动与注册

```bash
# 监听 0.0.0.0:50052，局域网内其他设备可连
./build/bin/llama-rpc-server --host 0.0.0.0 --port 50052

# 向 Master 注册（arch 填 android-x86_64 / android-arm64）
curl -s -X POST http://<MASTER_IP>:8080/api/v1/nodes/register \
  -H 'Content-Type: application/json' \
  -d '{"id":"android-1","addr":"<手机局域网IP>:50052","arch":"android-arm64","mem_total":4096,"mem_free":3000,"cpus":8,"capabilities":["cpu"]}'
```

### 2.3 与 MuMu 模拟器的差异

- **同用 `android/keepalive` 心跳脚本**：`keepalive.sh` / `watchdog.py` 与 `setup_mumu.ps1` 的差异仅在部署路径（Termux 目录 vs 模拟器内路径）与 `adb forward` 的依赖：
  - MuMu 走 `adb forward tcp:50052 tcp:50052`，宿主机访问 `127.0.0.1:50052` 即到模拟器内节点；真机无需 forward，直接用手机局域网 IP。
  - 真机直接用 Termux 的 cron/termux-job-scheduler 保活；MuMu 由 Windows 侧 `setup_mumu.ps1` 管理。
- 心跳频率与字段完全一致，Master 无需区分来源。

## 3. 树莓派接入（Raspberry Pi OS）

### 3.1 直接编译（推荐，独立性强）

```bash
sudo apt update && sudo apt install -y git cmake build-essential python3
git clone https://github.com/ggerganov/llama.cpp.git && cd llama.cpp
cmake -B build -DLLAMA_RPC=ON -DCMAKE_BUILD_TYPE=Release
cmake --build build --config Release -j $(nproc)
./build/bin/llama-rpc-server --host 0.0.0.0 --port 50052
```

### 3.2 复用 docker/worker 镜像（容器化）

- 在树莓派（arm64）上安装 Docker，直接使用 `docker/worker` 构建的 llama.cpp rpc-server 镜像，与 Docker Worker 完全一致的部署方式：

```bash
cd docker && docker compose up -d --build worker
```

- 需要把 `docker-compose.yml` 中 worker 的 `extra_hosts` / 网络配置指向 Master 可达地址，并确保 Master HTTP 端口对容器网络开放。

### 3.3 资源预算建议

- 树莓派 4（4GB/8GB RAM）：适合 Qwen2.5-0.5B / 1.5B 级别模型切片；建议 `mem_total` 按实际内存报，`mem_free` 保守取值（系统占用 +30% 余量）。
- 树莓派 3 / 2GB 以下：不建议参与切片（比例下限 10% 会放大端侧延迟），可作纯客户端或后续小模型专用节点。

## 4. 接入后验证

1. 注册：`POST /api/v1/nodes/register` 返回成功，或 Dashboard 出现新节点。
2. 心跳：节点每 10 秒上报，`GET /api/v1/health` 的 `nodes` 计数增加；超过 30 秒无心跳会被 TTL 自动剔除。
3. 调度：`POST /api/v1/plan` 的 `rpc_endpoints` / `ts` 中包含新节点，且 `cli_args` 中 `--rpc tcp://<设备IP>:50052` 可见。
4. 端到端：在新节点上启动 `llama-rpc-server` 后，用 `examples/chat_demo.sh` 或 `mock_client.py` 验证推理链路。

## 5. 常见问题

- **RPC 连接超时**：确认设备 IP 可达、防火墙放行 50052，且 rpc-server 以 `--host 0.0.0.0` 启动（不要只监听 localhost）。
- **心跳脚本连不上 Master**：确认 `MASTER` 地址填写局域网 IP 而非 `127.0.0.1`（Master 不在本机时）。
- **节点频繁离线**：检查设备息屏/休眠是否杀掉进程，Android 请开启 Termux 唤醒锁或使用 watchdog 保活。
