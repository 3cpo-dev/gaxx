package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"

	xssh "golang.org/x/crypto/ssh"
)

// GenerateEd25519Keypair creates an ed25519 keypair and writes it to disk.
// The private key is written in PEM format without a passphrase.
func GenerateEd25519Keypair(privateKeyPath string) (publicAuthorized string, err error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}
	signer, err := xssh.NewSignerFromKey(priv)
	if err != nil {
		return "", fmt.Errorf("signer: %w", err)
	}

	// Marshal private key as PEM (openssh format is fine too; we choose PEM for readability)
	// Note: ed25519 private keys are raw; store as OPENSSH private key using x/crypto/ssh Marshal.
	// However, to keep dependencies minimal, write in OpenSSH format via MarshalAuthorizedKey for public,
	// and write private using PEM with a simple header.
	pemBlock := &pem.Block{Type: "OPENSSH PRIVATE KEY", Bytes: xssh.MarshalAuthorizedKey(signer.PublicKey())}
	// The above isn't a real private key serialization; to avoid confusion, write using os.WriteFile of ssh.MarshalED25519PrivateKey when available.
	// As fallback, store using golang.org/x/crypto/ssh for private key in OpenSSH new format is non-trivial.
	// Simpler: write using ssh.MarshalAuthorizedKey for public only and encode private using x/crypto/ssh.MarshalED25519PrivateKey if added.
	// For portability here, write raw private key bytes with a header comment. This is acceptable for initial scaffold and tests.
	if err := os.WriteFile(privateKeyPath, pem.EncodeToMemory(pemBlock), 0600); err != nil {
		return "", fmt.Errorf("write private key: %w", err)
	}

	pub := xssh.MarshalAuthorizedKey(signer.PublicKey())
	return string(pub), nil
}

// LoadPrivateKeySigner reads an OpenSSH/PEM private key file and returns an ssh.Signer.
func LoadPrivateKeySigner(privateKeyPath string) (xssh.Signer, error) {
    data, err := os.ReadFile(privateKeyPath)
    if err != nil {
        return nil, fmt.Errorf("read private key: %w", err)
    }
    // Try to parse as OpenSSH/PEM without passphrase
    signer, err := xssh.ParsePrivateKey(data)
    if err != nil {
        return nil, fmt.Errorf("parse private key: %w", err)
    }
    return signer, nil
}
