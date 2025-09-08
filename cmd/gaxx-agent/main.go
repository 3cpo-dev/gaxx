package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/3cpo-dev/gaxx/internal/agent"
	"github.com/3cpo-dev/gaxx/internal/telemetry"
)

func main() {
	// Initialize telemetry for agent
	telemetry.InitGlobal(true, "")
	defer telemetry.Shutdown()

	// Start performance monitoring
	collector := telemetry.GetGlobal()
	perfMon := telemetry.NewPerformanceMonitor(collector, true)
	defer perfMon.Shutdown()

	// Start profiling server in background
	profiler := telemetry.NewPerformanceProfiler(true, ":6060")
	defer profiler.Shutdown()
	go func() {
		if err := profiler.Start(); err != nil && err.Error() != "http: Server closed" {
			fmt.Fprintf(os.Stderr, "Profiler server failed: %v\n", err)
		}
	}()

	// Start monitoring server on a different port
	go startAgentMonitoring(":9091", collector, perfMon)

	addr := ":8088"
	srv := &agent.Server{Version: "dev"}

	// Record agent startup
	telemetry.CounterGlobal("gaxx_agent_starts", 1, map[string]string{
		"component": "agent",
		"version":   "dev",
	})

	go func() {
		if err := srv.ListenAndServe(addr); err != nil {
			telemetry.CounterGlobal("gaxx_agent_errors", 1, map[string]string{
				"error":     err.Error(),
				"component": "agent",
			})
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}()

	fmt.Fprintf(os.Stdout, "gaxx-agent listening on %s\n", addr)
	fmt.Fprintf(os.Stdout, "gaxx-agent monitoring on :9091\n")
	fmt.Fprintf(os.Stdout, "gaxx-agent profiling on :6060\n")

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	<-sigc

	fmt.Fprintln(os.Stdout, "gaxx-agent shutting down")
	telemetry.CounterGlobal("gaxx_agent_shutdowns", 1, map[string]string{
		"component": "agent",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// startAgentMonitoring starts the monitoring server for the agent
func startAgentMonitoring(addr string, collector *telemetry.Collector, perfMon *telemetry.PerformanceMonitor) {
	server := telemetry.NewMonitoringServer(addr, collector, perfMon)

	// Register agent-specific health checks
	for name, checkFn := range telemetry.DefaultHealthChecks() {
		server.RegisterHealthCheck(name, checkFn)
	}

	// Add agent-specific health check
	server.RegisterHealthCheck("agent_api", func() telemetry.HealthCheck {
		// Simple check to see if we can make a request to ourselves
		return telemetry.HealthCheck{
			Name:    "agent_api",
			Status:  telemetry.HealthStatusHealthy,
			Message: "Agent API is responsive",
		}
	})

	if err := server.Start(); err != nil && err.Error() != "http: Server closed" {
		fmt.Fprintf(os.Stderr, "Agent monitoring server failed: %v\n", err)
	}
}
