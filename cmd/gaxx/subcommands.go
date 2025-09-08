package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/3cpo-dev/gaxx/internal/agent"
	core "github.com/3cpo-dev/gaxx/internal/core"
	prov "github.com/3cpo-dev/gaxx/internal/providers"
	lin "github.com/3cpo-dev/gaxx/internal/providers/linode"
	localssh "github.com/3cpo-dev/gaxx/internal/providers/localssh"
	vlt "github.com/3cpo-dev/gaxx/internal/providers/vultr"
	gssh "github.com/3cpo-dev/gaxx/internal/ssh"
	"github.com/3cpo-dev/gaxx/internal/telemetry"
	"github.com/3cpo-dev/gaxx/pkg/api"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Resolve the registry
func resolveRegistry(cmd *cobra.Command) (*prov.Registry, coreConfig, error) {
	cfgPath, _ := cmd.Flags().GetString("config")
	cfg, err := core.LoadConfig(cfgPath)
	if err != nil {
		return nil, coreConfig{}, err
	}
	reg := prov.NewRegistry()
	reg.Register(localssh.New(cfg))
	reg.Register(lin.New(cfg))
	reg.Register(vlt.New(cfg))
	return reg, coreConfig{cfg: cfg}, nil
}

type coreConfig struct{ cfg prov.Config }

// Spawn a fleet
func newSpawnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spawn",
		Short: "Spawn a fleet or even a single box",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			count, _ := cmd.Flags().GetInt("count")
			provider, _ := cmd.Flags().GetString("provider")
			region, _ := cmd.Flags().GetString("region")
			image, _ := cmd.Flags().GetString("image")
			size, _ := cmd.Flags().GetString("size")
			reg, cc, err := resolveRegistry(cmd)
			if err != nil {
				return err
			}
			if provider == "" {
				provider = cc.cfg.Providers.Default
			}
			p, err := reg.Get(provider)
			if err != nil {
				return err
			}
			fleet, err := p.CreateFleet(cmd.Context(), prov.CreateFleetRequest{Name: name, Count: count, Region: region, Image: image, Size: size})
			if err != nil {
				return err
			}
			fmt.Printf("spawned fleet %s with %d nodes\n", fleet.Name, len(fleet.Nodes))
			return nil
		},
	}
	cmd.Flags().String("name", "", "fleet name")
	cmd.Flags().Int("count", 1, "number of instances")
	cmd.Flags().String("provider", "", "provider name")
	cmd.Flags().String("region", "", "region/zone id (provider-specific)")
	cmd.Flags().String("image", "", "image/os id (provider-specific)")
	cmd.Flags().String("size", "", "plan/size/type (provider-specific)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

// List running boxes
func newLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List running boxes",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			provider, _ := cmd.Flags().GetString("provider")
			reg, cc, err := resolveRegistry(cmd)
			if err != nil {
				return err
			}
			if provider == "" {
				provider = cc.cfg.Providers.Default
			}
			p, err := reg.Get(provider)
			if err != nil {
				return err
			}
			nodes, err := p.ListNodes(cmd.Context(), name)
			if err != nil {
				return err
			}
			for _, n := range nodes {
				fmt.Printf("%s\t%s\t%s\n", n.Name, n.IP, n.ID)
			}
			return nil
		},
	}
}

// Delete resources
func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete",
		Short: "Delete an existing fleet or even a single box",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			provider, _ := cmd.Flags().GetString("provider")
			reg, cc, err := resolveRegistry(cmd)
			if err != nil {
				return err
			}
			if provider == "" {
				provider = cc.cfg.Providers.Default
			}
			p, err := reg.Get(provider)
			if err != nil {
				return err
			}
			if err := p.DeleteFleet(cmd.Context(), name); err != nil {
				return err
			}
			fmt.Printf("deleted %s\n", name)
			return nil
		},
	}
}

