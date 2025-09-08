package telemetry

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" // Import pprof for profiling
	"runtime"
	"time"

	"github.com/rs/zerolog/log"
)

// ProfilingServer provides HTTP endpoints for Go profiling
type ProfilingServer struct {
	server *http.Server
	addr   string
}

// NewProfilingServer creates a new profiling server
func NewProfilingServer(addr string) *ProfilingServer {
	return &ProfilingServer{
		addr: addr,
	}
}

// Start starts the profiling server with pprof endpoints
func (ps *ProfilingServer) Start() error {
	mux := http.NewServeMux()

	// Add custom profiling endpoints
	mux.HandleFunc("/debug/stats", ps.statsHandler)
	mux.HandleFunc("/debug/gc", ps.gcHandler)
	mux.HandleFunc("/debug/build", ps.buildInfoHandler)

	// pprof endpoints are automatically registered at /debug/pprof/ when imported

	ps.server = &http.Server{
		Addr:    ps.addr,
		Handler: mux,
	}

	log.Info().Str("addr", ps.addr).Msg("Starting profiling server")
	return ps.server.ListenAndServe()
}

// Shutdown gracefully shuts down the profiling server
func (ps *ProfilingServer) Shutdown(ctx context.Context) error {
	if ps.server != nil {
		return ps.server.Shutdown(ctx)
	}
	return nil
}

// statsHandler provides runtime statistics
func (ps *ProfilingServer) statsHandler(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	_ = map[string]interface{}{
		"memory": map[string]interface{}{
			"alloc_mb":         bToMb(m.Alloc),
			"total_alloc_mb":   bToMb(m.TotalAlloc),
			"sys_mb":           bToMb(m.Sys),
			"heap_alloc_mb":    bToMb(m.HeapAlloc),
			"heap_sys_mb":      bToMb(m.HeapSys),
			"heap_idle_mb":     bToMb(m.HeapIdle),
			"heap_inuse_mb":    bToMb(m.HeapInuse),
			"heap_released_mb": bToMb(m.HeapReleased),
			"heap_objects":     m.HeapObjects,
			"stack_inuse_mb":   bToMb(m.StackInuse),
			"stack_sys_mb":     bToMb(m.StackSys),
		},
		"gc": map[string]interface{}{
			"num_gc":          m.NumGC,
			"num_forced_gc":   m.NumForcedGC,
			"gc_cpu_fraction": m.GCCPUFraction,
			"pause_total_ns":  m.PauseTotalNs,
			"pause_ns":        m.PauseNs[(m.NumGC+255)%256],
		},
		"goroutines": runtime.NumGoroutine(),
		"cpu_cores":  runtime.NumCPU(),
		"go_version": runtime.Version(),
		"go_os":      runtime.GOOS,
		"go_arch":    runtime.GOARCH,
		"timestamp":  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"memory": {
			"alloc_mb": %.2f,
			"total_alloc_mb": %.2f,
			"sys_mb": %.2f,
			"heap_alloc_mb": %.2f,
			"heap_sys_mb": %.2f,
			"heap_idle_mb": %.2f,
			"heap_inuse_mb": %.2f,
			"heap_released_mb": %.2f,
			"heap_objects": %d,
			"stack_inuse_mb": %.2f,
			"stack_sys_mb": %.2f
		},
		"gc": {
			"num_gc": %d,
			"num_forced_gc": %d,
			"gc_cpu_fraction": %.4f,
			"pause_total_ns": %d,
			"pause_ns": %d
		},
		"goroutines": %d,
		"cpu_cores": %d,
		"go_version": "%s",
		"go_os": "%s",
		"go_arch": "%s",
		"timestamp": "%s"
	}`,
		bToMb(m.Alloc), bToMb(m.TotalAlloc), bToMb(m.Sys),
		bToMb(m.HeapAlloc), bToMb(m.HeapSys), bToMb(m.HeapIdle),
		bToMb(m.HeapInuse), bToMb(m.HeapReleased), m.HeapObjects,
		bToMb(m.StackInuse), bToMb(m.StackSys),
		m.NumGC, m.NumForcedGC, m.GCCPUFraction,
		m.PauseTotalNs, m.PauseNs[(m.NumGC+255)%256],
		runtime.NumGoroutine(), runtime.NumCPU(),
		runtime.Version(), runtime.GOOS, runtime.GOARCH,
		time.Now().Format(time.RFC3339))
}

// gcHandler triggers garbage collection
func (ps *ProfilingServer) gcHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	before := time.Now()
	runtime.GC()
	duration := time.Since(before)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"message": "Garbage collection triggered",
		"duration_ms": %.2f,
		"timestamp": "%s"
	}`, duration.Seconds()*1000, time.Now().Format(time.RFC3339))
}

