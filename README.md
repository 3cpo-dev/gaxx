# Gaxx

Spin up a temporary fleet of VPS instances, run tasks across them in parallel, collect the results, and tear everything down—built for speed, simplicity, and security-minded operations.

Gaxx lets teams prototype distributed workloads, load-test services, and batch-run scripts with minimal setup across Linode and Vultr providers

## Quick Start

Get up and running in minutes: install the CLI, set provider credentials, spawn a fleet, run a command, then delete the infrastructure when finished.
This flow keeps costs predictable and environments clean by default.

```bash
# 1. Build and install
make build
echo 'export PATH="/path/to/gaxx/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

# 2. Set up credentials
export LINODE_TOKEN="your-linode-token"
export VULTR_API_KEY="your-vultr-key"

# 3. Create and use fleet
gaxx spawn --provider linode --count 3 --name myfleet
gaxx run --name myfleet --command "echo Hello from $(hostname)"
gaxx delete myfleet
```

## Commands

| Command | Description |
|---------|-------------|
| `gaxx spawn --provider <name> --count <n> --name <fleet>` | Create fleet |
| `gaxx run --name <fleet> --command <cmd>` | Execute commands |
| `gaxx ls [fleet-name]` | List instances |
| `gaxx delete [fleet-name]` | Delete fleet |
| `gaxx metrics` | Show performance metrics |
| `gaxx version` | Show version info |

## Configuration

### Environment Variables

Gaxx reads credentials from environment variables and can optionally use a YAML config for defaults such as provider, region, and concurrency.
Keep credentials outside the repo and use a profile or CI secret store for safety and reproducibility.

```bash
# Cloud provider credentials (required)
LINODE_TOKEN=your-linode-token
VULTR_API_KEY=your-vultr-key

# Optional: Custom config file
export GAXX_CONFIG=/path/to/config.yaml
```

### Config File (`~/.config/gaxx/config.yaml`)

Use the config file to set smart defaults for repeatable runs, and override via flags as needed during experimentation.
This makes both local development and CI pipelines straightforward without duplicating flags everywhere.

```yaml
provider: linode
region: us-east
ssh_key_path: ~/.config/gaxx/ssh/id_ed25519
monitoring: true
concurrency: 10
```

### Basic Usage
```bash
# Create 5 instances
gaxx spawn --provider linode --count 5 --name workers

# Run commands across fleet
gaxx run --name workers --command "echo Processing $(date)"

# List instances
gaxx ls workers

# Clean up
gaxx delete workers
```

### Performance Monitoring

Metrics help validate throughput and latency during parallel runs, making it easier to tune concurrency and spot bottlenecks.
Version output is useful in bug reports and CI logs to ensure predictable behavior across environments.

```bash
# Check metrics
gaxx metrics

# Show version
gaxx version
```

## Features

Gaxx focuses on pragmatic speed, reliability, and observability for short-lived, distributed work.

- Multi-provider support for Linode and Vultr, so fleets can run where it’s most cost-effective or closest to downstream systems.
- High performance execution core with simplified code paths for lower overhead and faster orchestration, enabling large parallel runs with minimal fuss.
- Core engine manages concurrency, SSH execution, and result aggregation at scale.
- Security-first defaults including SSH keys, retry logic, and robust error handling to protect access and improve resilience during flaky network conditions.
- Built-in monitoring for real-time metrics to verify scale, performance, and success rates across a fleet.
- Resilience via automatic retries and connection pooling to keep long or bursty runs stable.

## Architecture

The system is split into a lean CLI, provider integrations, and a high-throughput execution core.

- CLI orchestrates fleet lifecycle and command fan-out with a small, predictable surface area.
- Provider adapters talk to cloud APIs with retry logic and sensible error reporting.
- Core engine manages concurrency, SSH execution, and result aggregation at scale.

## Development

```bash
make build          # Build binaries
make test           # Run tests
make install        # System install
make install-user   # User install
```

## License

Apache-2.0

**Note**: This is a personal project and is provided as-is; use at your own risk.