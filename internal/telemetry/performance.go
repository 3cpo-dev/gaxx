package telemetry

import (
	"context"
	"runtime"
	"sync"
	"time"
)

// PerformanceMonitor tracks system and application performance metrics
type PerformanceMonitor struct {
	mu          sync.RWMutex
	enabled     bool
	collector   *Collector
	startTime   time.Time
	lastMetrics runtime.MemStats
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(collector *Collector, enabled bool) *PerformanceMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	pm := &PerformanceMonitor{
		enabled:   enabled,
		collector: collector,
		startTime: time.Now(),
		ctx:       ctx,
		cancel:    cancel,
	}

	if enabled {
		go pm.collectSystemMetrics()
	}

	return pm
}

// collectSystemMetrics periodically collects system performance metrics
func (pm *PerformanceMonitor) collectSystemMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.recordSystemMetrics()
		}
	}
}

// recordSystemMetrics records current system metrics
func (pm *PerformanceMonitor) recordSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	pm.mu.Lock()
	defer pm.mu.Unlock()

	labels := map[string]string{"component": "system"}

	// Memory metrics
	pm.collector.Gauge("gaxx_memory_heap_bytes", float64(m.HeapAlloc), labels)
	pm.collector.Gauge("gaxx_memory_heap_sys_bytes", float64(m.HeapSys), labels)
	pm.collector.Gauge("gaxx_memory_stack_bytes", float64(m.StackSys), labels)
	pm.collector.Gauge("gaxx_memory_gc_pause_ns", float64(m.PauseNs[(m.NumGC+255)%256]), labels)

	// GC metrics
	pm.collector.Counter("gaxx_gc_total", float64(m.NumGC-pm.lastMetrics.NumGC), labels)
	pm.collector.Gauge("gaxx_gc_cpu_fraction", m.GCCPUFraction*100, labels)

	// Goroutine metrics
	pm.collector.Gauge("gaxx_goroutines_total", float64(runtime.NumGoroutine()), labels)

	// CPU count
	pm.collector.Gauge("gaxx_cpu_cores", float64(runtime.NumCPU()), labels)

	// Uptime
	uptime := time.Since(pm.startTime)
	pm.collector.Gauge("gaxx_uptime_seconds", uptime.Seconds(), labels)

	pm.lastMetrics = m
}

// RecordTaskMetrics records metrics for task execution
func (pm *PerformanceMonitor) RecordTaskMetrics(taskName string, nodeCount int, duration time.Duration, successful, failed int) {
	if !pm.enabled {
		return
	}

	labels := map[string]string{
		"task":      taskName,
		"component": "task_execution",
	}

	pm.collector.Timer("gaxx_task_duration", duration, labels)
	pm.collector.Gauge("gaxx_task_nodes", float64(nodeCount), labels)
	pm.collector.Counter("gaxx_task_executions_successful", float64(successful), labels)
	pm.collector.Counter("gaxx_task_executions_failed", float64(failed), labels)

	// Calculate success rate
	total := successful + failed
	if total > 0 {
		successRate := float64(successful) / float64(total) * 100
		pm.collector.Gauge("gaxx_task_success_rate", successRate, labels)
	}
}

// RecordFleetMetrics records metrics for fleet operations
func (pm *PerformanceMonitor) RecordFleetMetrics(provider, operation string, nodeCount int, duration time.Duration, success bool) {
	if !pm.enabled {
		return
	}

	labels := map[string]string{
		"provider":  provider,
		"operation": operation,
		"component": "fleet",
	}

	pm.collector.Timer("gaxx_fleet_operation_duration", duration, labels)
	pm.collector.Gauge("gaxx_fleet_nodes", float64(nodeCount), labels)

	if success {
		pm.collector.Counter("gaxx_fleet_operations_successful", 1, labels)
	} else {
		pm.collector.Counter("gaxx_fleet_operations_failed", 1, labels)
	}
}

// RecordAgentMetrics records metrics for agent operations
func (pm *PerformanceMonitor) RecordAgentMetrics(nodeIP, operation string, duration time.Duration, success bool) {
	if !pm.enabled {
		return
	}

	labels := map[string]string{
		"node_ip":   nodeIP,
		"operation": operation,
		"component": "agent",
	}

	pm.collector.Timer("gaxx_agent_operation_duration", duration, labels)

	if success {
		pm.collector.Counter("gaxx_agent_operations_successful", 1, labels)
	} else {
		pm.collector.Counter("gaxx_agent_operations_failed", 1, labels)
	}
}

// RecordFileTransferMetrics records metrics for file transfer operations
func (pm *PerformanceMonitor) RecordFileTransferMetrics(nodeIP string, fileSize int64, duration time.Duration, success bool) {
	if !pm.enabled {
		return
	}

	labels := map[string]string{
		"node_ip":   nodeIP,
		"component": "file_transfer",
	}

	pm.collector.Timer("gaxx_file_transfer_duration", duration, labels)
	pm.collector.Histogram("gaxx_file_transfer_size_bytes", float64(fileSize), labels)

	if success {
		pm.collector.Counter("gaxx_file_transfers_successful", 1, labels)
		// Calculate throughput in MB/s
		if duration.Seconds() > 0 {
			throughputMBps := float64(fileSize) / (1024 * 1024) / duration.Seconds()
			pm.collector.Histogram("gaxx_file_transfer_throughput_mbps", throughputMBps, labels)
		}
	} else {
		pm.collector.Counter("gaxx_file_transfers_failed", 1, labels)
	}
}

// Shutdown stops the performance monitor
func (pm *PerformanceMonitor) Shutdown() {
	if pm.cancel != nil {
		pm.cancel()
	}
}

// TimerScope represents a scoped timer for measuring durations
type TimerScope struct {
	startTime time.Time
	name      string
	labels    map[string]string
	collector *Collector
}

// NewTimerScope creates a new timer scope
func NewTimerScope(name string, labels map[string]string) *TimerScope {
	return &TimerScope{
		startTime: time.Now(),
		name:      name,
		labels:    labels,
		collector: GetGlobal(),
	}
}

// End completes the timer and records the duration
func (ts *TimerScope) End() time.Duration {
	duration := time.Since(ts.startTime)
	ts.collector.Timer(ts.name, duration, ts.labels)
	return duration
}

// WithTimerScope executes a function and measures its duration
func WithTimerScope(name string, labels map[string]string, fn func()) time.Duration {
	timer := NewTimerScope(name, labels)
	fn()
	return timer.End()
}
