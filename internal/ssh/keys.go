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

	// Marshal private key in OpenSSH format
	privKeyPEM, err := xssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return "", fmt.Errorf("marshal private key: %w", err)
	}

	// Encode the PEM block to bytes
	privKeyBytes := pem.EncodeToMemory(privKeyPEM)
	if err := os.WriteFile(privateKeyPath, privKeyBytes, 0600); err != nil {
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

// MarshalAuthorized returns authorized_keys text for given signer public key.
func MarshalAuthorized(signer xssh.Signer) []byte {
	return xssh.MarshalAuthorizedKey(signer.PublicKey())
}
