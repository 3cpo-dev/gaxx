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
	root := newRootCmd()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	root.SetContext(ctx)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
