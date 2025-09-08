package ssh

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
	xssh "golang.org/x/crypto/ssh"
)

// PushFile uploads a local file to a remote path via SFTP.
func PushFile(ctx context.Context, client *xssh.Client, localPath, remotePath string) error {
	sf, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("sftp client: %w", err)
	}
	defer sf.Close()
	// Ensure remote directory exists
	if err := sf.MkdirAll(filepath.Dir(remotePath)); err != nil {
		return fmt.Errorf("mkdir remote: %w", err)
	}
	src, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local: %w", err)
	}
	defer src.Close()
	dst, err := sf.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create remote: %w", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return nil
}

// PullFile downloads a remote file to a local path via SFTP.
func PullFile(ctx context.Context, client *xssh.Client, remotePath, localPath string) error {
    sf, err := sftp.NewClient(client)
    if err != nil {
        return fmt.Errorf("sftp client: %w", err)
    }
    defer sf.Close()
    // Ensure local directory exists
    if err := os.MkdirAll(filepath.Dir(localPath), 0700); err != nil {
        return fmt.Errorf("mkdir local: %w", err)
    }
    src, err := sf.Open(remotePath)
    if err != nil {
        return fmt.Errorf("open remote: %w", err)
    }
    defer src.Close()
    dst, err := os.Create(localPath)
    if err != nil {
        return fmt.Errorf("create local: %w", err)
    }
    defer dst.Close()
    if _, err := io.Copy(dst, src); err != nil {
        return fmt.Errorf("copy: %w", err)
    }
    return nil
}
