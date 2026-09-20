package api

import (
	"context"
	"net/http"
	"runtime/metrics"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/grains/grains_file"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"

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

	dirs := []string{
		app.UranusInstance.Config.PlutoImageDir,
		app.UranusInstance.Config.PlutoCacheDir,
		app.UranusInstance.Config.ProfileImageDir,
	}

	multiStats := grains_file.MultiDirStats(dirs)

	resp := model.HealthResponse{
		Status:     "ok",
		Goroutines: goroutines,
		Threads:    threads,
		CPU: model.CPUInfo{
			UsagePercent: cpuPercent,
		},
		Memory: model.MemoryInfo{
			Total:       vmStat.Total,
			Available:   vmStat.Available,
			Used:        vmStat.Used,
			UsedPercent: vmStat.UsedPercent,
		},
		Host: model.HostInfo{
			Hostname: hostInfo.Hostname,
			Uptime:   hostInfo.Uptime,
			OS:       hostInfo.OS,
			Platform: hostInfo.Platform,
		},
		Dirs:        multiStats,
		Temperature: temps,
	}

	apiRequest.Success(http.StatusOK, resp)
}

func (h *ApiHandler) GetServerInfo(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-server-info")

	ctx := gc.Request.Context()

	// Database check with its own short timeout.
	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	databaseStatus := "ok"

	var result int
	err := h.DbPool.QueryRow(dbCtx, "SELECT 1").Scan(&result)
	if err != nil {
		databaseStatus = "error"
	}

	// Server uptime.
	hostInfo, err := host.Info()
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	status := "ok"
	httpStatus := http.StatusOK

	if databaseStatus != "ok" {
		status = "error"
		httpStatus = http.StatusServiceUnavailable
	}

	resp := model.ServerInfoResponse{
		Status:   status,
		Database: databaseStatus,
		Uptime:   hostInfo.Uptime,
	}

	apiRequest.Success(httpStatus, resp)
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
