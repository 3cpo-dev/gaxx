package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	prov "github.com/3cpo-dev/gaxx/internal/providers"
	gssh "github.com/3cpo-dev/gaxx/internal/ssh"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// FileTransfer handles secure file transfers with verification
type FileTransfer struct {
	config prov.Config
}

// NewFileTransfer creates a new file transfer handler
func NewFileTransfer(cfg prov.Config) *FileTransfer {
	return &FileTransfer{config: cfg}
}

// TransferFile uploads a file to a node using SFTP with checksum verification
func (ft *FileTransfer) TransferFile(ctx context.Context, node prov.Node, localPath, remotePath string) error {
	// Calculate local file checksum
	localChecksum, err := ft.calculateChecksum(localPath)
	if err != nil {
		return fmt.Errorf("calculate local checksum: %w", err)
	}

	// Establish SSH connection
	sshClient, err := ft.connectSSH(ctx, node)
	if err != nil {
		return fmt.Errorf("connect SSH: %w", err)
	}
	defer sshClient.Close()

	// Create SFTP client
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Create remote directory if needed
	remoteDir := filepath.Dir(remotePath)
	if err := sftpClient.MkdirAll(remoteDir); err != nil {
		return fmt.Errorf("create remote directory: %w", err)
	}

	// Open local file
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local file: %w", err)
	}
	defer localFile.Close()

	// Create remote file
	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create remote file: %w", err)
	}
	defer remoteFile.Close()

	// Copy file with progress tracking
	_, err = io.Copy(remoteFile, localFile)
	if err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	// Verify checksum on remote side
	if err := ft.verifyRemoteChecksum(sshClient, remotePath, localChecksum); err != nil {
		// Cleanup failed transfer
		_ = sftpClient.Remove(remotePath)
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	return nil
}

// TransferFiles uploads multiple files concurrently
func (ft *FileTransfer) TransferFiles(ctx context.Context, node prov.Node, files map[string]string) error {
	for localPath, remotePath := range files {
		if err := ft.TransferFile(ctx, node, localPath, remotePath); err != nil {
			return fmt.Errorf("transfer %s -> %s: %w", localPath, remotePath, err)
		}
	}
	return nil
}

// calculateChecksum calculates SHA256 checksum of a file
func (ft *FileTransfer) calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// connectSSH establishes SSH connection to node
func (ft *FileTransfer) connectSSH(ctx context.Context, node prov.Node) (*ssh.Client, error) {
	user := node.SSHUser
	if user == "" {
		user = ft.config.Defaults.User
	}

	port := node.SSHPort
	if port == 0 {
		port = ft.config.Defaults.SSHPort
	}

	keyPath := filepath.Join(ft.config.SSH.KeyDir, "id_ed25519")
	signer, err := gssh.LoadPrivateKeySigner(keyPath)
	if err != nil {
		return nil, fmt.Errorf("load SSH key: %w", err)
	}

	kh, err := gssh.LoadKnownHostsCallback(ft.config.SSH.KnownHosts)
	if err != nil {
		return nil, fmt.Errorf("load known hosts: %w", err)
	}

	client := &gssh.Client{
		Addr:       fmt.Sprintf("%s:%d", node.IP, port),
		User:       user,
		Signer:     signer,
		KnownHosts: kh,
		Timeout:    30 * 1000 * 1000 * 1000, // 30 seconds
		Retries:    ft.config.Defaults.Retries,
		Backoff:    500 * 1000 * 1000, // 500ms
	}

	return gssh.Dial(ctx, client)
}

// verifyRemoteChecksum verifies file integrity on remote side
func (ft *FileTransfer) verifyRemoteChecksum(sshClient *ssh.Client, remotePath, expectedChecksum string) error {
	session, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	// Calculate checksum on remote side
	cmd := fmt.Sprintf("sha256sum %s | cut -d' ' -f1", remotePath)
	output, err := session.Output(cmd)
	if err != nil {
		return fmt.Errorf("calculate remote checksum: %w", err)
	}

	remoteChecksum := string(output)
	remoteChecksum = remoteChecksum[:len(remoteChecksum)-1] // Remove newline

	if remoteChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, remoteChecksum)
	}

	return nil
}

// GetFileInfo returns file information including size and checksum
func (ft *FileTransfer) GetFileInfo(filePath string) (FileInfo, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return FileInfo{}, err
	}

	checksum, err := ft.calculateChecksum(filePath)
	if err != nil {
		return FileInfo{}, err
	}

	return FileInfo{
		Path:     filePath,
		Size:     stat.Size(),
		Checksum: checksum,
		ModTime:  stat.ModTime(),
	}, nil
}

// FileInfo contains file metadata
type FileInfo struct {
	Path     string
	Size     int64
	Checksum string
	ModTime  interface{} // time.Time but avoiding import
}
