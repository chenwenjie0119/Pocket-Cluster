# Pocket-Cluster MVP 开发与部署指南（16GB Windows 极限环境）

目标：在一台 16GB 内存的 Windows 11 电脑上，用 Docker + WSL2 + MuMu 模拟器搭出 **1 个 Master + 至少 2 个 Worker**（可扩展至 3 个 Worker），跑通 llama.cpp 分布式推理，全程宿主机内存占用不超过 90%（约 14.4GB，推荐 13.5GB）。

## 0. 前置条件清单

| 项目 | 要求 |
|---|---|
| Windows | 11（22H2+，支持 WSL2） |
| 内存 | 16GB（本方案按此预算设计） |
| WSL2 | Ubuntu 22.04 已安装并设为默认发行版 |
| Docker Desktop | WSL2 后端模式（Settings → General → Use the WSL 2 based engine） |
| MuMu 模拟器 | MuMu 12（x86_64 Android 镜像） |
| 工具链 | Go 1.21+（Master 编译用）、Python 3.10+（监控/客户端用） |
| 模型 | 1.5B~3B 量化的 GGUF（如 Qwen2.5-1.5B-Instruct Q4_K_M，约 1GB） |

> 预算内模型：总权重（GB）必须小于"worker 可用内存总和 - 1GB"。
> 以本方案 3 个 worker 为例：worker-1 1.5GB + worker-2 1.5GB + MuMu ~1GB ≈ 4GB，可跑 3B Q4（约 2GB）模型，留 2GB 给 KV cache 与激活值。

## 1. 单机模拟集群拓扑（推荐 4 节点）

| 节点 | 名称 | 位置 | 内存上限 | RPC 端口（宿主视角） |
|---|---|---|---|---|
| Master | pocketd | Windows 原生 | 0.05 GB | HTTP :8080 |
| Worker-1 | pocket-worker-1 | WSL2 Docker 容器 | 1.5 GB | localhost:50052 |
| Worker-2 | pocket-worker-2 | WSL2 Docker 容器 | 1.5 GB | localhost:50053 |
| Worker-3 | pocket-mumu | MuMu 模拟器 | 1.2 GB | localhost:50054 (adb forward) |

> 4 节点 = 1 Master + 3 Worker，超过"至少 3 个测试节点（1 Master + 2 Worker）"的要求，同时演示 Docker 与 Android 双异构。

## 2. 资源限制策略（核心）

### 2.1 WSL2 全局限制

把 `wsl/.wslconfig` 复制到 `%USERPROFILE%\.wslconfig`：

```ini
[wsl2]
memory=5GB
processors=4
swap=2GB
localhostForwarding=true
```

执行 `wsl --shutdown` 后重新打开 WSL，配置生效。可用 `wsl.exe -- sh -c "free -m"` 验证（Mem total 应约 5120MB）。

### 2.2 Docker 容器限制

`docker/docker-compose.yml` 中每个 worker：

```yaml
mem_limit: 1536m
memswap_limit: 1536m   # 禁止 swap 逃逸到 WSL2
cpus: 2.0
```

验证：`docker stats` 观察 Mem Usage 与 LIMIT。

### 2.3 MuMu 模拟器内存

MuMu 12 设置 → 其他设置 → 内存：设为 **2048MB**。同时：
- 分辨率设为 1280×720 以下（降低显存/内存开销）；
- 关闭"后台渲染"之外不必要的特效；
- 不开启多开（多开会叠加内存）。

### 2.4 总内存预算（必须遵守）

| 组件 | 预算 |
|---|---|
| Windows 基线 | 4.0 GB |
| WSL2 总额 | 5.0 GB |
| MuMu | 2.0 GB |
| 监控/杂项 | 0.5 GB |
| **合计** | **~13.5 GB（86%）** |

## 3. 搭建步骤

### 步骤 A：启动 Docker Worker（2 个 Linux 节点）

```bash
cd docker
docker compose up -d --build
docker ps                     # 应看到 pocket-worker-1 / pocket-worker-2
docker stats --no-stream      # 确认内存未超限
```

### 步骤 B：部署 MuMu Android Worker

1. 启动 MuMu 模拟器（内存 2048MB）。
2. 在 MuMu 内安装 Termux（或直接使用 Android 原生可执行文件方案，见 android/README 说明）。
3. 将编译好的 x86_64 llama-rpc-server 二进制 push 进模拟器：
   ```powershell
   adb push llama-rpc-server /data/local/tmp/
   adb shell chmod +x /data/local/tmp/llama-rpc-server
   ```
   > MuMu 12 是 x86_64 镜像，需用 x86_64 的 llama.cpp Android 构建（NDK 编译，-DANDROID_ABI=x86_64 -DLLAMA_RPC=ON）。
4. 端口转发：
   ```powershell
   adb forward tcp:50054 tcp:50052
   ```
5. 启动 worker：
   ```bash
   adb shell "nohup /data/local/tmp/llama-rpc-server --host 0.0.0.0 --port 50052 >/dev/null 2>&1 &"
   ```
