package registry_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pocket-cluster/pocket-cluster/master/internal/registry"
)

// TestRegisterAndHeartbeat 覆盖节点注册与心跳更新。
func TestRegisterAndHeartbeat(t *testing.T) {
	r := registry.New(30 * time.Second)
	r.Register(registry.Node{ID: "worker-1", Addr: "127.0.0.1:50052", MemFreeMB: 1400})

	nodes := r.Online()
	if len(nodes) != 1 || nodes[0].ID != "worker-1" {
		t.Fatalf("expected 1 online node worker-1, got %+v", nodes)
	}
	if nodes[0].Status != "online" {
		t.Fatalf("expected status online, got %q", nodes[0].Status)
	}

	if !r.Heartbeat("worker-1", 900) {
		t.Fatal("heartbeat for existing node should return true")
	}
	if r.Heartbeat("missing", 100) {
		t.Fatal("heartbeat for unknown node should return false")
	}

	nodes = r.Online()
	if len(nodes) != 1 || nodes[0].MemFreeMB != 900 {
		t.Fatalf("expected mem_free updated to 900, got %+v", nodes)
	}
}

// TestHeartbeatTTLEvictsStaleNode 覆盖 TTL 内无心跳的节点被剔除。
func TestHeartbeatTTLEvictsStaleNode(t *testing.T) {
	r := registry.New(100 * time.Millisecond)
	r.Register(registry.Node{ID: "worker-1", Addr: "127.0.0.1:50052", MemFreeMB: 1400})

	if len(r.Online()) != 1 {
		t.Fatal("node should be online right after register")
	}
	time.Sleep(150 * time.Millisecond)
	if len(r.Online()) != 0 {
		t.Fatal("node should be evicted from Online() after TTL without heartbeat")
	}
	// All() 仍保留该节点（离线展示用途）。
	if all := r.All(); len(all) != 1 {
		t.Fatalf("All() should still list the node for display, got %+v", all)
	}
}

// TestOnlineAndAllReturnCopies 验证返回值是节点副本，而非内部切片引用。
func TestOnlineAndAllReturnCopies(t *testing.T) {
	r := registry.New(30 * time.Second)
	r.Register(registry.Node{ID: "worker-1", Addr: "127.0.0.1:50052", MemFreeMB: 1400})

	got := r.Online()
	got[0].MemFreeMB = 1
	got[0].Addr = "mutated"

	again := r.Online()
	if again[0].MemFreeMB != 1400 || again[0].Addr != "127.0.0.1:50052" {
		t.Fatalf("Online() should return copies, internal node mutated: %+v", again)
	}

	all := r.All()
	all[0].MemFreeMB = 1
	all[0].Addr = "mutated"
	again = r.Online()
	if again[0].MemFreeMB != 1400 || again[0].Addr != "127.0.0.1:50052" {
		t.Fatalf("All() should return copies, internal node mutated: %+v", again)
	}
}

// TestConcurrentRegisterHeartbeatAndList 并发读写，配合 -race 验证无 data race。
func TestConcurrentRegisterHeartbeatAndList(t *testing.T) {
	r := registry.New(30 * time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("worker-%d", i)
			r.Register(registry.Node{
				ID:     id,
				Addr:   fmt.Sprintf("127.0.0.1:%d", 50000+i),
				MemFreeMB: int64(100 + i*100),
			})
			for j := 0; j < 50; j++ {
				r.Heartbeat(id, int64(100+j))
				_ = r.Online()
				_ = r.All()
			}
		}(i)
	}
	wg.Wait()
	if got := len(r.Online()); got != 8 {
		t.Fatalf("expected 8 online nodes, got %d", got)
	}
}