// buildInfoHandler provides build information
func (ps *ProfilingServer) buildInfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"go_version": "%s",
		"go_os": "%s",
		"go_arch": "%s",
		"compiler": "%s",
		"num_cpu": %d,
		"max_procs": %d
	}`,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
		runtime.Compiler,
		runtime.NumCPU(),
		runtime.GOMAXPROCS(0))
}

// bToMb converts bytes to megabytes
func bToMb(b uint64) float64 {
	return float64(b) / 1024 / 1024
}

// PerformanceProfiler provides performance profiling utilities
type PerformanceProfiler struct {
	enabled bool
	server  *ProfilingServer
}

// NewPerformanceProfiler creates a new performance profiler
func NewPerformanceProfiler(enabled bool, addr string) *PerformanceProfiler {
	var server *ProfilingServer
	if enabled {
		server = NewProfilingServer(addr)
	}

	return &PerformanceProfiler{
		enabled: enabled,
		server:  server,
	}
}

// Start starts the performance profiler
func (pp *PerformanceProfiler) Start() error {
	if !pp.enabled || pp.server == nil {
		return nil
	}

	go func() {
		if err := pp.server.Start(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("Performance profiler server failed")
		}
	}()

	return nil
}

// Shutdown stops the performance profiler
func (pp *PerformanceProfiler) Shutdown() error {
	if pp.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return pp.server.Shutdown(ctx)
	}
	return nil
}

// ProfileFunction profiles the execution of a function
func ProfileFunction(name string, fn func()) (time.Duration, error) {
	start := time.Now()

	// Record initial memory stats
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Execute function
	fn()

	// Record final memory stats
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	duration := time.Since(start)

	// Record profiling metrics
	labels := map[string]string{
		"function":  name,
		"component": "profiler",
	}

	GetGlobal().Timer("function_execution_duration", duration, labels)
	GetGlobal().Gauge("function_memory_delta_mb",
		float64(memAfter.Alloc-memBefore.Alloc)/1024/1024, labels)

	return duration, nil
}

// ProfiledFunction returns a wrapper that profiles function execution
func ProfiledFunction(name string, fn func()) func() {
	return func() {
		_, _ = ProfileFunction(name, fn)
	}
}

// MemorySnapshot captures a memory snapshot for analysis
type MemorySnapshot struct {
	Timestamp    time.Time `json:"timestamp"`
	AllocMB      float64   `json:"alloc_mb"`
	TotalAllocMB float64   `json:"total_alloc_mb"`
	SysMB        float64   `json:"sys_mb"`
	NumGC        uint32    `json:"num_gc"`
	Goroutines   int       `json:"goroutines"`
}

// TakeMemorySnapshot captures current memory state
func TakeMemorySnapshot() MemorySnapshot {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return MemorySnapshot{
		Timestamp:    time.Now(),
		AllocMB:      bToMb(m.Alloc),
		TotalAllocMB: bToMb(m.TotalAlloc),
		SysMB:        bToMb(m.Sys),
		NumGC:        m.NumGC,
		Goroutines:   runtime.NumGoroutine(),
	}
}