6. 保活：`android/keepalive/watchdog.py` 会每 15 秒检查一次，进程被杀自动拉起；`android/keepalive/keepalive.sh` 可在模拟器内做双进程守护。

### 步骤 C：启动 Master

```powershell
cd master
go build -o pocketd.exe ./cmd/pocketd
.\pocketd.exe --listen :8080
```

验证：`curl http://127.0.0.1:8080/api/v1/health` 返回各节点状态。

### 步骤 D：注册节点（模拟器内 worker 手动注册一次）

```bash
curl -X POST http://127.0.0.1:8080/api/v1/nodes/register \
  -H "Content-Type: application/json" \
  -d '{"id":"worker-1","addr":"127.0.0.1:50052","arch":"linux-x86_64","mem_total":1536,"mem_free":1400,"cpus":2,"capabilities":["cpu"]}'
# 同理注册 worker-2 (50053)、worker-3 (50054)
```

> 生产环境应让 worker 启动时自动注册；MVP 阶段先用 curl 或 `examples/client_test.py` 演示。

### 步骤 E：发起推理

```bash
# 方案 1：用 Master 推荐参数
python examples/client_test.py            # 打印推荐的 --rpc 与 -ts 参数
# 方案 2：直接用 llama-cli
llama-cli -m models/qwen2.5-1.5b-instruct-q4_k_m.gguf \
  --rpc tcp://127.0.0.1:50052 --rpc tcp://127.0.0.1:50053 --rpc tcp://127.0.0.1:50054 \
  -ts 0.4,0.4,0.2 -p "Hello, cluster!"
```

## 4. 极简技术栈选型论证

### 4.1 Master：Go，为什么不是 Rust？

| 维度 | Go | Rust |
|---|---|---|
| 编译体积 | ~8MB | ~3MB |
| 运行时内存 | ~20MB | ~10MB |
| 开发速度 | 极快（标准库 net/http 够用） | 慢（所有权/生命周期） |
| 部署 | 单静态二进制 | 单静态二进制 |
| 社区生态（控制面场景） | 丰富 | 一般 |

结论：两者都达标。MVP 选 **Go**（开发速度 + 标准库零依赖）；后续对内存极敏感场景可把 Master 用 Rust 重写（接口不变，协议是 HTTP+JSON，无缝替换）。

### 4.2 Worker：基于 llama.cpp RPC 改造，而非自研

- llama.cpp 自带 `llama-rpc-server`（构建参数 `-DLLAMA_RPC=ON`），提供跨设备张量并行。
- **需要自己写的东西**：不是通信协议，而是"节点管理 + 调度建议"（即 Master 控制面）与"保活/监控"（运维面）。
- 改造点（本项目已有）：
  1. Docker 化封装（docker/worker/Dockerfile）；
  2. Android 部署脚本与保活（android/）；
  3. Master 的 /api/v1/plan 自动生成 `--rpc`/`-ts`（master/）；
  4. 内存红线监控与自动降级（scripts/monitor/）。

### 4.3 如何获取 llama-rpc-server 二进制

llama.cpp 官方预编译 release 默认**不包含** `rpc-server`：RPC 为可选构建项，需显式开启 `-DLLAMA_RPC=ON` 编译。以下给出三种获取方式与模型 GGUF 下载方法。

**源码编译（通用命令）**
```bash
cmake -B build -DLLAMA_RPC=ON
cmake --build build --target llama-rpc-server
```

**① Windows 原生**
- 安装 CMake 与支持 C++17 的工具链（Visual Studio Build Tools 或 MinGW）；
- 在 llama.cpp 源码根目录执行上面的通用命令，产物为 `build/bin/llama-rpc-server.exe`。

**② WSL2 / Docker 容器内 Linux**
- 容器内准备工具链：`apt install build-essential cmake git`；
- 克隆 llama.cpp 后执行通用命令，产物为 `build/bin/llama-rpc-server`；
- 本项目 `docker/worker/Dockerfile` 已内置该构建流程，直接 `docker compose up -d --build` 即可。

**③ Android NDK 交叉编译（x86_64）**
- 需要 Android NDK，配置交叉工具链后执行：
```bash
cmake -B build-android -DCMAKE_TOOLCHAIN_FILE=$ANDROID_NDK/build/cmake/android.toolchain.cmake \
  -DANDROID_ABI=x86_64 -DANDROID_PLATFORM=android-24 -DLLAMA_RPC=ON
cmake --build build-android --target llama-rpc-server
```
- MuMu 12 为 x86_64 镜像，必须使用 x86_64 产物（勿用 arm64），push 到 `/data/local/tmp/` 后 `chmod +x` 即可运行。

**模型 GGUF 获取**

