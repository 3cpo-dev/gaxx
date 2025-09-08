# Gaxx
Gaxx is a distributed task runner for short-lived fleets of VPS machines. It quickly rents and coordinates temporary cloud servers to run lots of small jobs in parallel, installing a lightweight agent on each server, splitting your input into chunks, running them simultaneously, then collecting results—with sensible security defaults.

## What is it?
Gaxx orchestrates "burst" fleets of servers for parallel task execution. It's designed to spin up a large pool of servers for a short time, finish work fast, and shut them down to save cost and reduce manual operations.

## How it works
- **Provisioning**: Spins up new VPS nodes or connects to existing ones
- **Agent Deployment**: Installs a lightweight HTTP agent on each node  
- **Task Distribution**: Splits your input into manageable chunks and fans them out
- **Parallel Execution**: Runs tasks simultaneously across all nodes for speed
- **Result Collection**: Aggregates and returns combined outputs
- **Cleanup**: Tears down or detaches from nodes when jobs complete

## Why it's useful
- **Speed**: Faster batch processing through massive parallelization
- **Cost**: Pay only for burst windows instead of keeping machines idle
- **Simplicity**: Automated provisioning, distribution, and result collection
- **Security**: Secure defaults with SSH keys, strict host management, and agent-based execution

## Practical Example
Process 1 million URLs: Gaxx spins up hundreds of VPS instances, deploys agents, splits the URL list into chunks, runs fetch/parse tasks simultaneously, and returns one combined output file—then tears down the fleet.


## Features
- **Fleet Management**: Create, list, and delete fleets with multiple providers
- **Agent-Based Execution**: Lightweight HTTP agent for command execution and monitoring
- **Task Modules**: YAML-based task definitions with variable substitution and chunking
- **File Upload**: Efficient file transfer to nodes before task execution
- **Parallel Processing**: Configurable concurrency with intelligent load distribution
- **Monitoring & Telemetry**: Built-in performance monitoring, health checks, and profiling
- **Security**: SSH key-only authentication, strict host management, non-root execution
- **Multi-Provider**: Support for Linode, Vultr, and local SSH hosts
- **Structured Logging**: JSON-based logging with configurable levels

## Commands
```bash
# Core Commands
gaxx init                                    # Initialize configuration and SSH keys
gaxx spawn --provider <name> --count <n>    # Create a fleet of nodes
gaxx run --name <fleet> [command]           # Execute commands on fleet
gaxx scan --name <fleet> --module <yaml>    # Upload files and run tasks
gaxx ls --name <fleet>                      # List fleet nodes
gaxx delete --name <fleet>                  # Delete fleet

# Utility Commands  
gaxx ssh --name <fleet> --node <name>       # SSH into specific node
gaxx scp --name <fleet> <src>:<dst>         # Transfer files
gaxx version                                 # Show version info
gaxx --help                                  # Show help
```

### Available Flags
- `--config string`: Custom config file path
- `--log string`: Log level (debug, info, warn, error, fatal)
- `--proxy string`: HTTP proxy for debugging
- Provider-specific flags for regions, sizes, images, etc.

## Installation & Setup

### 1. Build from Source
```bash
# Clone and build
git clone https://github.com/3cpo-dev/gaxx.git
cd gaxx
make build

# Or build specific components
go build -o bin/gaxx ./cmd/gaxx
go build -o bin/gaxx-agent ./cmd/gaxx-agent
```

### 2. Initialize Configuration
```bash
# First-time setup (creates ~/.config/gaxx/)
./bin/gaxx init

# This creates:
# - SSH key pair (id_ed25519)
# - Default config.yaml
# - Known hosts file
# - Secrets template
```

### 3. Configure Providers
Edit `~/.config/gaxx/config.yaml`:
```yaml
providers:
  default: localssh
  localssh:
    hosts:
      - name: local-test
        ip: 127.0.0.1
        user: youruser
        key_path: ~/.config/gaxx/ssh/id_ed25519
        port: 22
  # Add Linode/Vultr credentials in secrets.env
telemetry:
  enabled: true
  monitoring_port: 9090
```

## Quick Examples

