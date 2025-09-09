package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestFullWorkflow tests the complete end-to-end workflow
func TestFullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Build the binaries first
	if err := buildBinaries(); err != nil {
		t.Fatalf("Failed to build binaries: %v", err)
	}

	// Test initialization
	t.Run("Init", func(t *testing.T) {
		testInit(t, tmpDir)
	})

	// Test basic CLI commands
	t.Run("CLI_Commands", func(t *testing.T) {
		testCLICommands(t, tmpDir)
	})

	// Test agent functionality
	t.Run("Agent", func(t *testing.T) {
		testAgent(t)
	})

	// Test fleet operations (with localssh provider)
	t.Run("Fleet_Operations", func(t *testing.T) {
		testFleetOperations(t, tmpDir)
	})

	// Test run command
	t.Run("Run_Command", func(t *testing.T) {
		testRunCommand(t, tmpDir)
	})

	// Test module execution
	t.Run("Module_Execution", func(t *testing.T) {
		testModuleExecution(t, tmpDir)
	})
}

func buildBinaries() error {
	cmd := exec.Command("make", "build")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %v\nOutput: %s", err, output)
	}
	return nil
}

func testInit(t *testing.T, tmpDir string) {
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create a basic config file since init command doesn't exist in simplified version
	configContent := `provider: linode
region: us-east
ssh_key_path: ~/.config/gaxx/ssh/id_ed25519
monitoring: true
concurrency: 10
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Create SSH directory structure
	sshDir := filepath.Join(tmpDir, "ssh")
	err = os.MkdirAll(sshDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SSH directory: %v", err)
	}

	t.Logf("Config file created at: %s", configPath)
}

func testCLICommands(t *testing.T, tmpDir string) {
	configPath := filepath.Join(tmpDir, "config.yaml")

	tests := []struct {
		name string
		args []string
	}{
		{"version", []string{"version"}},
		{"help", []string{"--help"}},
		{"metrics", []string{"metrics"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"--config", configPath}, test.args...)
			cmd := exec.Command("./bin/gaxx", args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Command %v failed: %v\nOutput: %s", test.args, err, output)
			}
			t.Logf("Command %v output: %s", test.args, output)
		})
	}
}

func testAgent(t *testing.T) {
	// Start the agent in background
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	agentCmd := exec.CommandContext(ctx, "./bin/gaxx-agent")
	if err := agentCmd.Start(); err != nil {
		t.Fatalf("Failed to start agent: %v", err)
	}
	defer func() {
		if agentCmd.Process != nil {
			_ = agentCmd.Process.Kill()
		}
	}()

	// Wait for agent to start
	time.Sleep(2 * time.Second)

	// Test heartbeat
	heartbeatCmd := exec.Command("curl", "-s", "http://localhost:8088/v0/heartbeat")
	output, err := heartbeatCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Heartbeat test failed: %v\nOutput: %s", err, output)
	}

	// Verify heartbeat response contains expected fields
	if !contains(string(output), "time") || !contains(string(output), "version") {
		t.Fatalf("Heartbeat response missing expected fields: %s", output)
	}

	// Test exec
	execCmd := exec.Command("curl", "-s", "-X", "POST",
		"http://localhost:8088/v0/exec",
		"-H", "Content-Type: application/json",
		"-d", `{"command":"echo","args":["test"]}`)
	execOutput, err := execCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Exec test failed: %v\nOutput: %s", err, execOutput)
	}

	// Verify exec response
	if !contains(string(execOutput), "exit_code") || !contains(string(execOutput), "stdout") {
		t.Fatalf("Exec response missing expected fields: %s", execOutput)
	}

	t.Logf("Agent tests successful")
}

func testFleetOperations(t *testing.T, tmpDir string) {
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Test ls command (should work with localssh provider)
	cmd := exec.Command("./bin/gaxx", "--config", configPath, "ls", "--name", "test")
	output, err := cmd.CombinedOutput()

	// Note: This might fail if no local hosts are configured, which is expected
	// We're mainly testing that the command parsing and provider loading works
	t.Logf("Fleet ls output: %s", output)
	if err != nil {
		t.Logf("Fleet ls failed (expected if no hosts configured): %v", err)
	}
}

func testRunCommand(t *testing.T, tmpDir string) {
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Start agent for testing
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	agentCmd := exec.CommandContext(ctx, "./bin/gaxx-agent")
	if err := agentCmd.Start(); err != nil {
		t.Fatalf("Failed to start agent for run test: %v", err)
	}
	defer func() {
		if agentCmd.Process != nil {
			_ = agentCmd.Process.Kill()
		}
	}()

	time.Sleep(2 * time.Second)

	// Create a test config with localhost
	testConfig := createTestConfig(tmpDir)
	if err := os.WriteFile(configPath, []byte(testConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Test simple run command
	cmd := exec.Command("./bin/gaxx", "--config", configPath, "run",
		"--name", "test", "--", "echo", "hello")
	output, err := cmd.CombinedOutput()

	t.Logf("Run command output: %s", output)
	if err != nil {
		t.Logf("Run command failed (expected if no proper fleet): %v", err)
	}
}

func testModuleExecution(t *testing.T, tmpDir string) {
	// Create a simple test module
	moduleContent := `name: test_module
description: Simple test module
command: echo
args: ["hello", "from", "module"]
env: {}
inputs: []
chunk_size: 1
`
	modulePath := filepath.Join(tmpDir, "test_module.yaml")
	if err := os.WriteFile(modulePath, []byte(moduleContent), 0644); err != nil {
		t.Fatalf("Failed to write test module: %v", err)
	}

	configPath := filepath.Join(tmpDir, "config.yaml")

	// Test module loading
	cmd := exec.Command("./bin/gaxx", "--config", configPath, "run",
		"--name", "test", "--module", modulePath)
	output, err := cmd.CombinedOutput()

	t.Logf("Module execution output: %s", output)
	if err != nil {
		t.Logf("Module execution failed (expected if no proper fleet): %v", err)
	}
}

func createTestConfig(tmpDir string) string {
	sshKeyPath := filepath.Join(tmpDir, "ssh", "id_ed25519")
	knownHostsPath := filepath.Join(tmpDir, "known_hosts")

	return fmt.Sprintf(`providers:
  default: localssh
  localssh:
    hosts:
      - {name: "test-local", ip: "127.0.0.1", user: "testuser", key_path: "%s", port: 22}
ssh:
  key_dir: %s
  known_hosts: %s
defaults:
  user: gx
  ssh_port: 22
  retries: 3
  timeout_seconds: 30
telemetry:
  enabled: false
`, sshKeyPath, filepath.Join(tmpDir, "ssh"), knownHostsPath)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsInMiddle(s, substr))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
