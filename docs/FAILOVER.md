---
AIGC:
    Label: "1"
    ContentProducer: 001191440300708461136T1XGW3
    ProduceID: 3144a37a9207ea4301a32a24b64ea984_1f296c5fca9e11f18533525400638852
    ReservedCode1: RXmcwXNlAr+mexlt2y76OQb3GR0/n6dR+kPADK8f/irrj9pTDpt/b87y1+tgNpj5IFwBGbD7oIze951XfaALKpwf5jihvkuqkzaUOmmC1jA+gWCIHo9PZP2mqVgpHKEqLN+OYkhLvJ1Yz9WNoUCCcLMx2ZfBZI1K+ywnNs9+v2hbt6iA8eqLrYRB+Ms=
    ContentPropagator: 001191440300708461136T1XGW3
    PropagateID: 3144a37a9207ea4301a32a24b64ea984_1f296c5fca9e11f18533525400638852
    ReservedCode2: RXmcwXNlAr+mexlt2y76OQb3GR0/n6dR+kPADK8f/irrj9pTDpt/b87y1+tgNpj5IFwBGbD7oIze951XfaALKpwf5jihvkuqkzaUOmmC1jA+gWCIHo9PZP2mqVgpHKEqLN+OYkhLvJ1Yz9WNoUCCcLMx2ZfBZI1K+ywnNs9+v2hbt6iA8eqLrYRB+Ms=
---

# 故障转移与任务重试（Failover & Retry）

Master（pocketd）在调度层只依赖**在线节点**，任何节点离线（心跳超时或被主动标记失败）都会自动从推理计划中排除。本文档说明三层故障处理机制与客户端重试策略。

## 1. 心跳 TTL 自动剔除（被动发现）

- Worker 每 `HEARTBEAT_SEC`（默认 10 秒）上报一次心跳，Master 用 `LastSeen` 记录最近一次心跳时间。
- `registry.Online()` 只返回 `now - LastSeen < TTL`（Master 默认 `--node-ttl 30 秒`）的节点；超过 TTL 未心跳的节点自动从在线列表消失，调度器不会再把它写入 `--rpc` 参数。
- 节点只是被过滤，`registry.All()` 仍保留它供 Dashboard 展示（状态标记为离线前的快照）。
- 示例：一个 Worker 断网 40 秒，第 41 秒请求 `GET /api/v1/plan` 时该节点已不在 `rpc_endpoints` 中。

## 2. MarkFailed 主动标记（主动发现）

心跳超时依赖"节点彻底断联"才能被发现，若 RPC 端口可达但推理失败（连接被拒、张量传输中断、OOM），应由 **Master 侧**在检测到错误后主动调用：

```go
registry.MarkFailed(nodeID, "rpc connect timeout") // 节点立即 offline，并记录失败原因
registry.ClearFailed(nodeID)                        // 节点恢复后重新标记 online
```

- `MarkFailed(id, reason) bool`：节点不存在返回 `false`；存在则置 `Status="offline"`、写入 `FailedReason`、清零心跳时间，使 `Online()` 即刻剔除。
- `ClearFailed(id) bool`：节点不存在返回 `false`；存在则恢复 `Status="online"`、清空 `FailedReason`、重置心跳时间。
- 设计取舍：**被动 TTL + 主动 MarkFailed 双通道**。TTL 兜底断联场景，MarkFailed 覆盖"连接仍在但已不可用"场景，两者都不需要改动 `Online()/All()` 的既有逻辑。

## 3. 客户端重试策略

llama.cpp 客户端拿到 `BuildPlan` 的 `--rpc` 端点列表后开始张量切分推理，若中间某个 RPC 端点失败（连接拒绝/超时/中断），请按以下流程重试：

1. 捕获失败：记录失败节点地址（`tcp://host:port`）与错误原因。
2. 上报 Master：通过 `POST /api/v1/nodes/register` 携带该节点信息，或调用 Master 侧标记接口将节点置为失败（实现中对应 `registry.MarkFailed`）。
3. 重新请求计划：再次 `POST /api/v1/plan`，此时失败节点已被排除，返回的新 `rpc_endpoints` 只含健康节点。
4. 用新计划重试推理：llama.cpp 重新按新 `-ts` 切分；若失败节点此前已切走部分层，本次会在剩余节点间重新分配。
5. 幂等与退避：重试上限建议 3 次，间隔 1s / 2s / 4s 指数退避；全部失败后返回明确错误，由上层决定降级（如退回单机模式）。

## 4. 示例 curl 流程

假设集群：`worker-1`（127.0.0.1:50052）、`worker-2`（127.0.0.1:50053），Master 在 `127.0.0.1:8080`。

```bash
# 1) 查看当前计划（两个节点都在）
curl -s -X POST http://127.0.0.1:8080/api/v1/plan \
  -H 'Content-Type: application/json' -d '{"model_size_gb":1.5}'

# 2) 推理时 worker-2 的 RPC 连接失败，Master 将其标记为失败
curl -s -X POST http://127.0.0.1:8080/api/v1/nodes/fail \
  -H 'Content-Type: application/json' -d '{"id":"worker-2","reason":"rpc connect timeout"}'

# 3) 重新请求计划：rpc_endpoints 只剩 worker-1，ts=[1.00]
curl -s -X POST http://127.0.0.1:8080/api/v1/plan \
  -H 'Content-Type: application/json' -d '{"model_size_gb":1.5}'

# 4) worker-2 修复后恢复在线
curl -s -X POST http://127.0.0.1:8080/api/v1/nodes/recover \
  -H 'Content-Type: application/json' -d '{"id":"worker-2"}'
```

> 说明：`POST /api/v1/nodes/fail` 与 `POST /api/v1/nodes/recover` 已由 pocketd 原生暴露，分别对应 `registry.MarkFailed` 与 `registry.ClearFailed`；节点恢复后也可直接通过心跳（`POST /api/v1/nodes/heartbeat`）重新回到在线池。
*（内容由AI生成，仅供参考）*
