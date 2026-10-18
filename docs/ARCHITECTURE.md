# Pocket-Cluster 架构设计

## 1. 总体架构

Pocket-Cluster 采用"控制面 / 数据面"分离的双平面架构：

- **控制面（Control Plane）**：Master（pocketd，Go 编写）通过 HTTP+JSON 管理节点注册、心跳、健康检查，并输出"推理会话推荐参数"（即应连接哪些 worker、按什么比例切分张量）。
- **数据面（Data Plane）**：llama.cpp 原生 RPC。客户端（llama-cli / llama-server）通过 `--rpc tcp://<host>:50052` 连接多个 `llama-rpc-server`，llama.cpp 内部完成张量（按层）切分与聚合。

为什么不自研通信协议？因为 llama.cpp 的 RPC 已经解决了最难的部分：跨设备张量并行、每层权重分发、激活值聚合、设备回退。自研 gRPC/ZeroMQ 只做控制面属于重复造轮子；控制面用 HTTP+JSON 是因为它跨语言、可 curl 调试、Go 标准库零依赖，内存占用最小。

## 2. 节点角色

| 角色 | 运行位置 | 进程 | 职责 |
|---|---|---|---|
| Master | Windows 原生（推荐）或 WSL2 容器 | pocketd (Go) | 节点注册表、心跳、健康检查、调度建议、内存红线监控 |
| Worker (Linux) | WSL2 内 Docker 容器 ×2 | llama-rpc-server | 提供 CPU 推理后端，监听 :50052 |
| Worker (Android) | MuMu 模拟器内（Termux 或原生进程） | llama-rpc-server | 同上，通过 adb forward 暴露 :50052 |
| 推理客户端 | Windows 原生 | llama-cli / llama-server | 按 Master 建议的 --rpc 参数发起推理 |

## 3. 通信协议

### 3.1 控制面（HTTP + JSON）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | /api/v1/nodes/register | 节点注册：{id, addr, arch, mem_total, mem_free, cpus, capabilities} |
| POST | /api/v1/nodes/heartbeat | 节点心跳：{id, mem_free, status}，30 秒无心跳自动剔除 |
| GET | /api/v1/nodes | 列出全部节点及其状态 |
| GET | /api/v1/health | 集群健康摘要 |
| POST | /api/v1/plan | 请求推理计划：{model_size_gb, prefer} → 返回推荐 --rpc 参数与切分比例 |
| GET | /api/v1/plan/default | 返回当前默认推荐参数 |

### 3.2 数据面（llama.cpp RPC，TCP）

- Worker 运行 `llama-rpc-server --host 0.0.0.0 --port 50052`。
- 客户端示例：
  ```bash
  llama-cli -m models/qwen2.5-1.5b-instruct-q4_k_m.gguf \
    --rpc tcp://127.0.0.1:50052 \    # worker-1 (docker 映射端口)
    --rpc tcp://127.0.0.1:50053 \    # worker-2 (docker 映射端口)
    --rpc tcp://127.0.0.1:50054      # worker-3 (MuMu, adb forward)
  ```
- 张量切分：llama.cpp 默认按张量轮流分配；可通过 `-ts` 指定各节点比例（如 `-ts 0.3,0.3,0.4`）。Master 的 /api/v1/plan 会根据各节点空闲内存自动生成 `-ts`。

## 4. 为什么"流水线/张量并行"而不是"数据并行"

- **数据并行**需要每节点一份完整模型副本，16GB 机器根本放不下几个节点；且单用户交互式推理没有高吞吐需求，数据并行收益低。
- **张量并行（按层切分）**：llama.cpp RPC 按层分发权重，每节点只需持有部分层，网络只传输激活值（每层输出），对带宽要求远低于权重全量传输。局域网 100Mbps 即可流畅跑 1.5B-3B 模型。
- **结论（MVP）**：张量并行 + Master 端做路由（把不同推理请求分发给不同的 rpc 组合）。带宽敏感时把切分比例调向"同机 worker"（Docker 两个容器走 WSL2 内部网络，几乎零延迟）。

## 5. 内存安全设计（三层限制）

1. **WSL2 层**：`wsl/.wslconfig` 设置 `memory=5GB`，从根源封死 WSL2 侧内存。
2. **容器层**：docker-compose 中每个 worker `mem_limit: 1536m` + `memswap_limit: 1536m`（禁止 swap 逃逸）。
3. **宿主机层**：`scripts/monitor/memory_watchdog.ps1` 实时监控，超过 88% 自动降级（先停 worker-2），超过 93% 强停 MuMu worker 并桌面通知。

## 6. 目录职责

```text
master/      控制面实现（Go）
docker/      worker/master 镜像与编排
wsl/         WSL2 内存配置
android/     Android worker 部署与保活
scripts/     宿主机运维（监控/启停/健康检查）
examples/    客户端调用示例
docs/        架构/指南/预算/排障
```
