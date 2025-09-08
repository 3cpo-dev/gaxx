package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	xssh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// EnsureKnownHostsFile makes sure the directory exists and the file is created.
func EnsureKnownHostsFile(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("mkdir known_hosts dir: %w", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(""), 0600); err != nil {
			return fmt.Errorf("create known_hosts: %w", err)
		}
	}
	return nil
}

// AppendKnownHost appends a known_hosts entry for host using the given authorized key text.
func AppendKnownHost(path, host, authorizedKey string) error {
	if err := EnsureKnownHostsFile(path); err != nil {
		return err
	}
	pubKey, _, _, _, err := xssh.ParseAuthorizedKey([]byte(strings.TrimSpace(authorizedKey)))
	if err != nil {
		return fmt.Errorf("parse authorized key: %w", err)
	}
	line := knownhosts.Line([]string{host}, pubKey)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open known_hosts: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(line + "\n"); err != nil {
		return fmt.Errorf("write known_hosts: %w", err)
	}
	return nil
}

// LoadKnownHostsCallback returns a strict host key callback using the given file.
func LoadKnownHostsCallback(path string) (xssh.HostKeyCallback, error) {
    if err := EnsureKnownHostsFile(path); err != nil {
        return nil, err
    }
    return knownhosts.New(path)
}
