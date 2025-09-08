# Gaxx Performance Monitoring & Optimization

This document describes the comprehensive performance monitoring and optimization system built into Gaxx.

## Overview

Gaxx includes a multi-layered performance monitoring system that provides:

- **Real-time telemetry collection** with custom metrics
- **Health monitoring** with automated health checks
- **Performance profiling** using Go's built-in pprof
- **HTTP dashboards** for monitoring and debugging
- **Prometheus-compatible metrics** for integration

## Architecture

### Telemetry System

The telemetry system is built around several key components:

1. **Collector**: Centralized metric collection and storage
2. **Performance Monitor**: System resource monitoring
3. **Monitoring Server**: HTTP endpoints for health and metrics
4. **Profiling Server**: Go pprof integration for deep analysis

### Key Metrics Tracked

#### Application Metrics
- `gaxx_app_starts` - Application startup counter
- `gaxx_app_errors` - Application error counter
- `gaxx_tasks_started` - Task execution counter
- `gaxx_task_duration` - Task execution time
- `gaxx_task_success_rate_percent` - Task success rate

#### Agent Metrics
- `gaxx_agent_starts` - Agent startup counter
- `gaxx_agent_heartbeats` - Health check counter
- `gaxx_agent_exec_requests` - Command execution requests
- `gaxx_agent_exec_duration` - Command execution time
- `gaxx_agent_exec_output_size` - Command output size
- `gaxx_agent_exec_successful/failed` - Success/failure counters

#### System Metrics
- `gaxx_memory_heap_bytes` - Heap memory usage
- `gaxx_goroutines_total` - Active goroutine count
- `gaxx_gc_total` - Garbage collection counter
- `gaxx_uptime_seconds` - Application uptime

#### Performance Metrics
- `gaxx_node_execution_duration` - Per-node execution time
- `gaxx_file_transfer_duration` - File transfer timing
- `gaxx_file_transfer_throughput_mbps` - Transfer speed

## Configuration

Enable telemetry in your `config.yaml`:

```yaml
telemetry:
  enabled: true
  otlp_endpoint: ""  # Optional OTLP endpoint
  monitoring_port: 9090  # Monitoring dashboard port
  metrics_interval: 30   # Metrics collection interval
```

## Monitoring Endpoints

### Main Application (port 9090)
- `/health` - Health status (JSON)
- `/metrics` - Prometheus metrics (text)
- `/dashboard` - Web dashboard (HTML)
- `/api/metrics` - Metrics API (JSON)
- `/api/health` - Health API (JSON)

### Agent (port 9091)
- `/health` - Agent health status
- `/metrics` - Agent metrics
- `/dashboard` - Agent dashboard
- `/api/metrics` - Agent metrics API
- `/api/health` - Agent health API

### Profiling (port 6060)
- `/debug/pprof/` - Go pprof endpoints
- `/debug/stats` - Runtime statistics
- `/debug/gc` - Trigger garbage collection
- `/debug/build` - Build information

## Usage Examples

### Basic Monitoring

```bash
# Check agent health
curl http://localhost:9091/health

# Get metrics in Prometheus format
curl http://localhost:9091/metrics

# View web dashboard
open http://localhost:9091/dashboard
```

### Performance Profiling

```bash
# CPU profile (30 seconds)
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profile
go tool pprof http://localhost:6060/debug/pprof/goroutine

# Get runtime stats
curl http://localhost:6060/debug/stats | jq
```

### Metrics Analysis

```bash
# Get all metrics
curl -s http://localhost:9091/api/metrics | jq

# Filter specific metrics
curl -s http://localhost:9091/api/metrics | jq '.[] | select(.name | contains("exec"))'

# Monitor success rates
curl -s http://localhost:9091/api/metrics | jq '.[] | select(.name == "gaxx_task_success_rate_percent")'
```

## Performance Optimization Tips

### 1. Memory Management

Monitor heap usage and GC frequency:
```bash
curl http://localhost:6060/debug/stats | jq '.memory.heap_alloc_mb'
```

Optimize for:
- Keep heap usage under 100MB for typical workloads
- Minimize GC pauses (< 1ms typical)
- Watch for memory leaks in long-running operations

### 2. Concurrency Tuning

Monitor goroutine counts:
```bash
curl http://localhost:9091/api/metrics | jq '.[] | select(.name == "gaxx_goroutines_total")'
```

Recommendations:
- Limit concurrent task executions with `--concurrency` flag
- Monitor goroutine count during peak load
- Use semaphores to prevent goroutine explosion

### 3. Network Performance

Monitor agent communication:
```bash
curl -s http://localhost:9091/api/metrics | jq '.[] | select(.name | contains("request_duration"))'
```

Optimize for:
- Keep agent response times under 100ms
- Monitor file transfer throughput
- Use connection pooling for high-frequency operations

### 4. Task Optimization

Monitor task execution metrics:
```bash
curl -s http://localhost:9091/api/metrics | jq '.[] | select(.name | contains("task"))'
```

Best practices:
- Chunk large inputs appropriately (balance parallelism vs overhead)
- Monitor task success rates and optimize retry logic
- Use timeouts to prevent stuck operations

## Integration with External Systems

### Prometheus Integration

Add to your `prometheus.yml`:
```yaml
scrape_configs:
  - job_name: 'gaxx-cli'
    static_configs:
      - targets: ['localhost:9090']
  - job_name: 'gaxx-agent'
    static_configs:
      - targets: ['localhost:9091']
```

### Grafana Dashboards

Key metrics to visualize:
- Task execution rate and success rate
- Agent response times and throughput
- Memory usage and GC frequency
- Goroutine count trends

### Alerting Rules

Example Prometheus alerting rules:
```yaml
groups:
  - name: gaxx
    rules:
      - alert: GaxxHighMemoryUsage
        expr: gaxx_memory_heap_bytes > 200000000  # 200MB
        annotations:
          summary: "Gaxx high memory usage"
      
      - alert: GaxxTaskFailureRate
        expr: gaxx_task_success_rate_percent < 90
        annotations:
          summary: "Gaxx task failure rate high"
```

## Troubleshooting

### High Memory Usage
1. Check heap profile: `go tool pprof http://localhost:6060/debug/pprof/heap`
2. Look for memory leaks in long-running operations
3. Reduce chunk sizes or concurrency

### Slow Performance
1. Check CPU profile: `go tool pprof http://localhost:6060/debug/pprof/profile`
2. Monitor agent response times
3. Check network latency to nodes

### High Goroutine Count
1. Check goroutine profile: `go tool pprof http://localhost:6060/debug/pprof/goroutine`
2. Reduce concurrency settings
3. Look for goroutine leaks

## Performance Benchmarks

Typical performance characteristics:

### Single Agent Performance
- **Heartbeat response**: < 1ms
- **Simple command execution**: < 10ms
- **Memory usage**: < 20MB baseline
- **Goroutines**: < 20 baseline

### Fleet Performance (100 nodes)
- **Task distribution**: < 5s
- **Result collection**: < 2s
- **Memory usage**: < 100MB
- **Network overhead**: < 1MB per node

### Scalability Limits
- **Max nodes tested**: 1000
- **Max concurrent tasks**: 10,000
- **Max file size**: 100MB per file
- **Max memory usage**: 1GB for large fleets

## Monitoring Best Practices

1. **Always enable telemetry** in production
2. **Set up alerting** for critical metrics
3. **Monitor trends** over time, not just snapshots
4. **Profile regularly** during development
5. **Use dashboards** for real-time monitoring
6. **Archive metrics** for historical analysis