// Run a command on a fleet
func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Send a command to a fleet",
		Long: `Execute a command across all nodes in a fleet.

Examples:
  # Run a simple command
  gaxx run --name myfleet -- echo "hello world"
  
  # Run a module
  gaxx run --name myfleet --module port_scan.yaml --inputs targets.txt
  
  # Run with environment variables
  gaxx run --name myfleet --env ports=22,80,443 -- nmap -p $ports`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFleetCommand(cmd, args)
		},
	}
	cmd.Flags().String("name", "", "Fleet name (required)")
	cmd.Flags().String("provider", "", "Provider name (defaults to config)")
	cmd.Flags().String("module", "", "YAML module file to execute")
	cmd.Flags().StringArray("inputs", nil, "Input files for module execution")
	cmd.Flags().StringToString("env", nil, "Environment variables (key=value)")
	cmd.Flags().Int("timeout", 300, "Command timeout in seconds")
	cmd.Flags().Int("concurrency", 0, "Max concurrent executions (0 = all nodes)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

// Copy files to/from fleet
func newScpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scp",
		Short: "Send a file/folder to a fleet using SCP",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			provider, _ := cmd.Flags().GetString("provider")
			push, _ := cmd.Flags().GetStringSlice("push")
			pull, _ := cmd.Flags().GetStringSlice("pull")
			reg, cc, err := resolveRegistry(cmd)
			if err != nil {
				return err
			}
			if provider == "" {
				provider = cc.cfg.Providers.Default
			}
			p, err := reg.Get(provider)
			if err != nil {
				return err
			}
			nodes, err := p.ListNodes(cmd.Context(), name)
			if err != nil {
				return err
			}
			if len(nodes) == 0 {
				return fmt.Errorf("no nodes found for fleet %s", name)
			}
			node := nodes[0]
			signer, err := gssh.LoadPrivateKeySigner(cc.cfg.SSH.KeyDir + "/id_ed25519")
			if err != nil {
				return err
			}
			kh, _ := gssh.LoadKnownHostsCallback(cc.cfg.SSH.KnownHosts)
			c := &gssh.Client{Addr: fmt.Sprintf("%s:%d", node.IP, node.SSHPort), User: node.SSHUser, Signer: signer, KnownHosts: kh, Timeout: 15 * time.Second, Retries: 2, Backoff: 500 * time.Millisecond}
			cli, err := gssh.Dial(cmd.Context(), c)
			if err != nil {
				return err
			}
			defer cli.Close()
			for _, spec := range push {
				parts := strings.SplitN(spec, ":", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid --push spec: %s", spec)
				}
				if err := gssh.PushFile(cmd.Context(), cli, parts[0], parts[1]); err != nil {
					return err
				}
			}
			for _, spec := range pull {
				parts := strings.SplitN(spec, ":", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid --pull spec: %s", spec)
				}
				if err := gssh.PullFile(cmd.Context(), cli, parts[0], parts[1]); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().String("name", "", "fleet name")
	cmd.Flags().String("provider", "", "provider name")
	cmd.Flags().StringSlice("push", nil, "local:remote specs to upload via SFTP")
	cmd.Flags().StringSlice("pull", nil, "remote:local specs to download via SFTP")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

// Open SSH to a node
func newSSHCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh",
		Short: "Start SSH terminal for a box",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			provider, _ := cmd.Flags().GetString("provider")
			nodeName, _ := cmd.Flags().GetString("node")
			reg, cc, err := resolveRegistry(cmd)
			if err != nil {
				return err
			}
			if provider == "" {
				provider = cc.cfg.Providers.Default
			}
			p, err := reg.Get(provider)
			if err != nil {
				return err
			}
			nodes, err := p.ListNodes(cmd.Context(), name)
			if err != nil {
				return err
			}
			var node prov.Node
			if nodeName == "" && len(nodes) > 0 {
				node = nodes[0]
			} else {
				for _, n := range nodes {
					if n.Name == nodeName {
						node = n
						break
					}
				}
			}
			if node.Name == "" {
				return fmt.Errorf("node not found")
			}
			signer, err := gssh.LoadPrivateKeySigner(cc.cfg.SSH.KeyDir + "/id_ed25519")
			if err != nil {
				return err
			}
			kh, _ := gssh.LoadKnownHostsCallback(cc.cfg.SSH.KnownHosts)
			c := &gssh.Client{Addr: fmt.Sprintf("%s:%d", node.IP, node.SSHPort), User: node.SSHUser, Signer: signer, KnownHosts: kh, Timeout: 15 * time.Second}
			stdout, _, err := c.RunCommand(cmd.Context(), "uname -a")
			if err != nil {
				return err
			}
			fmt.Println(strings.TrimSpace(stdout))
			return nil
		},
	}
	cmd.Flags().String("name", "", "fleet name")
	cmd.Flags().String("provider", "", "provider name")
	cmd.Flags().String("node", "", "node name (defaults to first node)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

// Show image options (stub)
func newImagesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "images",
		Short: "Show image options",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("images not yet implemented")
			return nil
		},
	}
}

