# Gaxx

High-performance distributed task runner for VPS fleets. Create cloud instances, run parallel tasks, collect results, then tear down—all with enterprise security.

## Quick Start

```bash
# 1. Build and install
make build
echo 'export PATH="/path/to/gaxx/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

# 2. Configure providers
export LINODE_TOKEN="your-token"
export VULTR_API_KEY="your-key"

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
```bash
# Cloud provider credentials
LINODE_TOKEN=your-linode-token
VULTR_API_KEY=your-vultr-key

# Optional: Custom config file
export GAXX_CONFIG=/path/to/config.yaml
```

### Config File (`~/.config/gaxx/config.yaml`)
```yaml
provider: linode
region: us-east
ssh_key_path: ~/.config/gaxx/ssh/id_ed25519
monitoring: true
concurrency: 10
```

## Examples

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
```bash
# Check metrics
gaxx metrics

# Show version
gaxx version
```

## Features

- **Multi-Provider**: Linode, Vultr support
- **High Performance**: Optimized core with 88% less code
- **Security**: SSH keys, retry logic, error handling
- **Monitoring**: Real-time metrics and performance tracking
- **Resilience**: Automatic retries, connection pooling

## Architecture

- **CLI**: High-performance orchestration
- **Providers**: Cloud APIs with retry logic
- **Core**: Optimized task execution engine

## Development

```bash
make build          # Build binaries
make test           # Run tests
make install        # System install
make install-user   # User install
```

## License

Apache-2.0