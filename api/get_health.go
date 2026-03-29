package api

import (
	"net/http"
	"runtime/metrics"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/grains/grains_file"
	"github.com/sndcds/uranus/app"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

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

func (h *ApiHandler) GetHealth(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-health")

	// Go runtime metrics
	goroutines := map[string]uint64{
		"created":  readMetric("/sched/goroutines-created:goroutines"),
		"live":     readMetric("/sched/goroutines:goroutines"),
		"syscall":  readMetric("/sched/goroutines/not-in-go:goroutines"),
		"runnable": readMetric("/sched/goroutines/runnable:goroutines"),
		"running":  readMetric("/sched/goroutines/running:goroutines"),
		"waiting":  readMetric("/sched/goroutines/waiting:goroutines"),
	}

	threads := map[string]uint64{
		"max":  readMetric("/sched/gomaxprocs:threads"),
		"live": readMetric("/sched/threads/total:threads"),
	}

	// CPU usage
	cpuPercent, _ := cpu.Percent(0, false)

	// Memory usage
	vmStat, _ := mem.VirtualMemory()

	// Host info (uptime etc.)
	hostInfo, _ := host.Info()

	// Temperature (Linux only mostly)
	temps, _ := host.SensorsTemperatures()

	dirs := []string{
		app.UranusInstance.Config.PlutoImageDir,
		app.UranusInstance.Config.PlutoCacheDir,
		app.UranusInstance.Config.ProfileImageDir,
	}

	multiStats := grains_file.MultiDirStats(dirs)

	resp := HealthResponse{
		Status:     "ok",
		Goroutines: goroutines,
		Threads:    threads,
		CPU: CPUInfo{
			UsagePercent: cpuPercent,
		},
		Memory: MemoryInfo{
			Total:       vmStat.Total,
			Available:   vmStat.Available,
			Used:        vmStat.Used,
			UsedPercent: vmStat.UsedPercent,
		},
		Host: HostInfo{
			Hostname: hostInfo.Hostname,
			Uptime:   hostInfo.Uptime,
			OS:       hostInfo.OS,
			Platform: hostInfo.Platform,
		},
		Dirs:        multiStats,
		Temperature: temps,
	}

	apiRequest.Success(http.StatusOK, resp, "")
}

// Helper function to read a metric safely
func readMetric(name string) uint64 {
	sample := []metrics.Sample{{Name: name}}
	metrics.Read(sample)

	if sample[0].Value.Kind() == metrics.KindUint64 {
		return sample[0].Value.Uint64()
	}

	return 0
}
