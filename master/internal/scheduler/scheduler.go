// Package scheduler 生成推理计划：选择节点、计算张量切分比例（-ts）。
package scheduler

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pocket-cluster/pocket-cluster/master/internal/registry"
)

// Plan 是一次推理会话的推荐配置。
type Plan struct {
	ModelSizeGB  float64   `json:"model_size_gb"`
	Enough       bool      `json:"enough"`
	RpcEndpoints []string  `json:"rpc_endpoints"`
	Ts           []float64 `json:"ts_ratios"`
	CliArgs      string    `json:"cli_args"`
	Reason       string    `json:"reason,omitempty"`
}

// BuildPlan 根据在线节点空闲内存生成 --rpc 与 -ts 推荐。
// modelSizeGB 为模型文件大小；推理还需约 20% 内存用于激活值/KV cache，
// 因此按 needMB = modelSizeGB * 1024 * 1.2 估算。
func BuildPlan(nodes []registry.Node, modelSizeGB float64) Plan {
	online := make([]registry.Node, 0, len(nodes))
	for _, n := range nodes {
		if n.Status == "online" {
			online = append(online, n)
		}
	}
	if len(online) == 0 {
		return Plan{ModelSizeGB: modelSizeGB, Enough: false, Reason: "no online nodes"}
	}

	// 空闲内存大的节点优先参与，并分配更多层。
	sort.SliceStable(online, func(i, j int) bool {
		return online[i].MemFreeMB > online[j].MemFreeMB
	})

	var totalFree int64
	for _, n := range online {
		totalFree += n.MemFreeMB
	}
	// 1.2 冗余系数：多出的 20% 用于 KV Cache 与激活值等推理运行时开销。
	needMB := int64(modelSizeGB * 1024 * 1.2)
	if totalFree < needMB {
		return Plan{
			ModelSizeGB: modelSizeGB,
			Enough:      false,
			Reason:      fmt.Sprintf("need ~%dMB free, have %dMB", needMB, totalFree),
		}
	}

	eps := make([]string, 0, len(online))
	ts := make([]float64, 0, len(online))
	for _, n := range online {
		eps = append(eps, "tcp://"+n.Addr)
		ratio := float64(n.MemFreeMB) / float64(totalFree)
		if ratio < 0.1 {
			ratio = 0.1 // 每个节点至少 10%，避免极端比例
		}
		ts = append(ts, ratio)
	}
	// 归一化到总和 1。
	var sum float64
	for _, v := range ts {
		sum += v
	}
	for i := range ts {
		ts[i] = round2(ts[i] / sum)
	}

	rpcArgs := make([]string, 0, len(eps))
	for _, e := range eps {
		rpcArgs = append(rpcArgs, "--rpc "+e)
	}
	tsArgs := fmt.Sprintf("-ts %s", strings.Join(fmtStrs(ts), ","))

	return Plan{
		ModelSizeGB:  modelSizeGB,
		Enough:       true,
		RpcEndpoints: eps,
		Ts:           ts,
		CliArgs:      strings.Join(rpcArgs, " ") + " " + tsArgs,
		Reason:       fmt.Sprintf("%d nodes, %dMB free", len(online), totalFree),
	}
}

func round2(f float64) float64 { return float64(int(f*100+0.5)) / 100 }

func fmtStrs(fs []float64) []string {
	out := make([]string, len(fs))
	for i, v := range fs {
		out[i] = fmt.Sprintf("%.2f", v)
	}
	return out
}
