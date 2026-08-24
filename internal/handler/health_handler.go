package handler

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"gozaq/config"
	"gozaq/pkg/cache"
	"gozaq/pkg/response"
)

var startTime = time.Now()

type HealthHandler struct {
	cfg   *config.Config
	db    *pgxpool.Pool
	cache cache.Cache
}

func NewHealthHandler(cfg *config.Config, db *pgxpool.Pool, cache cache.Cache) *HealthHandler {
	return &HealthHandler{
		cfg:   cfg,
		db:    db,
		cache: cache,
	}
}

type HealthResponse struct {
	Status       string             `json:"status"`
	Environment  string             `json:"environment"`
	Build        config.VersionInfo `json:"build"`
	Uptime       string             `json:"uptime"`
	UptimeSec    int64              `json:"uptime_seconds"`
	System       SystemMetrics      `json:"system"`
	Dependencies DependencyHealth   `json:"dependencies"`
}

type SystemMetrics struct {
	GoVersion   string `json:"go_version"`
	Goroutines  int    `json:"goroutines"`
	MemoryAlloc string `json:"memory_alloc"`
	MemorySys   string `json:"memory_sys"`
	GCCycles    uint32 `json:"gc_cycles"`
	CPUCores    int    `json:"cpu_cores"`
}

type DependencyHealth struct {
	Database DatabaseHealth `json:"database"`
	Cache    CacheHealth    `json:"cache"`
}

type DatabaseHealth struct {
	Status        string  `json:"status"`
	Type          string  `json:"type"`
	LatencyMs     float64 `json:"latency_ms"`
	TotalConns    int32   `json:"total_conns"`
	AcquiredConns int32   `json:"acquired_conns"`
	IdleConns     int32   `json:"idle_conns"`
	MaxConns      int32   `json:"max_conns"`
}

type CacheHealth struct {
	Status    string  `json:"status"`
	Type      string  `json:"type"`
	LatencyMs float64 `json:"latency_ms"`
}

func (h *HealthHandler) Check(w http.ResponseWriter, _ *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	overallStatus := "UP"

	// 1. Check Database Health (PostgreSQL)
	dbHealth := DatabaseHealth{
		Type:     "PostgreSQL",
		MaxConns: int32(h.cfg.Database.MaxOpenConns),
	}

	dbStart := time.Now()
	if err := h.db.Ping(ctx); err != nil {
		dbHealth.Status = "DOWN"
		overallStatus = "DEGRADED"
	} else {
		dbHealth.Status = "UP"
		dbHealth.LatencyMs = float64(time.Since(dbStart).Microseconds()) / 1000.0

		stats := h.db.Stat()
		dbHealth.TotalConns = stats.TotalConns()
		dbHealth.AcquiredConns = stats.AcquiredConns()
		dbHealth.IdleConns = stats.IdleConns()
	}

	// 2. Check Cache Health (Redis)
	cacheHealth := CacheHealth{
		Type: "Redis",
	}

	cacheStart := time.Now()
	// Test cache read/write ping
	if err := h.cache.Set(ctx, "health:ping", "pong", 5*time.Second); err != nil {
		cacheHealth.Status = "DOWN (using Noop fallback)"
		if overallStatus == "UP" {
			overallStatus = "DEGRADED"
		}
	} else {
		cacheHealth.Status = "UP"
		cacheHealth.LatencyMs = float64(time.Since(cacheStart).Microseconds()) / 1000.0
	}

	// 3. Collect System & Memory Stats
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	uptimeDuration := time.Since(startTime)

	data := HealthResponse{
		Status:      overallStatus,
		Environment: h.cfg.App.Env,
		Build:       config.GetVersionInfo(),
		Uptime:      uptimeDuration.Round(time.Second).String(),
		UptimeSec:   int64(uptimeDuration.Seconds()),
		System: SystemMetrics{
			GoVersion:   runtime.Version(),
			Goroutines:  runtime.NumGoroutine(),
			MemoryAlloc: formatBytes(mem.Alloc),
			MemorySys:   formatBytes(mem.Sys),
			GCCycles:    mem.NumGC,
			CPUCores:    runtime.NumCPU(),
		},
		Dependencies: DependencyHealth{
			Database: dbHealth,
			Cache:    cacheHealth,
		},
	}

	statusCode := http.StatusOK
	if overallStatus == "DOWN" {
		statusCode = http.StatusServiceUnavailable
	}

	response.Success(w, statusCode, "System health report generated successfully", data)
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