// Initialize configuration and environment
func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "gaxx initialization command. Run this the first time.",
		Long: `Initialize Gaxx configuration and environment.

This command will:
- Create a default configuration file
- Generate SSH keys if needed
- Set up known_hosts file
- Create necessary directories`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return initializeGaxx(cmd)
		},
	}
	cmd.Flags().Bool("force", false, "Overwrite existing configuration")
	return cmd
}

// Scan command with file upload and chunking
func newScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Send a command to a fleet, but also with files upload & chunks splitting",
		Long: `Execute a task with file upload and intelligent chunking across fleet nodes.

Examples:
  # Upload and scan a wordlist
  gaxx scan --name myfleet --module dns_bruteforce.yaml --upload wordlist.txt --inputs wordlist.txt --env domain=example.com
  
  # Upload multiple files and run port scan
  gaxx scan --name myfleet --module port_scan.yaml --upload targets.txt --inputs targets.txt --env ports=80,443,8080`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScanCommand(cmd, args)
		},
	}
	cmd.Flags().String("name", "", "Fleet name (required)")
	cmd.Flags().String("provider", "", "Provider name (defaults to config)")
	cmd.Flags().String("module", "", "YAML module file to execute (required)")
	cmd.Flags().StringArray("upload", nil, "Files to upload to nodes before execution")
	cmd.Flags().StringArray("inputs", nil, "Input files for module execution")
	cmd.Flags().StringToString("env", nil, "Environment variables (key=value)")
	cmd.Flags().Int("timeout", 600, "Command timeout in seconds")
	cmd.Flags().Int("concurrency", 0, "Max concurrent executions (0 = all nodes)")
	cmd.Flags().String("remote-dir", "/tmp/gaxx", "Remote directory for uploaded files")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("module")
	return cmd
}

// runFleetCommand executes commands across fleet nodes
func runFleetCommand(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	provider, _ := cmd.Flags().GetString("provider")
	modulePath, _ := cmd.Flags().GetString("module")
	inputs, _ := cmd.Flags().GetStringArray("inputs")
	envVars, _ := cmd.Flags().GetStringToString("env")
	timeout, _ := cmd.Flags().GetInt("timeout")
	concurrency, _ := cmd.Flags().GetInt("concurrency")

	reg, cc, err := resolveRegistry(cmd)
	if err != nil {
		return err
	}
	if provider == "" {
		provider = cc.cfg.Providers.Default
	}

	p, err := reg.Get(provider)
	if err != nil {
		return err
	}

	nodes, err := p.ListNodes(cmd.Context(), name)
	if err != nil {
		return err
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes found for fleet %s", name)
	}

	var task *api.TaskSpec
	if modulePath != "" {
		task, err = loadTaskModule(modulePath)
		if err != nil {
			return fmt.Errorf("load module: %w", err)
		}
		// Merge environment variables
		for k, v := range envVars {
			task.Env[k] = v
		}
	} else if len(args) > 0 {
		// Direct command execution
		task = &api.TaskSpec{
			Name:        "direct-command",
			Description: "Direct command execution",
			Command:     args[0],
			Args:        args[1:],
			Env:         envVars,
			ChunkSize:   1,
		}
	} else {
		return fmt.Errorf("either --module or command arguments required")
	}

	return executeTaskOnFleet(cmd.Context(), nodes, task, inputs, timeout, concurrency)
}

// loadTaskModule loads a YAML task module
func loadTaskModule(path string) (*api.TaskSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var task api.TaskSpec
	if err := yaml.Unmarshal(data, &task); err != nil {
		return nil, err
	}
	if task.Env == nil {
		task.Env = make(map[string]string)
	}
	return &task, nil
}

