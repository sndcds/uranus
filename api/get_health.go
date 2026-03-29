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

	fileCount, dirSize := grains_file.FileCountAndDirSize(app.UranusInstance.Config.PlutoImageDir)

	apiRequest.Success(http.StatusOK, gin.H{
		"status": "ok",

		// Go runtime
		"goroutines": goroutines,
		"threads":    threads,

		// System metrics
		"cpu": gin.H{
			"usage_percent": cpuPercent,
		},
		"memory": gin.H{
			"total":        vmStat.Total,
			"available":    vmStat.Available,
			"used":         vmStat.Used,
			"used_percent": vmStat.UsedPercent,
		},
		"host": gin.H{
			"hostname": hostInfo.Hostname,
			"uptime":   hostInfo.Uptime,
			"os":       hostInfo.OS,
			"platform": hostInfo.Platform,
		},
		"dirs": gin.H{
			"fileCount": fileCount,
			"dirSize":   grains_file.HumanSize(dirSize),
		},
		"temperature": temps,
	}, "")
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
