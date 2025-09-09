package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/3cpo-dev/gaxx/internal/core"
	"github.com/spf13/cobra"
)

var (
	version   = "2.0.0"
	commit    = ""
	buildDate = "8/9/2025"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gaxx",
		Short: "Distributed task runner for VPS fleets",
		Long:  "High-performance distributed task runner for VPS fleets with enterprise security.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringP("log", "l", "info", "Set log level. Available: debug, info, warn, error, fatal")
	cmd.PersistentFlags().String("config", "", "config file")
	cmd.PersistentFlags().String("proxy", "", "HTTP Proxy (Useful for debugging. Example: http://127.0.0.1:8080)")

	cmd.AddCommand(newSpawnCmd())
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newMetricsCmd())
	cmd.AddCommand(newVersionCmd())

	return cmd
}

func newSpawnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spawn",
		Short: "Create a fleet of instances",
		Long:  "Create a fleet of cloud instances for distributed task execution.",
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, _ := cmd.Flags().GetString("provider")
			count, _ := cmd.Flags().GetInt("count")
			name, _ := cmd.Flags().GetString("name")

			if name == "" {
				return fmt.Errorf("fleet name is required")
			}

			config, err := core.LoadConfig("")
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			var p core.Provider
			switch provider {
			case "linode":
				if config.Token == "" {
					return fmt.Errorf("LINODE_TOKEN environment variable is required")
				}
				p = core.NewLinodeProvider(config.Token)
			case "vultr":
				if config.Token == "" {
					return fmt.Errorf("VULTR_API_KEY environment variable is required")
				}
				p = core.NewVultrProvider(config.Token)
			default:
				return fmt.Errorf("unsupported provider: %s (supported: linode, vultr)", provider)
			}

			gaxx := core.NewGaxx(config, p)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			fmt.Printf("🚀 Creating fleet '%s' with %d instances using %s...\n", name, count, provider)
			instances, err := gaxx.SpawnFleet(ctx, name, count)
			if err != nil {
				return fmt.Errorf("spawn fleet: %w", err)
			}

			fmt.Printf("✅ Created fleet '%s' with %d instances:\n", name, len(instances))
			for _, inst := range instances {
				fmt.Printf("  %s: %s\n", inst.Name, inst.IP)
			}
			return nil
		},
	}

	cmd.Flags().String("provider", "linode", "Cloud provider (linode, vultr)")
	cmd.Flags().Int("count", 1, "Number of instances to create")
	cmd.Flags().String("name", "", "Fleet name (required)")

	return cmd
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute command on fleet",
		Long:  "Execute a command across all instances in a fleet.",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			command, _ := cmd.Flags().GetString("command")

			if name == "" {
				return fmt.Errorf("fleet name is required")
			}
			if command == "" {
				return fmt.Errorf("command is required")
			}

			config, err := core.LoadConfig("")
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Use Linode as default provider for now
			p := core.NewLinodeProvider(config.Token)
			gaxx := core.NewGaxx(config, p)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()

			fmt.Printf("📋 Listing instances for fleet '%s'...\n", name)
			instances, err := gaxx.ListInstances(ctx, name)
			if err != nil {
				return fmt.Errorf("list instances: %w", err)
			}

			if len(instances) == 0 {
				return fmt.Errorf("no instances found for fleet '%s'", name)
			}

			task := core.Task{
				Command: command,
				Args:    args,
			}

			fmt.Printf("⚡ Executing command on %d instances...\n", len(instances))
			start := time.Now()
			err = gaxx.ExecuteTasks(ctx, instances, []core.Task{task})
			duration := time.Since(start)

			if err != nil {
				return fmt.Errorf("execute tasks: %w", err)
			}

			fmt.Printf("✅ Command completed in %v across %d instances\n", duration, len(instances))
			return nil
		},
	}

	cmd.Flags().String("name", "", "Fleet name (required)")
	cmd.Flags().String("command", "", "Command to execute (required)")

	return cmd
}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls [fleet-name]",
		Short: "List instances",
		Long:  "List all instances or instances in a specific fleet.",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}

			config, err := core.LoadConfig("")
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Use Linode as default provider for now
			p := core.NewLinodeProvider(config.Token)
			gaxx := core.NewGaxx(config, p)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			instances, err := gaxx.ListInstances(ctx, name)
			if err != nil {
				return fmt.Errorf("list instances: %w", err)
			}

			if len(instances) == 0 {
				if name != "" {
					fmt.Printf("No instances found for fleet '%s'\n", name)
				} else {
					fmt.Println("No instances found")
				}
				return nil
			}

			fmt.Printf("%-20s %-15s %-10s %-8s\n", "NAME", "IP", "ID", "USER")
			fmt.Println(strings.Repeat("-", 55))
			for _, inst := range instances {
				fmt.Printf("%-20s %-15s %-10s %-8s\n", inst.Name, inst.IP, inst.ID, inst.User)
			}
			return nil
		},
	}

	return cmd
}

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [fleet-name]",
		Short: "Delete fleet",
		Long:  "Delete all instances in a fleet or all instances if no fleet specified.",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}

			config, err := core.LoadConfig("")
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Use Linode as default provider for now
			p := core.NewLinodeProvider(config.Token)
			gaxx := core.NewGaxx(config, p)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			if name != "" {
				fmt.Printf("🗑️  Deleting fleet '%s'...\n", name)
			} else {
				fmt.Println("🗑️  Deleting all instances...")
			}

			if err := gaxx.DeleteFleet(ctx, name); err != nil {
				return fmt.Errorf("delete fleet: %w", err)
			}

			if name != "" {
				fmt.Printf("✅ Deleted fleet '%s'\n", name)
			} else {
				fmt.Println("✅ Deleted all instances")
			}
			return nil
		},
	}

	return cmd
}

func newMetricsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Show performance metrics",
		Long:  "Display current performance metrics for the simplified Gaxx instance.",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := core.LoadConfig("")
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Create a temporary instance to get metrics
			p := core.NewLinodeProvider(config.Token)
			gaxx := core.NewGaxx(config, p)

			requests, errors, duration := gaxx.GetMetrics()

			fmt.Println("📊 Gaxx Performance Metrics")
			fmt.Println(strings.Repeat("-", 40))
			fmt.Printf("Total Requests: %d\n", requests)
			fmt.Printf("Total Errors:   %d\n", errors)
			fmt.Printf("Total Duration: %v\n", duration)

			if requests > 0 {
				avgDuration := duration / time.Duration(requests)
				errorRate := float64(errors) / float64(requests) * 100
				fmt.Printf("Avg Duration:   %v\n", avgDuration)
				fmt.Printf("Error Rate:     %.2f%%\n", errorRate)
			}

			return nil
		},
	}

	return cmd
}

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Gaxx v%s\n", version)
			fmt.Printf("Build Date: %s\n", buildDate)
			if commit != "" {
				fmt.Printf("Commit: %s\n", commit)
			}
			fmt.Println("Architecture: High-performance core")
			fmt.Println("Performance: Optimized for speed and reliability")
			return nil
		},
	}

	return cmd
}
