package scheduler_test

import (
	"testing"

	"github.com/pocket-cluster/pocket-cluster/master/internal/registry"
	"github.com/pocket-cluster/pocket-cluster/master/internal/scheduler"
)

// TestBuildPlanSortsByFreeMemoryDesc 覆盖按可用内存从大到小排序。
func TestBuildPlanSortsByFreeMemoryDesc(t *testing.T) {
	nodes := []registry.Node{
		{ID: "worker-1", Addr: "127.0.0.1:50052", MemFreeMB: 600, Status: "online"},
		{ID: "worker-2", Addr: "127.0.0.1:50053", MemFreeMB: 1400, Status: "online"},
		{ID: "worker-3", Addr: "127.0.0.1:50054", MemFreeMB: 900, Status: "online"},
	}
	plan := scheduler.BuildPlan(nodes, 1.5)
	if !plan.Enough {
		t.Fatalf("expected enough memory, got reason: %s", plan.Reason)
	}
	want := []string{
		"tcp://127.0.0.1:50053",
		"tcp://127.0.0.1:50054",
		"tcp://127.0.0.1:50052",
	}
	for i := range want {
		if plan.RpcEndpoints[i] != want[i] {
			t.Fatalf("endpoints not sorted by free memory desc: got %v, want %v",
				plan.RpcEndpoints, want)
		}
	}
}

// TestBuildPlanStableForEqualFreeMemory 覆盖相同空闲内存时保持输入顺序（稳定）。
func TestBuildPlanStableForEqualFreeMemory(t *testing.T) {
	nodes := []registry.Node{
		{ID: "worker-a", Addr: "127.0.0.1:50052", MemFreeMB: 1000, Status: "online"},
		{ID: "worker-b", Addr: "127.0.0.1:50053", MemFreeMB: 1000, Status: "online"},
	}
	plan := scheduler.BuildPlan(nodes, 1.0)
	if !plan.Enough {
		t.Fatalf("expected enough memory, got reason: %s", plan.Reason)
	}
	if plan.RpcEndpoints[0] != "tcp://127.0.0.1:50052" ||
		plan.RpcEndpoints[1] != "tcp://127.0.0.1:50053" {
		t.Fatalf("equal free memory should keep input order, got %v", plan.RpcEndpoints)
	}
}

// TestBuildPlanSkipsOfflineNodes 覆盖离线节点不参与调度。
func TestBuildPlanSkipsOfflineNodes(t *testing.T) {
	nodes := []registry.Node{
		{ID: "worker-1", Addr: "127.0.0.1:50052", MemFreeMB: 1400, Status: "online"},
		{ID: "worker-2", Addr: "127.0.0.1:50053", MemFreeMB: 99999, Status: "offline"},
	}
	plan := scheduler.BuildPlan(nodes, 1.0)
	if !plan.Enough {
		t.Fatalf("expected enough memory from online node, got reason: %s", plan.Reason)
	}
	if len(plan.RpcEndpoints) != 1 || plan.RpcEndpoints[0] != "tcp://127.0.0.1:50052" {
		t.Fatalf("offline node should be excluded, got %v", plan.RpcEndpoints)
	}
}

// TestBuildPlanInsufficientMemory 覆盖空闲内存不足时返回不可执行计划。
func TestBuildPlanInsufficientMemory(t *testing.T) {
	nodes := []registry.Node{
		{ID: "worker-1", Addr: "127.0.0.1:50052", MemFreeMB: 200, Status: "online"},
	}
	plan := scheduler.BuildPlan(nodes, 8.0)
	if plan.Enough {
		t.Fatal("expected plan to be insufficient")
	}
	if plan.Reason == "" {
		t.Fatal("expected non-empty reason for insufficient memory")
	}
	if len(plan.RpcEndpoints) != 0 || len(plan.Ts) != 0 {
		t.Fatalf("insufficient plan should have no endpoints/ratios, got %v / %v",
			plan.RpcEndpoints, plan.Ts)
	}
}

// TestBuildPlanTsRatiosTrackNodeCount 覆盖切分比例随节点数正确变化。
func TestBuildPlanTsRatiosTrackNodeCount(t *testing.T) {
	one := scheduler.BuildPlan([]registry.Node{
		{ID: "worker-1", Addr: "127.0.0.1:50052", MemFreeMB: 1400, Status: "online"},
	}, 0.5)
	if !one.Enough {
		t.Fatalf("expected enough memory, got reason: %s", one.Reason)
	}
	if len(one.Ts) != 1 || one.Ts[0] != 1.0 {
		t.Fatalf("single node should get ts=1, got %v", one.Ts)
	}

	two := scheduler.BuildPlan([]registry.Node{
		{ID: "worker-1", Addr: "127.0.0.1:50052", MemFreeMB: 700, Status: "online"},
		{ID: "worker-2", Addr: "127.0.0.1:50053", MemFreeMB: 700, Status: "online"},
	}, 0.5)
	if len(two.Ts) != 2 {
		t.Fatalf("two nodes should get two ratios, got %v", two.Ts)
	}
	sum := two.Ts[0] + two.Ts[1]
	if sum < 0.99 || sum > 1.01 {
		t.Fatalf("ratios should sum to ~1, got %v", sum)
	}
	if two.Ts[0] != two.Ts[1] {
		t.Fatalf("equal memory nodes should get equal ratios, got %v", two.Ts)
	}

	four := scheduler.BuildPlan([]registry.Node{
		{ID: "worker-1", Addr: "127.0.0.1:50052", MemFreeMB: 500, Status: "online"},
		{ID: "worker-2", Addr: "127.0.0.1:50053", MemFreeMB: 500, Status: "online"},
		{ID: "worker-3", Addr: "127.0.0.1:50054", MemFreeMB: 500, Status: "online"},
		{ID: "worker-4", Addr: "127.0.0.1:50055", MemFreeMB: 500, Status: "online"},
	}, 1.0)
	if len(four.Ts) != 4 {
		t.Fatalf("four nodes should get four ratios, got %v", four.Ts)
	}
	for _, v := range four.Ts {
		if v <= 0 {
			t.Fatalf("ratio must be positive, got %v", four.Ts)
		}
	}
}
