# Security Guide

Gaxx aims for secure-by-default distributed execution with options to harden for production environments following least-privilege and network-segmentation principles.

## Default Security

- SSH keys use Ed25519 by default with password logins disabled for stronger, faster key-based authentication.
- Non-root execution with a minimal-privilege user aligns with the principle of least privilege for reduced blast radius.
- Strict host verification enforces known_hosts checks to prevent man-in-the-middle attacks.
- Network isolation is expected; restrict agent endpoints to trusted CIDRs/VPCs and avoid broad public exposure.

## Network Endpoints

### Agent (`gaxx-agent`)
- API: `:8088` - for command/health, Monitoring :9091 for metrics, Profiling :6060 for pprof-based analysis.
- Monitoring: `:9091` - Metrics and dashboard
- Profiling: `:6060` - Performance analysis

### CLI (`gaxx`)
- CLI (gaxx): Local metrics on :9090 follows Prometheus-port conventions for local scraping and dashboards

## Production Hardening

### 1. Agent Authentication

Require mTLS and use strong bearer tokens in production to authenticate both client and server before any command execution.

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

Limit ingress to trusted ranges and block profiling externally; treat monitoring ports as restricted infrastructure interfaces, not public endpoints.

```bash
# Firewall rules (example)
ufw allow from 10.0.0.0/8 to any port 8088  # Agent API
ufw allow from 10.0.0.0/8 to any port 9091  # Monitoring
ufw deny 6060  # Block profiling in production
```

### 3. SSH Hardening

Disable root and password auth and enforce key-only access to reduce credential and privilege escalation risks.

```bash
# /etc/ssh/sshd_config
PermitRootLogin no
PasswordAuthentication no
PubkeyAuthentication yes
AuthorizedKeysFile .ssh/authorized_keys
```

## Secrets Management

Use environment variables or a secrets manager for credentials and never commit secrets to version control, applying least-privilege scopes where possible.

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

Set strict file permissions so SSH does not reject keys and sensitive files due to being “too open”.

```bash
chmod 600 ~/.config/gaxx/secrets.env
chmod 600 ~/.config/gaxx/ssh/id_ed25519
chmod 644 ~/.config/gaxx/ssh/id_ed25519.pub
```

## Security Checklist

- [ ] Use mTLS and strong tokens for agent communications.
- [ ] Configure firewalls and restrict ports to trusted networks only.
- [ ] Enforce key-only SSH and rotate credentials periodically.
- [ ] Monitor and alert on agent/metrics endpoints and dashboards.
- [ ] Enable audit logging and regularly test security controls and recovery.
- [ ] Rotate credentials regularly.

## Best Practices

1. **Network Isolation**: Use VPCs and security groups
2. **Least Privilege**: Minimal required permissions
3. **Monitoring**: Enable telemetry and alerting
4. **Updates**: Keep dependencies current
5. **Backup**: Secure configuration backups

**Note**: This is a personal project and is provided as-is; use at your own risk.