从 Hugging Face 下载量化后的 GGUF（如 Qwen2.5-1.5B-Instruct-GGUF）：
```bash
huggingface-cli download Qwen/Qwen2.5-1.5B-Instruct-GGUF --include "*.gguf" --local-dir models/
# 或直接用 curl：
# curl -L -o models/qwen2.5-1.5b-instruct-q4_k_m.gguf \
#   https://huggingface.co/Qwen/Qwen2.5-1.5B-Instruct-GGUF/resolve/main/qwen2.5-1.5b-instruct-q4_k_m.gguf
```

### 4.4 低内存要点

- Master：纯标准库 HTTP 服务，无 ORM/无模板引擎；JSON 手写编码（Go 自带 encoding/json 即可）。
- Worker：llama.cpp 只加载分到的层权重；模型用 Q4_K_M 量化（1.5B ≈ 0.9GB，3B ≈ 2GB）。
- 禁止在 Master 上加载模型；推理客户端按需启动 llama-cli，用完即退。

## 5. 异构调度与张量分配

### 5.1 切分策略对比（MVP 结论：张量并行）

| 策略 | 带宽需求 | 内存需求 | 适用场景 |
|---|---|---|---|
| 数据并行（每节点完整副本） | 低（只需同步梯度） | 极高（N 份模型） | 大集群高吞吐，本机不适用 |
| 张量并行（按层切分） | 中（传输激活值） | 低（每节点部分层） | 单机多设备/小集群，本方案首选 |
| 流水线并行（按阶段） | 低（仅阶段间传递） | 低 | 模型太大张量并行放不下时 |

- **MVP 采用张量并行**：llama.cpp `--rpc` 天然支持，按层轮流切分；`-ts` 可调比例。
- **比例计算**：Master /api/v1/plan 根据各节点 `mem_free` 自动生成 `-ts`（空闲内存越大，分到的层越多）。
- **带宽避坑**：Docker 两个 worker 走 WSL2 内部网络（近零延迟）；MuMu worker 走 adb forward（USB/本地回环），切分比例给小一点（如 0.2），避免 NAT 转发成为瓶颈。

### 5.2 调度建议算法（master/internal/scheduler/scheduler.go 已实现）

1. 拉取所有在线节点（30 秒内有心跳）；
2. 按 `mem_free` 降序排列；
3. 计算总可用内存，若小于 `model_size_gb * 1.2` 则返回"内存不足"；
4. 生成 `--rpc` 列表（addr 字段）与 `-ts` 比例（mem_free / 总 mem_free，最小 0.1）。

## 6. 硬核优化（差异化卖点）

### 6.1 内存监控（PowerShell）

`scripts/monitor/memory_watchdog.ps1`：
- 每 5 秒采样：Windows 总内存（WMI）、WSL2 内存（wsl free）、Docker 容器内存（docker stats）、MuMu 进程内存（Get-Process）。
- 超 88% 自动降级（docker stop pocket-worker-2 → 停 MuMu worker）；超 93% 强停并桌面通知（[System.Windows.Forms.MessageBox] / BurntToast 可选）。
- 日志写入 `%USERPROFILE%\.pocket-cluster\memory.log`，可配合计划任务开机自启。

### 6.2 Android 保活（MuMu 内节点不被杀）

保活四件套（详见 android/keepalive/）：
1. **adb 系统层豁免**（一次性）：
   ```bash
   adb shell settings put global cached_apps_freezer disabled
   adb shell dumpsys deviceidle disable
   adb shell cmd appops set com.termux RUN_IN_BACKGROUND allow
   adb shell am set-standby-bucket com.termux active
   ```
2. **前台化**：把 rpc-server 包装成前台服务（常驻通知），或至少用 `nohup + setsid` 脱离终端。
3. **双进程守护**：keepalive.sh 每 10 秒检查 `pidof llama-rpc-server`，被杀立即重启。
4. **宿主侧看门狗**：watchdog.py（PC 上跑）每 15 秒 `adb shell pidof` 检查，配合 `adb forward` 保证端口常驻。

> MuMu 特有：在 MuMu 12 设置中关闭"锁屏后清理后台"相关选项；多开场景下不要用"一键清理"。

### 6.3 一键启停

`scripts/demo/start_cluster.ps1`：先起 Docker compose → 检查 MuMu 与 adb forward → 启动 Master → 自动注册节点 → 打印健康状态。
`scripts/demo/stop_cluster.ps1`：优雅停止（先停推理客户端 → 停 MuMu worker → docker compose down → 杀 Master）。

## 7. 验收清单（MVP Done）

- [ ] `docker ps` 看到 2 个 worker 容器，`docker stats` 内存不超 1536m；
- [ ] MuMu 内 `pidof llama-rpc-server` 非空，宿主 `curl 127.0.0.1:50054`（TCP 连通）成功；
- [ ] Master `curl /api/v1/health` 返回 3 个 worker online；
- [ ] `llama-cli --rpc ...` 跑通一句推理，`-ts` 比例生效（日志出现 split）；
- [ ] 故意 `docker stop pocket-worker-2`，watchdog 日志出现降级动作，宿主内存未破红线。