// executeTaskOnFleet executes a task across all nodes in the fleet
func executeTaskOnFleet(ctx context.Context, nodes []prov.Node, task *api.TaskSpec, inputs []string, timeout, concurrency int) error {
	// Start performance timing
	taskStart := time.Now()
	taskLabels := map[string]string{
		"task":      task.Name,
		"provider":  "unknown", // Will be set per node
		"component": "task_execution",
	}

	fmt.Printf("Executing task '%s' on %d nodes\n", task.Name, len(nodes))

	// Record task start
	telemetry.CounterGlobal("gaxx_tasks_started", 1, taskLabels)
	telemetry.GaugeGlobal("gaxx_task_nodes_total", float64(len(nodes)), taskLabels)

	var inputChunks [][]string
	if len(inputs) > 0 {
		allInputs, err := loadInputFiles(inputs)
		if err != nil {
			return fmt.Errorf("load inputs: %w", err)
		}
		if task.ChunkSize > 0 {
			inputChunks = core.ChunkInputs(allInputs, task.ChunkSize)
		} else {
			inputChunks = [][]string{allInputs}
		}
	} else {
		// No inputs, just run once per node
		inputChunks = [][]string{{}}
	}

	if concurrency <= 0 {
		concurrency = len(nodes)
	}

	semaphore := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make(map[string][]nodeResult)

	for i, node := range nodes {
		chunkIdx := i % len(inputChunks)
		chunk := inputChunks[chunkIdx]

		wg.Add(1)
		go func(node prov.Node, chunk []string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := executeOnNode(ctx, node, task, chunk, timeout)

			mu.Lock()
			results[node.Name] = append(results[node.Name], result)
			mu.Unlock()

			status := "✓"
			if result.ExitCode != 0 {
				status = "✗"
			}
			fmt.Printf("%s %s: exit=%d duration=%dms\n", status, node.Name, result.ExitCode, result.Duration)
		}(node, chunk)
	}

	wg.Wait()

	// Calculate final metrics
	taskDuration := time.Since(taskStart)
	successful := 0
	failed := 0
	for _, nodeResults := range results {
		for _, result := range nodeResults {
			if result.ExitCode == 0 {
				successful++
			} else {
				failed++
			}
		}
	}

	// Record task completion metrics
	telemetry.TimerGlobal("gaxx_task_duration", taskDuration, taskLabels)
	telemetry.CounterGlobal("gaxx_task_executions_successful", float64(successful), taskLabels)
	telemetry.CounterGlobal("gaxx_task_executions_failed", float64(failed), taskLabels)

	// Calculate and record success rate
	total := successful + failed
	if total > 0 {
		successRate := float64(successful) / float64(total) * 100
		telemetry.GaugeGlobal("gaxx_task_success_rate_percent", successRate, taskLabels)
	}

	// Record completion
	if failed == 0 {
		telemetry.CounterGlobal("gaxx_tasks_completed_successfully", 1, taskLabels)
	} else {
		telemetry.CounterGlobal("gaxx_tasks_completed_with_failures", 1, taskLabels)
	}

	fmt.Printf("\nSummary: %d successful, %d failed (%.2fs total)\n", successful, failed, taskDuration.Seconds())

	if failed > 0 {
		fmt.Println("\nFailed outputs:")
		for nodeName, nodeResults := range results {
			for _, result := range nodeResults {
				if result.ExitCode != 0 {
					fmt.Printf("Node %s:\n%s\n", nodeName, result.Stderr)
				}
			}
		}
	}

	return nil
}

type nodeResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration int64
}

// executeOnNode executes a task on a single node
func executeOnNode(ctx context.Context, node prov.Node, task *api.TaskSpec, chunk []string, timeout int) nodeResult {
	nodeStart := time.Now()

	// For demonstration, try agent first, fall back to SSH
	result, err := executeViaAgent(ctx, node, task, chunk, timeout)

	nodeLabels := map[string]string{
		"node_ip":   node.IP,
		"node_name": node.Name,
		"task":      task.Name,
		"component": "node_execution",
	}

	if err == nil {
		// Record successful agent execution
		telemetry.TimerGlobal("gaxx_node_execution_duration", time.Since(nodeStart), nodeLabels)
		telemetry.CounterGlobal("gaxx_node_executions_successful", 1, nodeLabels)
		return result
	}

	// Record agent failure and try fallback
	telemetry.CounterGlobal("gaxx_agent_failures", 1, nodeLabels)

	// Fallback to SSH execution (simplified for now)
	telemetry.CounterGlobal("gaxx_node_executions_failed", 1, nodeLabels)
	telemetry.TimerGlobal("gaxx_node_execution_duration", time.Since(nodeStart), nodeLabels)

	return nodeResult{
		ExitCode: 1,
		Stderr:   fmt.Sprintf("Agent execution failed: %v", err),
		Duration: time.Since(nodeStart).Milliseconds(),
	}
}

