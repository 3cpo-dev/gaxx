package ssh

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateEd25519Keypair(t *testing.T) {
	dir := t.TempDir()
	priv := filepath.Join(dir, "id_ed25519")
	pub, err := GenerateEd25519Keypair(priv)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if _, err := os.Stat(priv); err != nil {
		t.Fatalf("private key not written: %v", err)
	}
	if len(pub) == 0 {
		t.Fatalf("expected public key string")
	}
}