### Basic Command Execution
```bash
# Run simple command
./bin/gaxx run --name local-test -- echo "Hello World"

# Run with environment variables
./bin/gaxx run --name local-test --env USER=test -- echo "User: $USER"
```

### Using Task Modules
```bash
# HTTP probes with built-in module
echo "https://example.com" > targets.txt
./bin/gaxx run --name local-test --module modules/http_probe.yaml --inputs targets.txt

# Port scanning
echo "127.0.0.1" > hosts.txt
./bin/gaxx run --name local-test --module modules/port_scan.yaml --inputs hosts.txt --env ports=22,80,443
```

### File Upload and Processing
```bash
# Upload files and run tasks
./bin/gaxx scan --name local-test --module modules/dns_bruteforce.yaml \
  --upload wordlist.txt --inputs wordlist.txt --env domain=example.com
```

### Monitoring and Health
```bash
# Start agent with monitoring
./bin/gaxx-agent &

# Check health and metrics
curl http://localhost:9091/health
curl http://localhost:9091/metrics
curl http://localhost:6060/debug/stats
```

## Task Modules

Gaxx includes pre-built YAML modules for common tasks:

### Available Modules
- **`http_probe.yaml`**: HTTP GET probes with response time and status codes
- **`port_scan.yaml`**: TCP port scanning using netcat across multiple targets
- **`dns_bruteforce.yaml`**: Subdomain enumeration using wordlists

### Module Structure
```yaml
name: my_task
description: Task description
command: curl
args: ["-sS", "-o", "/dev/null", "-w", "%{http_code}\n", "{{ item }}"]
env: {timeout: "10"}
inputs: ["${targets}"]
chunk_size: 50
```

### Custom Modules
Create your own YAML modules with:
- **Variable substitution**: Use `{{ item }}` for input templating
- **Environment variables**: Define in `env` section
- **Input chunking**: Distribute work across nodes with `chunk_size`
- **Command flexibility**: Any command available on target systems

## Architecture

### Components
- **CLI (`gaxx`)**: Main orchestration tool with telemetry and monitoring
- **Agent (`gaxx-agent`)**: Lightweight HTTP server deployed on nodes
- **Providers**: Pluggable backends for Linode, Vultr, and local SSH
- **Modules**: YAML task definitions with templating and chunking

### Communication
- **Agent API**: HTTP endpoints for command execution (`/v0/exec`) and health (`/v0/heartbeat`)
- **Monitoring**: Health checks, Prometheus metrics, and pprof profiling
- **SSH Fallback**: Direct SSH execution when agents unavailable

## Provider Status

| Provider | Status | Description |
|----------|--------|-------------|
| **LocalSSH** | ✅ **Implemented** | Connect to existing SSH hosts |
| **Linode** | 🚧 **Stub** | Cloud provider integration planned |
| **Vultr** | 🚧 **Stub** | Cloud provider integration planned |

## Security Features
- **SSH Key Authentication**: Ed25519 keys with strict host verification
- **Agent Security**: HTTP-only agent with command execution isolation
- **Non-root Execution**: Default user `gx` with minimal privileges
- **Network Security**: Known hosts management and connection validation
- **Audit Trail**: Comprehensive logging of all operations

## Performance & Monitoring
- **Real-time Metrics**: Memory, CPU, goroutines, and custom business metrics
- **Health Checks**: Automated health monitoring for all components
- **Profiling**: Built-in pprof endpoints for performance analysis
- **Telemetry**: OTLP-compatible metrics export (configurable)
- **Dashboard**: Web interface for metrics and health status

## Development

### Requirements
- Go 1.21+
- Make for build automation

### Build & Test
```bash
# Build all components
make build

# Run unit tests
make test-unit

# Run integration tests
make test-integration

# Run all tests
make test-all

# Test monitoring system
make test-monitoring
```

### Project Structure
```
cmd/                    # Main binaries
├── gaxx/              # CLI tool
└── gaxx-agent/        # Node agent

internal/              # Core implementation
├── agent/             # Agent HTTP server
├── core/              # Business logic
├── providers/         # Cloud provider interfaces
├── ssh/               # SSH utilities
└── telemetry/         # Monitoring & metrics

modules/               # Pre-built task modules
pkg/api/              # Public API types
```

## License
Apache-2.0


