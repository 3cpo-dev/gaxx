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
- **Agent-Based Execution**: Lightweight HTTP agent with optional mTLS authentication
- **Task Modules**: YAML-based task definitions with variable substitution and chunking
- **SFTP File Transfer**: Secure file upload with SHA256 checksum verification
- **Parallel Processing**: Configurable concurrency with intelligent load distribution
- **OTLP Telemetry**: OpenTelemetry-compatible metrics export for observability
- **Cloud Provider Resilience**: Retry logic, rate limiting, and validation
- **Security**: SSH keys, mTLS authentication, strict host management, non-root execution
- **Multi-Provider**: Support for Linode, Vultr, and local SSH hosts with production hardening
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
  profiling_port: 6060
  otlp_endpoint: "http://otel-collector:4318/v1/metrics"
  
security:
  require_mtls: false  # Set to true for production
```

### 4. Set Environment Variables
For production deployments, configure security and cloud provider access:
```bash
# Agent Security (optional)
export GAXX_AGENT_TOKEN="your-secret-token"           # Bearer token auth
export GAXX_AGENT_TLS_CERT="/path/to/server.crt"      # Server certificate
export GAXX_AGENT_TLS_KEY="/path/to/server.key"       # Server private key
export GAXX_AGENT_CLIENT_CA="/path/to/client-ca.crt"  # Client CA for mTLS
export GAXX_AGENT_REQUIRE_MTLS="true"                 # Enforce mTLS

# Cloud Provider Credentials
export LINODE_TOKEN="your-linode-token"
export VULTR_API_KEY="your-vultr-key"
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

### SFTP File Upload and Processing
```bash
# Upload large files with checksum verification (SFTP)
./bin/gaxx scan --name local-test --module modules/dns_bruteforce.yaml \
  --upload large-wordlist.txt --inputs large-wordlist.txt --env domain=example.com

# Files are automatically verified with SHA256 checksums
# Failed transfers are automatically retried or cleaned up
```

### Cloud Fleet Operations
```bash
# Create Linode fleet with hardened providers
./bin/gaxx spawn --provider linode --count 10 --region us-east --name test-fleet

# Automatic cloud-init deployment with agent installation
# Built-in retry logic, rate limiting, and validation

# Run tasks across cloud fleet
./bin/gaxx run --name test-fleet --module modules/port_scan.yaml --inputs targets.txt

# Clean up with resilient deletion
./bin/gaxx delete --name test-fleet
```

### Production Security Setup
```bash
# Generate mTLS certificates for production
openssl genrsa -out ca-key.pem 4096
openssl req -new -x509 -days 365 -key ca-key.pem -out ca.pem
openssl genrsa -out server-key.pem 4096
openssl req -new -key server-key.pem -out server.csr
openssl x509 -req -days 365 -in server.csr -CA ca.pem -CAkey ca-key.pem -out server.pem

# Start agent with mTLS
export GAXX_AGENT_TLS_CERT=server.pem
export GAXX_AGENT_TLS_KEY=server-key.pem
export GAXX_AGENT_CLIENT_CA=ca.pem
export GAXX_AGENT_REQUIRE_MTLS=true
./bin/gaxx-agent

# Client connections now require valid certificates
```

### Advanced Monitoring and Observability
```bash
# Start with OTLP export to observability stack
export GAXX_OTLP_ENDPOINT="http://jaeger:14268/api/traces"
./bin/gaxx-agent &

# Check comprehensive health and metrics
curl http://localhost:9091/health          # Health status
curl http://localhost:9091/metrics         # Prometheus metrics  
curl http://localhost:9091/dashboard       # Web dashboard
curl http://localhost:6060/debug/pprof/    # Performance profiling

# Metrics include:
# - Request latency and throughput
# - System resource usage
# - Agent connection status
# - Task execution statistics
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
- **Agent API**: HTTP/HTTPS endpoints with optional mTLS authentication
- **SFTP Transfer**: Secure file upload with integrity verification
- **OTLP Export**: OpenTelemetry-compatible metrics for observability stacks
- **SSH Fallback**: Direct SSH execution when agents unavailable

## Provider Status

| Provider | Status | Features |
|----------|--------|----------|
| **LocalSSH** | ✅ | SSH hosts, SFTP transfer, agent deployment |
| **Linode** | ✅ | Fleet creation, cloud-init, retry logic, rate limiting |
| **Vultr** | ✅ | Fleet creation, cloud-init, retry logic, rate limiting |

## Security Features
- **mTLS Authentication**: Optional mutual TLS for agent communication
- **SSH Key Authentication**: Ed25519 keys with strict host verification  
- **Bearer Token Auth**: Optional token-based agent authentication
- **SFTP with Integrity**: File transfers with SHA256 checksum verification
- **Non-root Execution**: Default user `gx` with minimal privileges
- **Network Security**: Known hosts management and connection validation
- **Audit Trail**: Comprehensive logging of all operations and security events

## Performance & Monitoring
- **OTLP Export**: OpenTelemetry Protocol-compatible metrics for Jaeger, Prometheus, etc.
- **Real-time Metrics**: Memory, CPU, goroutines, request latency, and business metrics
- **Health Checks**: Automated health monitoring with detailed status reporting
- **Performance Profiling**: Built-in pprof endpoints for CPU, memory, and goroutine analysis
- **Web Dashboard**: Real-time web interface for metrics visualization and health status
- **Cloud Provider Resilience**: Retry logic, rate limiting, and transient error handling

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
├── agent/             # Agent HTTP server with mTLS support
├── core/              # Business logic and SFTP transfers
├── providers/         # Cloud provider interfaces with resilience
├── ssh/               # SSH utilities and key management
└── telemetry/         # OTLP monitoring & metrics

modules/               # Pre-built task modules
pkg/api/              # Public API types
```

## Production Deployment

### Environment Variables Reference

#### Agent Security
```bash
# Bearer token authentication
GAXX_AGENT_TOKEN="your-secret-token"

# TLS/mTLS configuration  
GAXX_AGENT_TLS_CERT="/path/to/server.crt"      # Server certificate
GAXX_AGENT_TLS_KEY="/path/to/server.key"       # Server private key
GAXX_AGENT_CLIENT_CA="/path/to/client-ca.crt"  # Client CA for mTLS
GAXX_AGENT_REQUIRE_MTLS="true"                 # Enforce client certificates
```

#### Telemetry Configuration
```bash
# OTLP export endpoint
GAXX_OTLP_ENDPOINT="http://otel-collector:4318/v1/metrics"

# Alternative observability endpoints
GAXX_OTLP_ENDPOINT="http://jaeger:14268/api/traces"
GAXX_OTLP_ENDPOINT="http://prometheus:9090/api/v1/write"
```

#### Cloud Provider Credentials
```bash
# Linode API access
LINODE_TOKEN="your-linode-personal-access-token"

# Vultr API access  
VULTR_API_KEY="your-vultr-api-key"
```

### Production Checklist
- [ ] Generate and configure mTLS certificates
- [ ] Set strong bearer tokens for agent authentication
- [ ] Configure OTLP endpoints for observability stack
- [ ] Set up cloud provider credentials securely
- [ ] Enable comprehensive logging and monitoring
- [ ] Test retry logic and error handling
- [ ] Verify SFTP integrity checks are working
- [ ] Configure proper network security and firewalls

## License
Apache-2.0


