# Security Guide

Gaxx implements secure-by-default distributed execution with enterprise hardening options.

## Default Security

- **SSH Keys**: Ed25519 keys, no passwords
- **Non-root**: Default user `gx` with minimal privileges  
- **Host Verification**: Strict `known_hosts` management
- **Network Isolation**: Agent APIs require network restrictions

## Network Endpoints

### Agent (`gaxx-agent`)
- **API**: `:8088` - Command execution and health
- **Monitoring**: `:9091` - Metrics and dashboard
- **Profiling**: `:6060` - Performance analysis

### CLI (`gaxx`)
- **Monitoring**: `:9090` - Local metrics collection

## Production Hardening

### 1. Agent Authentication
```bash
# Bearer token
export GAXX_AGENT_TOKEN="your-secret-token"

# mTLS certificates
export GAXX_AGENT_TLS_CERT=server.pem
export GAXX_AGENT_TLS_KEY=server.key
export GAXX_AGENT_CLIENT_CA=client-ca.pem
export GAXX_AGENT_REQUIRE_MTLS=true
```

### 2. Network Security
```bash
# Firewall rules (example)
ufw allow from 10.0.0.0/8 to any port 8088  # Agent API
ufw allow from 10.0.0.0/8 to any port 9091  # Monitoring
ufw deny 6060  # Block profiling in production
```

### 3. SSH Hardening
```bash
# /etc/ssh/sshd_config
PermitRootLogin no
PasswordAuthentication no
PubkeyAuthentication yes
AuthorizedKeysFile .ssh/authorized_keys
```

## Secrets Management

### Environment Variables
```bash
# Cloud provider credentials
LINODE_TOKEN=your-token
VULTR_API_KEY=your-key

# Agent security
GAXX_AGENT_TOKEN=your-secret
GAXX_AGENT_TLS_CERT=/path/to/cert
```

### File Permissions
```bash
chmod 600 ~/.config/gaxx/secrets.env
chmod 600 ~/.config/gaxx/ssh/id_ed25519
chmod 644 ~/.config/gaxx/ssh/id_ed25519.pub
```

## Security Checklist

- [ ] Use mTLS for agent communication
- [ ] Set strong bearer tokens
- [ ] Configure firewall rules
- [ ] Restrict SSH access
- [ ] Monitor agent endpoints
- [ ] Rotate credentials regularly
- [ ] Enable audit logging
- [ ] Test security configuration

## Vulnerability Reporting

Report security issues privately. I aim to respond within 48 hours and provide fixes within 7 days for critical issues.

## Best Practices

1. **Network Isolation**: Use VPCs and security groups
2. **Least Privilege**: Minimal required permissions
3. **Monitoring**: Enable telemetry and alerting
4. **Updates**: Keep dependencies current
5. **Backup**: Secure configuration backups