// executeViaAgent executes via the gaxx-agent API
func executeViaAgent(ctx context.Context, node prov.Node, task *api.TaskSpec, chunk []string, timeout int) (nodeResult, error) {
	url := fmt.Sprintf("http://%s:8088/v0/exec", node.IP)

	// Prepare command with template rendering if needed
	command := task.Command
	args := make([]string, len(task.Args))
	copy(args, task.Args)

	// Simple template replacement for {{ item }}
	if len(chunk) > 0 {
		// Create a temp file with chunk data
		tmpFile := fmt.Sprintf("/tmp/gaxx-chunk-%d", time.Now().UnixNano())
		chunkData := strings.Join(chunk, "\n")

		for i, arg := range args {
			args[i] = strings.ReplaceAll(arg, "{{ item }}", tmpFile)
		}

		// Add command to write chunk data first
		command = "sh"
		writeCmd := fmt.Sprintf("echo %q > %s && %s", chunkData, tmpFile, task.Command)
		args = []string{"-c", writeCmd + " " + strings.Join(args, " ")}
	}

	// Convert environment map to slice
	var env []string
	for k, v := range task.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	execReq := agent.ExecRequest{
		Command: command,
		Args:    args,
		Env:     env,
		Timeout: timeout,
	}

	reqBody, err := json.Marshal(execReq)
	if err != nil {
		return nodeResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		return nodeResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: time.Duration(timeout+10) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nodeResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nodeResult{}, fmt.Errorf("agent returned status %d", resp.StatusCode)
	}

	var execResp agent.ExecResponse
	if err := json.NewDecoder(resp.Body).Decode(&execResp); err != nil {
		return nodeResult{}, err
	}

	return nodeResult{
		ExitCode: execResp.ExitCode,
		Stdout:   execResp.Stdout,
		Stderr:   execResp.Stderr,
		Duration: execResp.Duration,
	}, nil
}

// loadInputFiles loads and combines multiple input files
func loadInputFiles(paths []string) ([]string, error) {
	var allInputs []string
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				allInputs = append(allInputs, line)
			}
		}
	}
	return allInputs, nil
}

// runScanCommand implements scan functionality with file upload
func runScanCommand(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	provider, _ := cmd.Flags().GetString("provider")
	modulePath, _ := cmd.Flags().GetString("module")
	uploadFiles, _ := cmd.Flags().GetStringArray("upload")
	inputs, _ := cmd.Flags().GetStringArray("inputs")
	envVars, _ := cmd.Flags().GetStringToString("env")
	timeout, _ := cmd.Flags().GetInt("timeout")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	remoteDir, _ := cmd.Flags().GetString("remote-dir")

	reg, cc, err := resolveRegistry(cmd)
	if err != nil {
		return err
	}
	if provider == "" {
		provider = cc.cfg.Providers.Default
	}

	p, err := reg.Get(provider)
	if err != nil {
		return err
	}

	nodes, err := p.ListNodes(cmd.Context(), name)
	if err != nil {
		return err
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes found for fleet %s", name)
	}

	task, err := loadTaskModule(modulePath)
	if err != nil {
		return fmt.Errorf("load module: %w", err)
	}

	// Merge environment variables
	for k, v := range envVars {
		task.Env[k] = v
	}

	// Upload files to all nodes first
	if len(uploadFiles) > 0 {
		fmt.Printf("Uploading %d files to %d nodes...\n", len(uploadFiles), len(nodes))
		if err := uploadFilesToFleet(cmd.Context(), nodes, uploadFiles, remoteDir, cc.cfg); err != nil {
			return fmt.Errorf("upload files: %w", err)
		}
		fmt.Println("File upload completed")
	}

	// Update input paths to use remote paths
	remoteInputs := make([]string, len(inputs))
	for i, input := range inputs {
		remoteInputs[i] = fmt.Sprintf("%s/%s", remoteDir, input)
	}

	return executeTaskOnFleet(cmd.Context(), nodes, task, remoteInputs, timeout, concurrency)
}

// uploadFilesToFleet uploads files to all nodes in the fleet
func uploadFilesToFleet(ctx context.Context, nodes []prov.Node, files []string, remoteDir string, cfg prov.Config) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(nodes))

	for _, node := range nodes {
		wg.Add(1)
		go func(node prov.Node) {
			defer wg.Done()
			if err := uploadFilesToNode(ctx, node, files, remoteDir, cfg); err != nil {
				errChan <- fmt.Errorf("upload to %s: %w", node.Name, err)
			}
		}(node)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		return err
	}

	return nil
}

