package providers

import (
	"fmt"
)

// CloudInitUserData returns a minimal cloud-init YAML that:
// - creates a non-root user
// - configures SSH hardening
// - writes the controller's ephemeral SSH public key
// - installs and starts gaxx-agent via a simple systemd unit
func CloudInitUserData(username, sshAuthorizedKey, agentDownloadURL string) string {
	if username == "" {
		username = "gx"
	}
	return fmt.Sprintf(`#cloud-config
users:
  - name: %s
    sudo: ["ALL=(ALL) NOPASSWD:ALL"]
    shell: /bin/bash
    ssh_authorized_keys:
      - %s
ssh_pwauth: false
disable_root: true
package_update: true
package_upgrade: true
write_files:
  - path: /etc/ssh/sshd_config.d/99-gaxx.conf
    permissions: '0644'
    content: |
      PermitRootLogin no
      PasswordAuthentication no
      ChallengeResponseAuthentication no
      UsePAM yes
runcmd:
  - |
    set -euo pipefail
    cd /tmp
    curl -fsSL %s -o gaxx-agent
    install -m 0755 gaxx-agent /usr/local/bin/gaxx-agent
    printf '[Unit]\nDescription=Gaxx Agent\nAfter=network.target\n[Service]\nExecStart=/usr/local/bin/gaxx-agent\nUser=%s\nRestart=always\nRestartSec=2\n[Install]\nWantedBy=multi-user.target\n' > /etc/systemd/system/gaxx-agent.service
    systemctl daemon-reload
    systemctl enable --now gaxx-agent
`, username, sshAuthorizedKey, agentDownloadURL, username)
}
