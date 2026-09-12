package model

import "github.com/sndcds/grains/grains_file"

type HealthResponse struct {
	Status      string                     `json:"status"`
	Goroutines  map[string]uint64          `json:"goroutines"`
	Threads     map[string]uint64          `json:"threads"`
	CPU         CPUInfo                    `json:"cpu"`
	Memory      MemoryInfo                 `json:"memory"`
	Host        HostInfo                   `json:"host"`
	Dirs        []grains_file.MultiDirInfo `json:"dirs"`
	Temperature interface{}                `json:"temperature"`
}

type CPUInfo struct {
	UsagePercent []float64 `json:"usage_percent"`
}

type MemoryInfo struct {
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
}

type HostInfo struct {
	Hostname string `json:"hostname"`
	Uptime   uint64 `json:"uptime"`
	OS       string `json:"os"`
	Platform string `json:"platform"`
}

type ServerInfoResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Uptime   uint64 `json:"uptime"`
}
