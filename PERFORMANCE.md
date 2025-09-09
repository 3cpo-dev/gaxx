# Performance Guide

Gaxx includes comprehensive performance monitoring and optimization tools.

## Quick Start

### Enable Monitoring
```bash
# Start agent with monitoring
gaxx-agent &

# Check health and metrics
curl http://localhost:9091/health
curl http://localhost:9091/metrics
curl http://localhost:9091/dashboard
```

### OTLP Export
```bash
# Export to observability stack
export GAXX_OTLP_ENDPOINT="http://otel-collector:4318/v1/metrics"
gaxx-agent
```

## Monitoring Endpoints

| Endpoint | Purpose | Port |
|----------|---------|------|
| `/health` | Health status | 9091 |
| `/metrics` | Prometheus metrics | 9091 |
| `/dashboard` | Web dashboard | 9091 |
| `/debug/pprof/` | Performance profiling | 6060 |

## Key Metrics

### Application Metrics
- `gaxx_app_starts` - Application startups
- `gaxx_app_errors` - Error count
- `gaxx_tasks_executed` - Tasks completed
- `gaxx_tasks_failed` - Task failures

### System Metrics
- `gaxx_memory_usage` - Memory consumption
- `gaxx_cpu_usage` - CPU utilization
- `gaxx_goroutines` - Active goroutines
- `gaxx_gc_duration` - Garbage collection time

### Network Metrics
- `gaxx_http_requests` - HTTP request count
- `gaxx_http_duration` - Request latency
- `gaxx_ssh_connections` - SSH connections
- `gaxx_agent_heartbeats` - Agent health

## Performance Optimization

### 1. Resource Tuning
```yaml
# config.yaml
defaults:
  retries: 3
  timeout_seconds: 300
  concurrency: 10  # Adjust based on resources
```

### 2. Monitoring Configuration
```yaml
telemetry:
  enabled: true
  monitoring_port: 9090
  profiling_port: 6060
  otlp_endpoint: "http://otel-collector:4318/v1/metrics"
  metrics_interval: 30
```

### 3. Profiling Analysis
```bash
# CPU profiling
go tool pprof http://localhost:6060/debug/pprof/profile

# Memory profiling  
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine analysis
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

## Cloud Provider Optimization

### Linode
- **Rate Limiting**: 2 req/sec (built-in)
- **Retry Logic**: Exponential backoff with jitter
- **Validation**: Pre-request parameter validation

### Vultr
- **Rate Limiting**: 2 req/sec (built-in)
- **Error Handling**: Automatic retry on 5xx errors
- **Pagination**: Efficient large dataset handling

## Performance Best Practices

### 1. Fleet Sizing
```bash
# Start small, scale up
gaxx spawn --provider linode --count 5 --name test
# Monitor performance, then scale
gaxx spawn --provider linode --count 50 --name production
```

### 2. Task Distribution
```yaml
# Optimize chunk sizes
chunk_size: 100  # Adjust based on task complexity
```

### 3. Resource Monitoring
```bash
# Monitor during execution
watch -n 1 'curl -s http://localhost:9091/metrics | grep gaxx_tasks'
```

## Troubleshooting

### High Memory Usage
```bash
# Check memory profile
go tool pprof http://localhost:6060/debug/pprof/heap
# Look for memory leaks in heap profile
```

### Slow Task Execution
```bash
# Check CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile
# Analyze hot spots
```

### Network Issues
```bash
# Monitor HTTP metrics
curl -s http://localhost:9091/metrics | grep http
# Check for high latency or errors
```

## Integration Examples

### Prometheus + Grafana
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'gaxx'
    static_configs:
      - targets: ['localhost:9091']
```

### Jaeger Tracing
```bash
export GAXX_OTLP_ENDPOINT="http://jaeger:14268/api/traces"
gaxx-agent
```

## Performance Targets

| Metric | Target | Alert Threshold |
|--------|--------|-----------------|
| Task Execution | < 1s | > 5s |
| Memory Usage | < 100MB | > 500MB |
| HTTP Latency | < 100ms | > 1s |
| Error Rate | < 1% | > 5% |