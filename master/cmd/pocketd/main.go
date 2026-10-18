// pocketd 是 Pocket-Cluster 的 Master 调度端。
// 仅使用 Go 标准库，零第三方依赖，编译后约 8MB、运行内存约 20MB。
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/pocket-cluster/pocket-cluster/master/internal/config"
	"github.com/pocket-cluster/pocket-cluster/master/internal/dashboard"
	"github.com/pocket-cluster/pocket-cluster/master/internal/registry"
	"github.com/pocket-cluster/pocket-cluster/master/internal/scheduler"
)

func main() {
	cfg := config.Parse()
	reg := registry.New(time.Duration(cfg.NodeTTL) * time.Second)

	mux := http.NewServeMux()

	// GET / 与 /dashboard 极简 Web UI 仪表盘
	mux.HandleFunc("/", dashboard.Handler(reg))
	mux.HandleFunc("/dashboard", dashboard.Handler(reg))

	// GET /api/v1/health 集群健康摘要
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"status":      "ok",
			"nodes":       len(reg.Online()),
			"max_mem_pct": cfg.MaxMemPct,
		})
	})

	// POST /api/v1/nodes/register 节点注册
	mux.HandleFunc("/api/v1/nodes/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var n registry.Node
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
			http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
			return
		}
		reg.Register(n)
		writeJSON(w, map[string]any{"ok": true, "id": n.ID})
	})

	// POST /api/v1/nodes/heartbeat 节点心跳
	mux.HandleFunc("/api/v1/nodes/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var hb struct {
			ID      string `json:"id"`
			MemFree int64  `json:"mem_free"`
		}
		if err := json.NewDecoder(r.Body).Decode(&hb); err != nil {
			http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
			return
		}
		if !reg.Heartbeat(hb.ID, hb.MemFree) {
			http.Error(w, "unknown node: "+hb.ID, http.StatusNotFound)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})

	// GET /api/v1/nodes 全部节点
	mux.HandleFunc("/api/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"nodes": reg.All()})
	})

	// POST /api/v1/nodes/fail 标记节点失败（置离线并记录原因）
	mux.HandleFunc("/api/v1/nodes/fail", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ID     string `json:"id"`
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
			return
		}
		if !reg.MarkFailed(req.ID, req.Reason) {
			http.Error(w, "unknown node: "+req.ID, http.StatusNotFound)
			return
		}
		writeJSON(w, map[string]any{"ok": true, "id": req.ID})
	})

	// POST /api/v1/nodes/recover 清除失败标记并恢复在线
	mux.HandleFunc("/api/v1/nodes/recover", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
			return
		}
		if !reg.ClearFailed(req.ID) {
			http.Error(w, "unknown node: "+req.ID, http.StatusNotFound)
			return
		}
		writeJSON(w, map[string]any{"ok": true, "id": req.ID})
	})

	// POST /api/v1/plan 生成推理计划
	mux.HandleFunc("/api/v1/plan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ModelSizeGB float64 `json:"model_size_gb"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
			return
		}
		plan := scheduler.BuildPlan(reg.Online(), req.ModelSizeGB)
		writeJSON(w, plan)
	})

	log.Printf("pocketd listening on %s", cfg.Listen)
	log.Fatal(http.ListenAndServe(cfg.Listen, mux))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
