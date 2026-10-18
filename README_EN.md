# Pocket-Cluster

> Stitch old devices and e-waste together to run local large models.
> A cross-platform (Windows / macOS / Linux / Android / iOS) heterogeneous distributed LLM inference cluster; the MVP focuses on a simulated single-machine cluster on Windows.

<blockquote style="background:#10151f; color:#e6e6e6; border-left:4px solid #22c55e; padding:12px 18px; border-radius:6px; margin:8px 0 24px;">
<b>Why Pocket-Cluster?</b> Built for old <b>16GB devices</b>; native hybrid scheduling across <b>Windows, WSL2, Docker and MuMu emulator</b>. No GPU required, run distributed inference on idle compute.
</blockquote>

## Why

- exo / cake and similar projects have poor Windows support and tedious setup; nobody does the Windows + Android emulator + WSL2 combination properly.
- Goal: let an old 16GB Windows PC spin up a "1 Master + N Worker" heterogeneous cluster via Docker + WSL2 + MuMu, and run llama.cpp inference.
- Vision: laptops, old phones, TV boxes, Raspberry Pis... any networked device can become your inference node.

## Core Design (understand in 30 seconds)

```
                 +------------------------------+
                 |   Master (pocketd, Go)        |
                 |   * Node register / heartbeat |
                 |   * Build llama.cpp --rpc args |
                 |   * Memory red line 90% auto  |
                 +---------------+--------------+
                                 | HTTP/JSON (control plane)
        +------------------------+------------------------+
        |                        |                        |
+-------v--------+    +---------v--------+    +----------v----------+
| Worker 1 (Linux)|    | Worker 2 (Linux)|    | Worker 3 (Android)   |
| Docker container|    | Docker container|    | Inside MuMu emulator |
| llama-rpc-server|    | llama-rpc-server|    | llama-rpc-server      |
| :50052          |    | :50052          |    | :50052 (adb forward)  |
+-------^--------+    +---------^--------+    +----------^----------+
        +------------------------+------------------------+
                          llama.cpp RPC (data plane, TCP)
```

- **Control plane**: Master (Go, zero third-party dependencies) manages node registration, heartbeats, health checks and scheduling suggestions over HTTP+JSON.
- **Data plane**: use llama.cpp's native RPC mechanism (`llama-rpc-server` + `--rpc`) directly - no custom tensor-split protocol needed. llama.cpp handles layer-wise tensor parallelism; Master only tells the client "which nodes to connect and in what ratio."
- **Minimalism**: Master compiles to ~8MB and uses ~20MB RAM; Workers reuse the official llama.cpp rpc-server binary with zero extra framework.

## Quick Start (Windows 16GB RAM, 30 minutes)

> **Option 0: 1-minute quick demo (no llama.cpp compilation needed)**
> 1. Start the Master: `cd master && go build -o pocketd.exe ./cmd/pocketd && .\pocketd.exe`
> 2. In another terminal, simulate three fake nodes: `python examples/mock_worker.py`
> 3. Open http://127.0.0.1:8080/ in your browser - three green online nodes light up on the Dashboard
> 4. Simulate an inference client: `python examples/mock_client.py` - prints tensor-split parameters and the node dispatch process
> Only requires Python 3.8+, never touches llama.cpp - great for quick demos and CI smoke tests.

> Prerequisites: make sure `adb` and `python` are on your system PATH.

1. Install WSL2 + Docker Desktop (see docs/MVP_GUIDE.md).
2. Copy `wsl/.wslconfig` to `%USERPROFILE%\.wslconfig`, then restart WSL with `wsl --shutdown`.
3. `cd docker && docker compose up -d --build` to start 2 Linux Workers.
   - Note: the `master` service in `docker-compose.yml` belongs to `profiles: ["optional"]` and is not started by default; Master is recommended to run natively on Windows (a containerized version is provided via `docker/master/Dockerfile`).
4. Follow `android/setup_mumu.ps1` to configure the MuMu emulator and deploy the Android Worker.
5. `go build -o pocketd.exe ./cmd/pocketd && ./pocketd.exe` to start the Master.
   - After startup, visit http://127.0.0.1:8080/ to view the node dashboard.
6. Run `python examples/client_test.py` to get recommended RPC parameters, or use `examples/chat_demo.sh` to run llama-cli directly.
   - Note: `scripts/demo/start_cluster.ps1` registers nodes with demo initial memory values; production workers should report real memory periodically via `POST /api/v1/nodes/heartbeat`.

## Directory Structure

```text
master/      Master scheduler (Go, standard library only)
docker/      Containerized Worker / Master deployment (llama.cpp rpc-server)
wsl/         WSL2 memory limit configuration
android/     MuMu/Android Worker deployment & keep-alive
scripts/     PowerShell/Python monitoring, health checks, one-click start/stop
examples/    Client call examples
docs/        Architecture, MVP guide, memory budget, troubleshooting
```

## Tech Stack

| Component | Choice | Why |
|---|---|---|
| Master | Go 1.21+, stdlib only | 8MB binary, ~20MB RAM, zero-dependency deployment |
| Worker | llama.cpp rpc-server | Official RPC tensor parallelism; controlled CPU inference memory |
| Control plane | HTTP + JSON | Cross-platform, debuggable, no codegen overhead |
| Data plane | llama.cpp RPC (TCP :50052) | Official implementation, layer-wise split, LAN-bandwidth friendly |
| Monitoring | PowerShell + Python | Native host capabilities, zero installation |

## Memory Red Line (non-negotiable)

- Host memory usage must never exceed 90% (~14.4GB on a 16GB machine).
- Three layers of limits: `.wslconfig memory=5GB` -> container `mem_limit=1536m` -> monitoring scripts auto-degrade at 88%/93%.
- See docs/MEMORY_BUDGET.md for details.

## Roadmap

- [x] MVP: single-machine simulated cluster (Windows + Docker + MuMu, 1 Master + 2/3 Workers)
- [x] Automatic tensor-split ratio calculation (based on per-node free memory / CPU cores)
- [ ] Real Android / Raspberry Pi node onboarding
- [ ] Model shard caching & incremental distribution
- [ ] Automatic removal of failed nodes & task retry
- [ ] iOS / macOS nodes (Metal backend)

## Author

Chen Wenjie  -  GitHub: [chenwenjie0119](https://github.com/chenwenjie0119)

## License

MIT License. See LICENSE.
