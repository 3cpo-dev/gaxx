package telemetry

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"time"

	"github.com/rs/zerolog/log"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents a health check result
type HealthCheck struct {
	Name        string            `json:"name"`
	Status      HealthStatus      `json:"status"`
	Message     string            `json:"message"`
	LastChecked time.Time         `json:"last_checked"`
	Duration    time.Duration     `json:"duration"`
	Details     map[string]string `json:"details,omitempty"`
}

// MonitoringServer provides HTTP endpoints for monitoring and metrics
type MonitoringServer struct {
	collector          *Collector
	performanceMonitor *PerformanceMonitor
	healthChecks       map[string]func() HealthCheck
	server             *http.Server
}

// NewMonitoringServer creates a new monitoring server
func NewMonitoringServer(addr string, collector *Collector, perfMon *PerformanceMonitor) *MonitoringServer {
	ms := &MonitoringServer{
		collector:          collector,
		performanceMonitor: perfMon,
		healthChecks:       make(map[string]func() HealthCheck),
	}

	mux := http.NewServeMux()
	ms.setupRoutes(mux)

	ms.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return ms
}

// setupRoutes configures HTTP routes for monitoring
func (ms *MonitoringServer) setupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", ms.healthHandler)
	mux.HandleFunc("/metrics", ms.metricsHandler)
	mux.HandleFunc("/dashboard", ms.dashboardHandler)
	mux.HandleFunc("/api/metrics", ms.apiMetricsHandler)
	mux.HandleFunc("/api/health", ms.apiHealthHandler)
}

// healthHandler provides a simple health endpoint
func (ms *MonitoringServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	checks := ms.runHealthChecks()

	overallStatus := HealthStatusHealthy
	for _, check := range checks {
		if check.Status == HealthStatusUnhealthy {
			overallStatus = HealthStatusUnhealthy
			break
		} else if check.Status == HealthStatusDegraded {
			overallStatus = HealthStatusDegraded
		}
	}

	response := map[string]interface{}{
		"status":    overallStatus,
		"timestamp": time.Now(),
		"checks":    checks,
	}

	w.Header().Set("Content-Type", "application/json")
	if overallStatus != HealthStatusHealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}

// metricsHandler provides Prometheus-style metrics
func (ms *MonitoringServer) metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := ms.collector.GetMetrics()

	w.Header().Set("Content-Type", "text/plain")

	for _, metric := range metrics {
		labelStr := ""
		if len(metric.Labels) > 0 {
			var pairs []string
			for k, v := range metric.Labels {
				pairs = append(pairs, fmt.Sprintf(`%s="%s"`, k, v))
			}
			sort.Strings(pairs)
			labelStr = "{" + fmt.Sprintf("%v", pairs) + "}"
		}

		fmt.Fprintf(w, "# TYPE %s %s\n", metric.Name, metric.Type)
		fmt.Fprintf(w, "%s%s %f %d\n", metric.Name, labelStr, metric.Value, metric.Timestamp.Unix())
	}
}

//go:embed static
var staticFS embed.FS

// dashboardHandler serves the embedded dashboard HTML
func (ms *MonitoringServer) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	data, err := staticFS.ReadFile("static/dashboard.html")
	if err != nil {
		http.Error(w, "dashboard not found", http.StatusNotFound)
		return
	}
	w.Write(data)
}

// apiMetricsHandler provides JSON metrics API
func (ms *MonitoringServer) apiMetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := ms.collector.GetMetrics()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// apiHealthHandler provides JSON health API
func (ms *MonitoringServer) apiHealthHandler(w http.ResponseWriter, r *http.Request) {
	checks := ms.runHealthChecks()

	overallStatus := HealthStatusHealthy
	for _, check := range checks {
		if check.Status == HealthStatusUnhealthy {
			overallStatus = HealthStatusUnhealthy
			break
		} else if check.Status == HealthStatusDegraded {
			overallStatus = HealthStatusDegraded
		}
	}

	response := map[string]interface{}{
		"status":    overallStatus,
		"timestamp": time.Now(),
		"checks":    checks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegisterHealthCheck registers a health check function
func (ms *MonitoringServer) RegisterHealthCheck(name string, checkFn func() HealthCheck) {
	ms.healthChecks[name] = checkFn
}

// runHealthChecks executes all registered health checks
func (ms *MonitoringServer) runHealthChecks() []HealthCheck {
	var checks []HealthCheck

	for _, checkFn := range ms.healthChecks {
		start := time.Now()
		check := checkFn()
		check.Duration = time.Since(start)
		check.LastChecked = time.Now()
		checks = append(checks, check)
	}

	return checks
}

// Start starts the monitoring server
func (ms *MonitoringServer) Start() error {
	log.Info().Str("addr", ms.server.Addr).Msg("Starting monitoring server")
	return ms.server.ListenAndServe()
}

// Shutdown gracefully shuts down the monitoring server
func (ms *MonitoringServer) Shutdown() error {
	if ms.server != nil {
		return ms.server.Shutdown(nil)
	}
	return nil
}

// DefaultHealthChecks returns a set of default health checks
func DefaultHealthChecks() map[string]func() HealthCheck {
	return map[string]func() HealthCheck{
		"memory": func() HealthCheck {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			heapMB := float64(m.HeapAlloc) / (1024 * 1024)
			status := HealthStatusHealthy
			message := fmt.Sprintf("Heap memory: %.2f MB", heapMB)

			if heapMB > 1000 {
				status = HealthStatusDegraded
				message = fmt.Sprintf("High memory usage: %.2f MB", heapMB)
			}
			if heapMB > 2000 {
				status = HealthStatusUnhealthy
				message = fmt.Sprintf("Critical memory usage: %.2f MB", heapMB)
			}

			return HealthCheck{
				Name:    "memory",
				Status:  status,
				Message: message,
				Details: map[string]string{
					"heap_mb":    fmt.Sprintf("%.2f", heapMB),
					"goroutines": fmt.Sprintf("%d", runtime.NumGoroutine()),
				},
			}
		},
		"goroutines": func() HealthCheck {
			count := runtime.NumGoroutine()
			status := HealthStatusHealthy
			message := fmt.Sprintf("Goroutines: %d", count)

			if count > 1000 {
				status = HealthStatusDegraded
				message = fmt.Sprintf("High goroutine count: %d", count)
			}
			if count > 5000 {
				status = HealthStatusUnhealthy
				message = fmt.Sprintf("Critical goroutine count: %d", count)
			}

			return HealthCheck{
				Name:    "goroutines",
				Status:  status,
				Message: message,
				Details: map[string]string{
					"count": fmt.Sprintf("%d", count),
				},
			}
		},
	}
}
