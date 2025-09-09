# Gaxx

Distributed task runner for short-lived VPS fleets. Spin up cloud servers, run parallel tasks, collect results, then tear down—all with enterprise security.

## Quick Start

```bash
# 1. Build and install
make build
echo 'export PATH="/path/to/gaxx/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

# 2. Initialize
gaxx init

# 3. Configure providers (edit ~/.config/gaxx/config.yaml)
# Add Linode/Vultr tokens to ~/.config/gaxx/secrets.env

# 4. Create and use fleet
gaxx spawn --provider linode --count 3 --name myfleet
gaxx run --name myfleet -- echo "Hello from $(hostname)"
gaxx delete  # Clean up
```

## Commands

| Command | Description |
|---------|-------------|
| `gaxx init` | Initialize config and SSH keys |
| `gaxx spawn --provider <name> --count <n>` | Create fleet |
| `gaxx run --name <fleet> [command]` | Execute commands |
| `gaxx scan --name <fleet> --module <yaml>` | Upload files + run tasks |
| `gaxx ls` | List running instances |
| `gaxx delete` | Delete all instances |
| `gaxx ssh --name <fleet>` | SSH to fleet node |

## Configuration

### Basic Setup (`~/.config/gaxx/config.yaml`)
```yaml
providers:
  default: linode
  linode:
    region: us-east
    type: g6-nanode-1
    image: linode/ubuntu22.04
    tags: [gaxx]

telemetry:
  enabled: true
  otlp_endpoint: "http://otel-collector:4318/v1/metrics"
```

### Secrets (`~/.config/gaxx/secrets.env`)
```bash
LINODE_TOKEN=your-linode-token
VULTR_API_KEY=your-vultr-key
```

## Examples

### Basic Command Execution
```bash
gaxx spawn --provider linode --count 5 --name workers
gaxx run --name workers -- echo "Processing $(date)"
gaxx delete
```

### File Upload and Processing
```bash
# Upload files with SFTP + checksum verification
gaxx scan --name workers --module modules/dns_bruteforce.yaml \
  --upload wordlist.txt --inputs wordlist.txt
```

### Production Security
```bash
# mTLS agent authentication
export GAXX_AGENT_TLS_CERT=server.pem
export GAXX_AGENT_TLS_KEY=server.key
export GAXX_AGENT_CLIENT_CA=client-ca.pem
export GAXX_AGENT_REQUIRE_MTLS=true
gaxx-agent
```

## Features

- **Multi-Provider**: Linode, Vultr, LocalSSH with retry logic
- **Security**: mTLS, SSH keys, SFTP with checksums
- **Monitoring**: OTLP metrics, health checks, profiling
- **Resilience**: Rate limiting, validation, error handling

## Task Modules

Pre-built YAML modules for common tasks:

```yaml
# modules/http_probe.yaml
name: http_probe
command: curl
args: ["-sS", "-o", "/dev/null", "-w", "%{http_code}\n", "{{ item }}"]
env: {timeout: "10"}
inputs: ["${targets}"]
chunk_size: 50
```

## Architecture

- **CLI**: Orchestration with telemetry
- **Agent**: Lightweight HTTP server on nodes
- **Providers**: Cloud APIs with resilience
- **Modules**: YAML task definitions

## Development

```bash
make build          # Build binaries
make test-all       # Run all tests
make install        # System install
make install-user   # User install
```

## License

Apache-2.0