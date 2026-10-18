package config

import "flag"

// Config 是 pocketd 的启动配置，全部来自命令行 flag。
type Config struct {
	Listen    string // HTTP 监听地址
	NodeTTL   int    // 节点心跳过期秒数
	MaxMemPct int    // 宿主机内存红线百分比（供 health 接口返回参考）
}

// Parse 解析命令行参数并返回配置。
func Parse() Config {
	c := Config{}
	flag.StringVar(&c.Listen, "listen", ":8080", "HTTP listen address")
	flag.IntVar(&c.NodeTTL, "node-ttl", 30, "node heartbeat TTL in seconds")
	flag.IntVar(&c.MaxMemPct, "max-mem-pct", 90, "host memory red line percent")
	flag.Parse()
	return c
}
