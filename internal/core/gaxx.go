package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// Config represents the simplified configuration
type Config struct {
	Provider    string `yaml:"provider"`
	Token       string `yaml:"token"`
	Region      string `yaml:"region"`
	SSHKeyPath  string `yaml:"ssh_key_path"`
	Monitoring  bool   `yaml:"monitoring"`
	Concurrency int    `yaml:"concurrency"`
}

// Instance represents a cloud instance
type Instance struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	IP   string `json:"ip"`
	User string `json:"user"`
	Port int    `json:"port"`
}

// Task represents a task to execute
type Task struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
	Input   string            `json:"input"`
}

// Provider interface for cloud providers
type Provider interface {
	CreateInstances(ctx context.Context, count int, name string) ([]Instance, error)
	DeleteInstances(ctx context.Context, name string) error
	ListInstances(ctx context.Context, name string) ([]Instance, error)
}

// SSHClient handles SSH operations
type SSHClient struct {
	keyPath string
	timeout time.Duration
	client  *ssh.Client
}

// NewSSHClient creates a new SSH client
func NewSSHClient(keyPath string) *SSHClient {
	return &SSHClient{
		keyPath: keyPath,
		timeout: 30 * time.Second,
	}
}

// Execute runs a command on a remote host
func (s *SSHClient) Execute(host string, cmd string) (string, error) {
	config := &ssh.ClientConfig{
		User: "gx",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(s.loadKey()),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Implement proper host key verification
		Timeout:         s.timeout,
	}

	client, err := ssh.Dial("tcp", host+":22", config)
	if err != nil {
		return "", fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("ssh session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	return string(output), err
}

// Upload uploads a file to a remote host
func (s *SSHClient) Upload(host string, localPath, remotePath string) error {
	// TODO: Implement SFTP upload with checksum verification
	return fmt.Errorf("upload not implemented yet")
}

// loadKey loads the SSH private key
func (s *SSHClient) loadKey() ssh.Signer {
	key, err := os.ReadFile(s.keyPath)
	if err != nil {
		panic(fmt.Sprintf("failed to read SSH key: %v", err))
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		panic(fmt.Sprintf("failed to parse SSH key: %v", err))
	}

	return signer
}

// Metrics tracks basic performance metrics
type Metrics struct {
	requests int64
	errors   int64
	duration time.Duration
	mu       sync.RWMutex
}

// NewMetrics creates a new metrics tracker
func NewMetrics() *Metrics {
	return &Metrics{}
}

// RecordRequest records a successful request
func (m *Metrics) RecordRequest(duration time.Duration) {
	m.mu.Lock()
	m.requests++
	m.duration += duration
	m.mu.Unlock()
}

// RecordError records an error
func (m *Metrics) RecordError() {
	m.mu.Lock()
	m.errors++
	m.mu.Unlock()
}

// GetStats returns current metrics
func (m *Metrics) GetStats() (int64, int64, time.Duration) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.requests, m.errors, m.duration
}

// Gaxx is the main simplified orchestrator
type Gaxx struct {
	config   *Config
	provider Provider
	ssh      *SSHClient
	metrics  *Metrics
}

// NewGaxx creates a new Gaxx instance
func NewGaxx(config *Config, provider Provider) *Gaxx {
	return &Gaxx{
		config:   config,
		provider: provider,
		ssh:      NewSSHClient(config.SSHKeyPath),
		metrics:  NewMetrics(),
	}
}

// SpawnFleet creates a fleet of instances
func (g *Gaxx) SpawnFleet(ctx context.Context, name string, count int) ([]Instance, error) {
	start := time.Now()
	defer func() {
		g.metrics.RecordRequest(time.Since(start))
	}()

	instances, err := g.provider.CreateInstances(ctx, count, name)
	if err != nil {
		g.metrics.RecordError()
		return nil, fmt.Errorf("create instances: %w", err)
	}

	// Wait for instances to be ready
	for _, instance := range instances {
		if err := g.WaitForInstance(ctx, instance); err != nil {
			g.metrics.RecordError()
			return nil, fmt.Errorf("instance %s not ready: %w", instance.ID, err)
		}
	}

	return instances, nil
}

// ExecuteTasks runs tasks across instances with controlled concurrency
func (g *Gaxx) ExecuteTasks(ctx context.Context, instances []Instance, tasks []Task) error {
	start := time.Now()
	defer func() {
		g.metrics.RecordRequest(time.Since(start))
	}()

	sem := make(chan struct{}, g.config.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for _, task := range tasks {
		for _, instance := range instances {
			wg.Add(1)
			go func(inst Instance, t Task) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				cmd := g.BuildCommand(t)
				output, err := g.ssh.Execute(inst.IP, cmd)

				if err != nil {
					g.metrics.RecordError()
					mu.Lock()
					errors = append(errors, fmt.Errorf("instance %s: %w", inst.ID, err))
					mu.Unlock()
				} else {
					fmt.Printf("[%s] %s\n", inst.Name, output)
				}
			}(instance, task)
		}
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("task execution failed: %v", errors)
	}
	return nil
}

// DeleteFleet removes all instances
func (g *Gaxx) DeleteFleet(ctx context.Context, name string) error {
	start := time.Now()
	defer func() {
		g.metrics.RecordRequest(time.Since(start))
	}()

	if err := g.provider.DeleteInstances(ctx, name); err != nil {
		g.metrics.RecordError()
		return fmt.Errorf("delete instances: %w", err)
	}
	return nil
}

// ListInstances returns current instances
func (g *Gaxx) ListInstances(ctx context.Context, name string) ([]Instance, error) {
	start := time.Now()
	defer func() {
		g.metrics.RecordRequest(time.Since(start))
	}()

	instances, err := g.provider.ListInstances(ctx, name)
	if err != nil {
		g.metrics.RecordError()
		return nil, fmt.Errorf("list instances: %w", err)
	}
	return instances, nil
}

// GetMetrics returns current performance metrics
func (g *Gaxx) GetMetrics() (int64, int64, time.Duration) {
	return g.metrics.GetStats()
}

// WaitForInstance waits for an instance to be ready (exported for testing)
func (g *Gaxx) WaitForInstance(ctx context.Context, instance Instance) error {
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for instance")
		case <-ticker.C:
			_, err := g.ssh.Execute(instance.IP, "echo ready")
			if err == nil {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// BuildCommand constructs the command string from a task (exported for testing)
func (g *Gaxx) BuildCommand(task Task) string {
	cmd := task.Command
	for _, arg := range task.Args {
		cmd += " " + arg
	}
	return cmd
}

// LoadConfig loads configuration from file or environment
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = filepath.Join(os.Getenv("HOME"), ".config", "gaxx", "config.yaml")
	}

	// For now, return a default config
	// TODO: Implement proper YAML loading
	return &Config{
		Provider:    "linode",
		Token:       os.Getenv("LINODE_TOKEN"),
		Region:      "us-east",
		SSHKeyPath:  filepath.Join(os.Getenv("HOME"), ".config", "gaxx", "ssh", "id_ed25519"),
		Monitoring:  true,
		Concurrency: 10,
	}, nil
}
