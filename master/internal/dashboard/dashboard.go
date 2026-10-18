// Package dashboard 提供 Master 的极简 Web UI 仪表盘。
// 仅使用 Go 标准库 html/template，无任何前端框架。
package dashboard

import (
	"html/template"
	"net/http"
	"sort"

	"github.com/pocket-cluster/pocket-cluster/master/internal/registry"
)

const version = "v0.1.0"

// row 是仪表盘表格的一行渲染数据。
type row struct {
	Online       bool
	ID           string
	Addr         string
	Arch         string
	MemFreeMB    int64
	MemTotalMB   int64
	MemUsedPct   int
	CPUs         int
	Capabilities string
}

// pageData 是仪表盘页面的全部渲染数据。
type pageData struct {
	TotalNodes  int
	OnlineNodes int
	Rows        []row
	Version     string
}

var page = template.Must(template.New("dashboard").Parse(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Pocket-Cluster Dashboard</title>
<style>
  body { margin:0; padding:32px; background:#0f1117; color:#e6e6e6; font-family:Consolas,'Courier New',monospace; }
  h1 { color:#4ade80; font-size:26px; margin:0 0 4px; }
  .sub { color:#64748b; font-size:13px; margin-bottom:20px; }
  .summary { display:flex; gap:16px; margin-bottom:20px; }
  .card { background:#1a1f2e; border:1px solid #2a3142; border-radius:8px; padding:14px 22px; }
  .card span { color:#94a3b8; font-size:13px; }
  .card b { display:block; font-size:28px; margin-top:4px; }
  table { width:100%; border-collapse:collapse; background:#1a1f2e; border:1px solid #2a3142; border-radius:8px; overflow:hidden; }
  th,td { padding:10px 12px; text-align:left; border-bottom:1px solid #242b3d; font-size:14px; }
  th { color:#94a3b8; font-size:13px; }
  tr:last-child td { border-bottom:none; }
  .dot { display:inline-block; width:10px; height:10px; border-radius:50%; margin-right:6px; vertical-align:middle; }
  .dot.on { background:#22c55e; box-shadow:0 0 8px #22c55e; }
  .dot.off { background:#ef4444; box-shadow:0 0 8px #ef4444; }
  .badge { padding:2px 10px; border-radius:999px; font-size:12px; }
  .badge.on { background:#14532d; color:#4ade80; border:1px solid #22c55e; }
  .badge.off { background:#7f1d1d; color:#fca5a5; border:1px solid #ef4444; }
  .mem-bar { width:120px; height:8px; background:#2a3142; border-radius:4px; overflow:hidden; display:inline-block; vertical-align:middle; }
  .mem-fill { height:100%; background:#3b82f6; }
  .mem-text { color:#94a3b8; font-size:12px; margin-left:8px; }
  footer { margin-top:24px; color:#64748b; font-size:12px; text-align:center; }
</style>
</head>
<body>
<h1>Pocket-Cluster Dashboard</h1>
<div class="sub">Master 极简仪表盘 · 仅标准库 html/template 渲染</div>
<div class="summary">
  <div class="card"><span>节点总数</span><b>{{.TotalNodes}}</b></div>
  <div class="card"><span>在线节点</span><b style="color:#4ade80;">{{.OnlineNodes}}</b></div>
</div>
<table>
  <tr><th>状态</th><th>ID</th><th>RPC 地址</th><th>架构</th><th>内存</th><th>CPU</th><th>能力</th></tr>
  {{range .Rows}}
  <tr>
    <td>{{if .Online}}<span class="dot on"></span><span class="badge on">在线</span>{{else}}<span class="dot off"></span><span class="badge off">离线</span>{{end}}</td>
    <td>{{.ID}}</td>
    <td>{{.Addr}}</td>
    <td>{{.Arch}}</td>
    <td><span class="mem-bar"><span class="mem-fill" style="width:{{.MemUsedPct}}%"></span></span><span class="mem-text">{{.MemFreeMB}}/{{.MemTotalMB}} MB</span></td>
    <td>{{.CPUs}}</td>
    <td>{{.Capabilities}}</td>
  </tr>
  {{else}}
  <tr><td colspan="7">暂无节点，等待注册...</td></tr>
  {{end}}
</table>
<footer>Pocket-Cluster {{.Version}} · 页面每 5 秒自动刷新</footer>
<script>
  setTimeout(function(){ location.reload(); }, 5000);
</script>
</body>
</html>
`))

// Handler 返回仪表盘 HTTP 处理器。
// 使用注册表中的全部节点（含离线）渲染，并按 ID 排序保持展示稳定。
func Handler(reg *registry.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/dashboard" {
			http.NotFound(w, r)
			return
		}
		all := reg.All()
		sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })

		online := 0
		rows := make([]row, 0, len(all))
		for _, n := range all {
			isOnline := n.Status == "online"
			if isOnline {
				online++
			}
			usedPct := 0
			if n.MemTotalMB > 0 && n.MemFreeMB > 0 {
				usedPct = int((n.MemTotalMB - n.MemFreeMB) * 100 / n.MemTotalMB)
				if usedPct < 0 {
					usedPct = 0
				}
				if usedPct > 100 {
					usedPct = 100
				}
			}
			caps := ""
			for i, c := range n.Capabilities {
				if i > 0 {
					caps += ", "
				}
				caps += c
			}
			rows = append(rows, row{
				Online:       isOnline,
				ID:           n.ID,
				Addr:         n.Addr,
				Arch:         n.Arch,
				MemFreeMB:    n.MemFreeMB,
				MemTotalMB:   n.MemTotalMB,
				MemUsedPct:   usedPct,
				CPUs:         n.CPUs,
				Capabilities: caps,
			})
		}

		data := pageData{
			TotalNodes:  len(all),
			OnlineNodes: online,
			Rows:        rows,
			Version:     version,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = page.Execute(w, data)
	}
}