// uploadFilesToNode uploads files to a single node using SCP
func uploadFilesToNode(ctx context.Context, node prov.Node, files []string, remoteDir string, cfg prov.Config) error {
	// This is a simplified implementation using the agent's exec endpoint
	// In a real implementation, you'd use SCP/SFTP

	// Create remote directory
	createDirCmd := fmt.Sprintf("mkdir -p %s", remoteDir)
	if err := executeSimpleCommand(ctx, node, createDirCmd); err != nil {
		return fmt.Errorf("create remote dir: %w", err)
	}

	// For each file, read content and write to remote
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read local file %s: %w", file, err)
		}

		// Use base64 encoding to safely transfer binary files
		remotePath := fmt.Sprintf("%s/%s", remoteDir, file)
		writeCmd := fmt.Sprintf("echo %q | base64 -d > %s", content, remotePath)

		if err := executeSimpleCommand(ctx, node, writeCmd); err != nil {
			return fmt.Errorf("write remote file %s: %w", remotePath, err)
		}
	}

	return nil
}

// executeSimpleCommand executes a simple command on a node via agent
func executeSimpleCommand(ctx context.Context, node prov.Node, command string) error {
	url := fmt.Sprintf("http://%s:8088/v0/exec", node.IP)

	execReq := agent.ExecRequest{
		Command: "sh",
		Args:    []string{"-c", command},
		Timeout: 30,
	}

	reqBody, err := json.Marshal(execReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 35 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("agent returned status %d", resp.StatusCode)
	}

	var execResp agent.ExecResponse
	if err := json.NewDecoder(resp.Body).Decode(&execResp); err != nil {
		return err
	}

	if execResp.ExitCode != 0 {
		return fmt.Errorf("command failed with exit code %d: %s", execResp.ExitCode, execResp.Stderr)
	}

	return nil
}

// initializeGaxx sets up the Gaxx environment
func initializeGaxx(cmd *cobra.Command) error {
	force, _ := cmd.Flags().GetBool("force")
	cfgPath, _ := cmd.Flags().GetString("config")

	// Determine config directory
	var configDir string
	if cfgPath == "" {
		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			home, _ := os.UserHomeDir()
			base = filepath.Join(home, ".config")
		}
		configDir = filepath.Join(base, "gaxx")
		cfgPath = filepath.Join(configDir, "config.yaml")
	} else {
		configDir = filepath.Dir(cfgPath)
	}

	fmt.Printf("Initializing Gaxx configuration in %s\n", configDir)

	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Create SSH directory
	sshDir := filepath.Join(configDir, "ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("create SSH directory: %w", err)
	}

	// Generate SSH key if it doesn't exist
	sshKeyPath := filepath.Join(sshDir, "id_ed25519")
	if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) || force {
		fmt.Println("Generating SSH key...")
		pubKey, err := gssh.GenerateEd25519Keypair(sshKeyPath)
		if err != nil {
			return fmt.Errorf("generate SSH key: %w", err)
		}
		fmt.Printf("SSH key generated: %s\n", sshKeyPath)
		fmt.Printf("Public key: %s\n", pubKey)
	} else {
		fmt.Println("SSH key already exists")
	}

	// Create known_hosts file
	knownHostsPath := filepath.Join(configDir, "known_hosts")
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) || force {
		if err := os.WriteFile(knownHostsPath, []byte(""), 0644); err != nil {
			return fmt.Errorf("create known_hosts: %w", err)
		}
		fmt.Printf("Created known_hosts file: %s\n", knownHostsPath)
	}

	// Create default config if it doesn't exist
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) || force {
		fmt.Printf("Creating default configuration: %s\n", cfgPath)
		defaultConfig := createDefaultConfig(sshDir, knownHostsPath)
		configData, err := yaml.Marshal(defaultConfig)
		if err != nil {
			return fmt.Errorf("marshal config: %w", err)
		}

		if err := os.WriteFile(cfgPath, configData, 0644); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
	} else {
		fmt.Println("Configuration file already exists")
	}

	// Create secrets.env template
	secretsPath := filepath.Join(configDir, "secrets.env")
	if _, err := os.Stat(secretsPath); os.IsNotExist(err) || force {
		secretsTemplate := `# Gaxx Secrets
# Add your cloud provider tokens here (these take precedence over config.yaml)
# LINODE_TOKEN=your_token_here
# VULTR_TOKEN=your_token_here
`
		if err := os.WriteFile(secretsPath, []byte(secretsTemplate), 0600); err != nil {
			return fmt.Errorf("write secrets template: %w", err)
		}
		fmt.Printf("Created secrets template: %s\n", secretsPath)
	}

	fmt.Println("\nGaxx initialization complete!")
	fmt.Println("\nNext steps:")
	fmt.Printf("1. Edit %s to configure your providers\n", cfgPath)
	fmt.Printf("2. Add provider tokens to %s if needed\n", secretsPath)
	fmt.Println("3. Test with: gaxx --help")

	return nil
}

