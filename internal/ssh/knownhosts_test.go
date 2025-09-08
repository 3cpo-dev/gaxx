package ssh

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKnownHostsAppend(t *testing.T) {
	dir := t.TempDir()
	kh := filepath.Join(dir, "known_hosts")
	// Use a dummy public key line (ed25519 for localhost). We'll generate a real one from keygen.
	priv := filepath.Join(dir, "id_ed25519")
	pub, err := GenerateEd25519Keypair(priv)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	if err := AppendKnownHost(kh, "example.com", pub); err != nil {
		t.Fatalf("append known host: %v", err)
	}
	b, err := os.ReadFile(kh)
	if err != nil {
		t.Fatalf("read known_hosts: %v", err)
	}
	if len(b) == 0 {
		t.Fatalf("expected content in known_hosts")
	}
}
