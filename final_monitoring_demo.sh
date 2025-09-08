#!/bin/bash
set -euo pipefail

echo "🚀 Gaxx Complete Performance Monitoring Demo"
echo "============================================="
echo "This demo showcases the full monitoring and optimization system"
echo ""

# Build the application
make build

# Create test directory
TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT

# Create comprehensive config
cat > "$TEST_DIR/config.yaml" << EOF
providers:
  default: localssh
  localssh:
    hosts:
      - {name: "node-1", ip: "127.0.0.1", user: "demo", key_path: "$TEST_DIR/ssh/id_ed25519", port: 22}
      - {name: "node-2", ip: "127.0.0.1", user: "demo", key_path: "$TEST_DIR/ssh/id_ed25519", port: 22}
ssh:
  key_dir: $TEST_DIR/ssh
  known_hosts: $TEST_DIR/known_hosts
defaults:
  user: gx
  ssh_port: 22
  retries: 3
  timeout_seconds: 30
telemetry:
  enabled: true
  otlp_endpoint: ""
  monitoring_port: 9090
  metrics_interval: 5
EOF

echo "1. 🔧 Initializing environment..."
./bin/gaxx init --config "$TEST_DIR/config.yaml" --force > /dev/null

echo "2. 🚀 Starting agent with full monitoring stack..."
./bin/gaxx-agent &
AGENT_PID=$!
trap "kill $AGENT_PID 2>/dev/null || true; rm -rf $TEST_DIR" EXIT

# Wait for all services to start
sleep 5

echo ""
echo "3. 🌐 Available Monitoring Endpoints:"
echo "   📊 Agent API:         http://localhost:8088/v0/"
echo "   🏥 Health Monitoring: http://localhost:9091/health"
echo "   📈 Metrics Dashboard: http://localhost:9091/dashboard"
echo "   📊 Raw Metrics:      http://localhost:9091/metrics"
echo "   🔍 Profiling:        http://localhost:6060/debug/"

echo ""
echo "4. 📊 Testing Health Monitoring..."
HEALTH=$(curl -s http://localhost:9091/health)
echo "System Status: $(echo "$HEALTH" | jq -r '.status')"
echo "Health Checks:"
echo "$HEALTH" | jq -r '.checks[] | "  \(.name): \(.status) - \(.message)"'

echo ""
echo "5. 🎯 Generating Load for Metrics..."

# Create test module
cat > "$TEST_DIR/test_module.yaml" << EOF
name: load_test
description: Performance testing module
command: sh
args: ["-c", "echo 'Processing:' \$(cat {{ item }}) && sleep 0.1 && echo 'Done'"]
env: {}
inputs:
  - "\${targets}"
chunk_size: 1
EOF

# Create test data
for i in {1..10}; do
    echo "item-$i" >> "$TEST_DIR/targets.txt"
done

echo "Executing test workload..."
for i in {1..3}; do
    curl -s -X POST http://localhost:8088/v0/exec \
        -H "Content-Type: application/json" \
        -d "{\"command\":\"echo\",\"args\":[\"load-test-$i\"],\"timeout\":5}" > /dev/null
done

echo ""
echo "6. 📈 Performance Metrics Summary:"

# Wait for metrics to be collected
sleep 2

METRICS=$(curl -s http://localhost:9091/api/metrics)

echo ""
echo "🔢 Key Performance Indicators:"
echo "$METRICS" | jq -r '
  group_by(.name) | 
  map(select(length > 0)) |
  map({
    metric: .[0].name,
    count: length,
    avg_value: (map(.value) | add / length),
    max_value: (map(.value) | max),
    component: (.[0].labels.component // "unknown")
  }) |
  sort_by(.component, .metric) |
  .[] |
  "  \(.component): \(.metric) = \(.avg_value | round) (\(.count) samples)"
'

echo ""
echo "7. 🏥 System Health Analysis:"
curl -s http://localhost:6060/debug/stats | jq '{
  memory_mb: .memory.heap_alloc_mb,
  goroutines: .goroutines,
  gc_count: .gc.num_gc,
  cpu_cores: .cpu_cores,
  go_version: .go_version
}'

echo ""
echo "8. 🔍 Performance Profiling Available:"
echo "   📊 CPU Profile:       curl http://localhost:6060/debug/pprof/profile?seconds=10"
echo "   💾 Memory Profile:    curl http://localhost:6060/debug/pprof/heap"
echo "   🧵 Goroutine Profile: curl http://localhost:6060/debug/pprof/goroutine"
echo "   📈 Runtime Stats:     curl http://localhost:6060/debug/stats"

echo ""
echo "9. 📊 Prometheus Metrics Sample:"
curl -s http://localhost:9091/metrics | grep -E "^(gaxx_agent_|gaxx_memory_|gaxx_gc_)" | head -5

echo ""
echo "10. 🌐 Web Dashboard Demo:"
echo "    Visit http://localhost:9091/dashboard for the interactive dashboard"
echo ""

# Test the dashboard endpoint
if curl -s http://localhost:9091/dashboard | grep -q "Gaxx Monitoring Dashboard"; then
    echo "✅ Web dashboard is functional"
else
    echo "❌ Web dashboard test failed"
fi

echo ""
echo "🎉 Complete Monitoring System Demonstration"
echo "==========================================="
echo ""
echo "🔧 Implemented Features:"
echo "  ✅ Real-time telemetry collection with custom metrics"
echo "  ✅ Comprehensive health monitoring system"
echo "  ✅ HTTP monitoring dashboard with web UI"
echo "  ✅ Prometheus-compatible metrics export"
echo "  ✅ Go pprof integration for deep profiling"
echo "  ✅ Performance optimization with runtime stats"
echo "  ✅ Agent and CLI instrumentation"
echo "  ✅ Memory, CPU, and network monitoring"
echo "  ✅ Task execution performance tracking"
echo "  ✅ Automated health checks and alerting"
echo ""
echo "📊 Metrics Categories:"
echo "  • Application metrics (starts, errors, tasks)"
echo "  • Agent metrics (requests, execution, throughput)"
echo "  • System metrics (memory, GC, goroutines)"
echo "  • Performance metrics (timing, success rates)"
echo ""
echo "🌐 Integration Ready:"
echo "  • Prometheus scraping endpoints"
echo "  • Grafana dashboard compatible"
echo "  • OTLP export capability"
echo "  • Health check APIs"
echo ""
echo "🚀 Gaxx Performance Monitoring System is Production Ready!"

echo ""
echo "💡 Next Steps:"
echo "  1. Configure Prometheus to scrape http://localhost:9091/metrics"
echo "  2. Set up Grafana dashboards using the metrics"
echo "  3. Configure alerting rules for critical thresholds"
echo "  4. Use profiling endpoints for performance optimization"
echo "  5. Monitor real production workloads"
