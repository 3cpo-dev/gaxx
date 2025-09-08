package main

import (
	"fmt"
	"strings"
	"time"

	core "github.com/3cpo-dev/gaxx/internal/core"
	prov "github.com/3cpo-dev/gaxx/internal/providers"
	lin "github.com/3cpo-dev/gaxx/internal/providers/linode"
	localssh "github.com/3cpo-dev/gaxx/internal/providers/localssh"
	vlt "github.com/3cpo-dev/gaxx/internal/providers/vultr"
	gssh "github.com/3cpo-dev/gaxx/internal/ssh"
	"github.com/spf13/cobra"
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

// Run a command on a fleet (stub)
func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Send a command to a fleet",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("run not yet implemented")
			return nil
		},
	}
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

// Initialize configuration and environment (stub)
func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "gaxx initialization command. Run this the first time.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := cmd.Flags().GetString("config")
			if cfg == "" {
				fmt.Println("creating default config at ~/.config/gaxx/config.yaml if missing")
			} else {
				fmt.Printf("creating default config at %s if missing\n", cfg)
			}
			fmt.Println("generating SSH key if needed and preparing known_hosts file")
			fmt.Println("init not yet implemented")
			return nil
		},
	}
}

// Scan command (stub)
func newScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Send a command to a fleet, but also with files upload & chunks splitting",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("scan not yet implemented")
			return nil
		},
	}
}
