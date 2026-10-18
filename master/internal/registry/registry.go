// Package registry 维护集群节点注册表与心跳。
package registry

import (
	"sort"
	"sync"
	"time"
)

// Node 描述一个推理 Worker 节点。
type Node struct {
	ID           string   `json:"id"`
	Addr         string   `json:"addr"` // llama.cpp RPC 端点，如 127.0.0.1:50052
	Arch         string   `json:"arch"` // linux-x86_64 / android-x86_64 / darwin-arm64 ...
	MemTotalMB   int64    `json:"mem_total"`
	MemFreeMB    int64    `json:"mem_free"`
	CPUs         int      `json:"cpus"`
	Capabilities []string `json:"capabilities"` // ["cpu"], 未来: ["gpu"], ["mps"]
	Status       string   `json:"status"`       // online / offline
	LastSeen     time.Time `json:"-"`
}

// Registry 是线程安全的节点注册表，TTL 内无心跳视为离线。
type Registry struct {
	mu    sync.RWMutex
	nodes map[string]*Node
	ttl   time.Duration
}

// New 创建注册表。ttl 为心跳过期时间。
func New(ttl time.Duration) *Registry {
	return &Registry{nodes: make(map[string]*Node), ttl: ttl}
}

// Register 注册或更新节点信息，并重置心跳时间。
func (r *Registry) Register(n Node) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n.Status = "online"
	n.LastSeen = time.Now()
	r.nodes[n.ID] = &n
}

// Heartbeat 更新节点心跳与空闲内存。节点不存在时返回 false。
func (r *Registry) Heartbeat(id string, memFreeMB int64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, ok := r.nodes[id]
	if !ok {
		return false
	}
	n.MemFreeMB = memFreeMB
	n.Status = "online"
	n.LastSeen = time.Now()
	return true
}

// Online 返回 TTL 内的在线节点列表，按 ID 排序保证调度器输入稳定。
func (r *Registry) Online() []Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	now := time.Now()
	out := make([]Node, 0, len(r.nodes))
	for _, n := range r.nodes {
		if now.Sub(n.LastSeen) < r.ttl {
			out = append(out, *n)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// All 返回全部节点（含离线，供展示）。
func (r *Registry) All() []Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Node, 0, len(r.nodes))
	for _, n := range r.nodes {
		out = append(out, *n)
	}
	return out
}
