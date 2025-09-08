# Security posture for gaxx

Gaxx is built with secure-by-default assumptions for remote execution:

- SSH key-only authentication
- `PermitRootLogin no`, `PasswordAuthentication no`
- Non-root user `gx` for agent operations
- Strict `known_hosts` management
- Reasonable timeouts and retries with backoff
- Optional `fail2ban` via cloud-init when provisioning cloud instances

Operational guidance:
- Rotate ephemeral SSH keys per fleet.
- Restrict network access to SSH from controller IPs.
- Store tokens via your OS keychain or environment variables, not in plaintext.
- Keep agents updated using signed releases.


