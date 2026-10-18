# Pocket-Cluster

**语言：** [English](./README.md) | [简体中文](./README_CN.md)

> 把旧设备、电子垃圾拼起来，跑本地大模型。
> 跨平台（Windows / macOS / Linux / Android / iOS）异构分布式大模型推理集群，MVP 阶段聚焦 Windows 单机模拟集群。

<blockquote style="background:#10151f; color:#e6e6e6; border-left:4px solid #22c55e; padding:12px 18px; border-radius:6px; margin:8px 0 24px;">
<b>💡 Why Pocket-Cluster?</b> 专为 <b>16GB 内存旧设备</b> 打造，支持 <b>Windows 原生、WSL2、Docker、MuMu 模拟器</b> 原生混合调度。无需显卡，用闲置算力跑分布式推理。
</blockquote>

## 为什么做

- exo / cake 等项目对 Windows 支持差、配置繁琐；Windows + Android 模拟器 + WSL2 的组合没人好好做。
- 目标：让 16GB 内存的旧 Windows 电脑，也能通过 Docker + WSL2 + MuMu 模拟出"1 Master + N Worker"的异构集群，跑 llama.cpp 推理。
- 最终愿景：笔记本、旧手机、电视盒子、树莓派……所有能联网的设备都能成为你的推理节点。

## 核心设计（30 秒看懂）

```
                 ┌──────────────────────────────┐
                 │   Master (pocketd, Go)        │
                 │   · 节点注册 / 心跳 / 健康检查  │
                 │   · 生成 llama.cpp --rpc 参数   │
                 │   · 内存红线 90% 自动降级        │
                 └──────────────┬───────────────┘
                                │ HTTP/JSON (控制面)
        ┌───────────────────────┼────────────────────────┐
        │                       │                        │
┌───────▼────────┐    ┌────────▼────────┐    ┌──────────▼──────────┐
│ Worker 1 (Linux)│    │ Worker 2 (Linux)│    │ Worker 3 (Android)   │
│ Docker 容器      │    │ Docker 容器      │    │ MuMu 模拟器内         │
│ llama-rpc-server│    │ llama-rpc-server│    │ llama-rpc-server      │
│ :50052          │    │ :50052          │    │ :50052 (adb forward)  │
└───────┬────────┘    └────────┬────────┘    └──────────┬──────────┘
        └──────────────────────┴───────────────────────┘
                          llama.cpp RPC (数据面, TCP)
```

- **控制面**：Master（Go，零第三方依赖）用 HTTP+JSON 管理节点注册、心跳、健康检查、调度建议。
- **数据面**：直接用 llama.cpp 原生的 RPC 机制（`llama-rpc-server` + `--rpc` 参数），不需要自研张量切分协议。llama.cpp 负责层间张量并行，Master 只负责告诉客户端"连哪些节点、按什么比例切"。
- **极简主义**：Master 编译后约 8MB、运行内存约 20MB；Worker 直接复用 llama.cpp 官方 rpc-server 二进制，零额外框架。

## 快速开始（Windows 16GB 内存，30 分钟）

> **选项 0：1 分钟极速验证（无需编译 llama.cpp）**
> 1. 启动 Master：`cd master && go build -o pocketd.exe ./cmd/pocketd && .\pocketd.exe`
> 2. 另开一个终端，模拟三个假节点：`python examples/mock_worker.py`
> 3. 浏览器打开 http://127.0.0.1:8080/ —— Dashboard 上三个绿色在线节点亮起
> 4. 模拟推理客户端：`python examples/mock_client.py` —— 打印张量切分参数与节点分发过程
> 仅需 Python 3.8+，全程不碰 llama.cpp，适合快速演示与 CI 冒烟测试。

> 前置要求：请确保 `adb` 和 `python` 已加入系统 PATH。

1. 安装 WSL2 + Docker Desktop（详见 docs/MVP_GUIDE.md）。
2. 把 `wsl/.wslconfig` 复制到 `%USERPROFILE%\.wslconfig`，然后 `wsl --shutdown` 重启 WSL。
3. `cd docker && docker compose up -d --build` 启动 2 个 Linux Worker。
   - 注：`docker-compose.yml` 中 master 服务属于 `profiles: ["optional"]`，默认不启动；Master 推荐 Windows 原生运行，容器化版本仅供参考（`docker/master/Dockerfile` 已提供）。
4. 按 `android/setup_mumu.ps1` 配置 MuMu 模拟器并部署 Android Worker。
5. `go build -o pocketd.exe ./cmd/pocketd && ./pocketd.exe` 启动 Master。
   - 启动后浏览器访问 http://127.0.0.1:8080/ 可查看节点仪表盘。
6. 运行 `python examples/client_test.py` 获取推荐 RPC 参数，或用 `examples/chat_demo.sh` 直接跑 llama-cli。
   - 注：`scripts/demo/start_cluster.ps1` 注册节点使用的是演示用初始内存值；生产环境应由 worker 定期通过 `POST /api/v1/nodes/heartbeat` 上报真实内存。

## 目录结构

```text
master/      Master 调度端（Go，标准库零依赖）
docker/      Worker / Master 容器化部署（llama.cpp rpc-server）
wsl/         WSL2 内存限制配置
android/     MuMu/Android Worker 部署与保活
scripts/     PowerShell/Python 监控、健康检查、一键启停
examples/    客户端调用示例
docs/        架构、MVP 指南、内存预算、排障
```

## 技术栈

| 组件 | 选型 | 理由 |
|---|---|---|
| Master | Go 1.21+，仅标准库 | 编译 8MB、内存 ~20MB、零依赖部署 |
| Worker | llama.cpp rpc-server | 官方支持 RPC 张量并行，CPU 推理内存可控 |
| 控制面协议 | HTTP + JSON | 跨平台、可调试、无代码生成负担 |
| 数据面协议 | llama.cpp RPC (TCP :50052) | 官方实现，按层切分，局域网带宽友好 |
| 监控 | PowerShell + Python | 宿主机原生能力，零安装 |

## 内存红线（不可违反）

- 宿主内存占用不得超过 90%（16GB 机器约 14.4GB）。
- 三层限制：`.wslconfig memory=5GB` → 容器 `mem_limit=1536m` → 监控脚本 88%/93% 自动降级。
- 详见 docs/MEMORY_BUDGET.md。

## Roadmap

- [x] MVP：单机模拟集群（Windows + Docker + MuMu，1 Master + 2/3 Worker）
- [x] 张量切分比例自动计算（按节点可用内存 / CPU 核数）
- [ ] 真机 Android / 树莓派节点接入
- [ ] 模型分片缓存与增量下发
- [x] 失败节点自动剔除与任务重试
- [ ] iOS / macOS 节点（Metal 后端）

## License

MIT License. 详见 LICENSE。

## Author

陈文杰（Chen Wenjie） · GitHub: [chenwenjie0119](https://github.com/chenwenjie0119)
