package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	core "github.com/3cpo-dev/gaxx/internal/core"
	prov "github.com/3cpo-dev/gaxx/internal/providers"
	lin "github.com/3cpo-dev/gaxx/internal/providers/linode"
	localssh "github.com/3cpo-dev/gaxx/internal/providers/localssh"
	vlt "github.com/3cpo-dev/gaxx/internal/providers/vultr"
	"github.com/3cpo-dev/gaxx/internal/telemetry"
)

var (
	version   = "1.0.0"
	commit    = ""
	buildDate = "8/9/2025"
)

// Create the root command
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gaxx",
		Short: "Gaxx: distributed, burstable VPS task orchestration",
		Long:  "Gaxx orchestrates short-lived fleets of nodes to execute tasks with strong security defaults.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringP("log", "l", "info", "Set log level. Available: debug, info, warn, error, fatal")
	cmd.PersistentFlags().String("config", "", "config file")
	cmd.PersistentFlags().String("proxy", "", "HTTP Proxy (Useful for debugging. Example: http://127.0.0.1:8080)")
	cmd.PersistentFlags().BoolP("toggle", "t", false, "Help message for toggle")

	cmd.PersistentPreRun = func(c *cobra.Command, args []string) {
		levelStr, _ := c.Flags().GetString("log")
		switch levelStr {
		case "trace":
			zerolog.SetGlobalLevel(zerolog.TraceLevel)
		case "debug":
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		case "info":
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		case "warn":
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		case "error":
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		case "fatal":
			zerolog.SetGlobalLevel(zerolog.FatalLevel)
		default:
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}
		if proxy, _ := c.Flags().GetString("proxy"); proxy != "" {
			_ = os.Setenv("HTTP_PROXY", proxy)
			_ = os.Setenv("HTTPS_PROXY", proxy)
		}
	}

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newImagesCmd())
	cmd.AddCommand(newSpawnCmd())
	cmd.AddCommand(newLsCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newScanCmd())
	cmd.AddCommand(newScpCmd())
	cmd.AddCommand(newSSHCmd())
	cmd.AddCommand(newCompletionCmd())
	return cmd
}

// Create the version command
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("gaxx %s (%s) %s\n", version, commit, buildDate)
		},
	}
}

// Setup the logger
func setupLogger() {
	level := zerolog.InfoLevel
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(level)
}

// Main entry point
func main() {
	setupLogger()

	// Initialize telemetry if enabled
	setupTelemetry()
	defer telemetry.Shutdown()

	root := newRootCmd()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	root.SetContext(ctx)

	// Record application start
	telemetry.CounterGlobal("gaxx_app_starts", 1, map[string]string{
		"version":   version,
		"component": "cli",
	})

	if err := root.Execute(); err != nil {
		telemetry.CounterGlobal("gaxx_app_errors", 1, map[string]string{
			"error":     err.Error(),
			"component": "cli",
		})
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// setupTelemetry initializes telemetry based on configuration
func setupTelemetry() {
	// Try to load config to get telemetry settings
	cfg, err := core.LoadConfig("")
	if err != nil {
		// If config loading fails, disable telemetry
		telemetry.InitGlobal(false, "")
		return
	}

	// Set defaults if not specified
	monitoringPort := cfg.Telemetry.MonitoringPort
	if monitoringPort == 0 {
		monitoringPort = 9090
	}

	telemetry.InitGlobal(cfg.Telemetry.Enabled, cfg.Telemetry.OTLPEndpoint)

	// Configure metrics interval if specified
	if cfg.Telemetry.MetricsInterval > 0 {
		// TODO: Use MetricsInterval to configure flush frequency
		log.Debug().Int("interval", cfg.Telemetry.MetricsInterval).Msg("Custom metrics interval configured")
	}

	if cfg.Telemetry.Enabled {
		log.Info().
			Bool("enabled", cfg.Telemetry.Enabled).
			Str("otlp_endpoint", cfg.Telemetry.OTLPEndpoint).
			Int("monitoring_port", monitoringPort).
			Msg("Telemetry initialized")

		// Start monitoring server in background if enabled
		go startMonitoringServer(monitoringPort)
	}
}

// startMonitoringServer starts the monitoring HTTP server
func startMonitoringServer(port int) {
	collector := telemetry.GetGlobal()
	perfMon := telemetry.NewPerformanceMonitor(collector, true)
	defer perfMon.Shutdown()

	addr := fmt.Sprintf(":%d", port)
	server := telemetry.NewMonitoringServer(addr, collector, perfMon)

	// Register default health checks
	for name, checkFn := range telemetry.DefaultHealthChecks() {
		server.RegisterHealthCheck(name, checkFn)
	}

	log.Info().Int("port", port).Msg("Starting CLI monitoring server")
	if err := server.Start(); err != nil && err.Error() != "http: Server closed" {
		log.Error().Err(err).Msg("CLI monitoring server failed")
	}
}

// Create the providers command
func newProvidersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Inspect configured providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			cfg, err := core.LoadConfig(cfgPath)
			if err != nil {
				return err
			}
			reg := prov.NewRegistry()
			reg.Register(localssh.New(cfg))
			reg.Register(lin.New(cfg))
			reg.Register(vlt.New(cfg))
			fmt.Printf("default: %s\n", cfg.Providers.Default)
			for _, name := range []string{"localssh", "linode", "vultr"} {
				if _, err := reg.Get(name); err == nil {
					fmt.Printf("registered: %s\n", name)
				}
			}
			return nil
		},
	}
	return cmd
}