// createDefaultConfig creates a default configuration
func createDefaultConfig(sshDir, knownHostsPath string) prov.Config {
	return prov.Config{
		Providers: struct {
			Default string `yaml:"default"`
			Linode  struct {
				Token  string   `yaml:"token"`
				Region string   `yaml:"region"`
				Type   string   `yaml:"type"`
				Image  string   `yaml:"image"`
				Tags   []string `yaml:"tags"`
			} `yaml:"linode"`
			Vultr struct {
				Token  string   `yaml:"token"`
				Region string   `yaml:"region"`
				Plan   string   `yaml:"plan"`
				OSID   string   `yaml:"os_id"`
				Tags   []string `yaml:"tags"`
			} `yaml:"vultr"`
			LocalSSH struct {
				Hosts []struct {
					Name    string `yaml:"name"`
					IP      string `yaml:"ip"`
					User    string `yaml:"user"`
					KeyPath string `yaml:"key_path"`
					Port    int    `yaml:"port"`
				} `yaml:"hosts"`
			} `yaml:"localssh"`
		}{
			Default: "linode",
			Linode: struct {
				Token  string   `yaml:"token"`
				Region string   `yaml:"region"`
				Type   string   `yaml:"type"`
				Image  string   `yaml:"image"`
				Tags   []string `yaml:"tags"`
			}{
				Token:  "",
				Region: "us-east",
				Type:   "g6-nanode-1",
				Image:  "linode/ubuntu22.04",
				Tags:   []string{"gaxx"},
			},
			Vultr: struct {
				Token  string   `yaml:"token"`
				Region string   `yaml:"region"`
				Plan   string   `yaml:"plan"`
				OSID   string   `yaml:"os_id"`
				Tags   []string `yaml:"tags"`
			}{
				Token:  "",
				Region: "ewr",
				Plan:   "vc2-1c-1gb",
				OSID:   "477",
				Tags:   []string{"gaxx"},
			},
			LocalSSH: struct {
				Hosts []struct {
					Name    string `yaml:"name"`
					IP      string `yaml:"ip"`
					User    string `yaml:"user"`
					KeyPath string `yaml:"key_path"`
					Port    int    `yaml:"port"`
				} `yaml:"hosts"`
			}{
				Hosts: []struct {
					Name    string `yaml:"name"`
					IP      string `yaml:"ip"`
					User    string `yaml:"user"`
					KeyPath string `yaml:"key_path"`
					Port    int    `yaml:"port"`
				}{
					{
						Name:    "example-local",
						IP:      "192.168.1.100",
						User:    "gx",
						KeyPath: filepath.Join(sshDir, "id_ed25519"),
						Port:    22,
					},
				},
			},
		},
		SSH: struct {
			KeyDir     string `yaml:"key_dir"`
			KnownHosts string `yaml:"known_hosts"`
		}{
			KeyDir:     sshDir,
			KnownHosts: knownHostsPath,
		},
		Defaults: struct {
			User           string `yaml:"user"`
			SSHPort        int    `yaml:"ssh_port"`
			Retries        int    `yaml:"retries"`
			TimeoutSeconds int    `yaml:"timeout_seconds"`
		}{
			User:           "gx",
			SSHPort:        22,
			Retries:        3,
			TimeoutSeconds: 300,
		},
		Telemetry: struct {
			Enabled         bool   `yaml:"enabled"`
			OTLPEndpoint    string `yaml:"otlp_endpoint"`
			MonitoringPort  int    `yaml:"monitoring_port"`
			MetricsInterval int    `yaml:"metrics_interval"`
		}{
			Enabled:         false,
			OTLPEndpoint:    "",
			MonitoringPort:  9090,
			MetricsInterval: 30,
		},
	}
}

// Generate shell completion scripts
func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate the autocompletion script for the specified shell",
		Args:      cobra.ExactValidArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
	return cmd
}
