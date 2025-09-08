package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	prov "github.com/3cpo-dev/gaxx/internal/providers"
	"gopkg.in/yaml.v3"
)

// LoadConfig reads YAML configuration from a path. If path is empty, it resolves
// $XDG_CONFIG_HOME/gaxx/config.yaml or ~/.config/gaxx/config.yaml.
func LoadConfig(path string) (prov.Config, error) {
	var cfg prov.Config
	if path == "" {
		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			home, _ := os.UserHomeDir()
			base = filepath.Join(home, ".config")
		}
		path = filepath.Join(base, "gaxx", "config.yaml")
	}
	f, err := os.Open(path)
	if err != nil {
		return cfg, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()
	content, err := io.ReadAll(f)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	// Merge secrets from secrets.env if present to avoid storing tokens in YAML
	secrets, _ := LoadSecretsEnv("")
	if v := os.Getenv("LINODE_TOKEN"); v != "" {
		secrets["LINODE_TOKEN"] = v
	}
	if v := os.Getenv("VULTR_TOKEN"); v != "" {
		secrets["VULTR_TOKEN"] = v
	}
	if t, ok := secrets["LINODE_TOKEN"]; ok && t != "" {
		cfg.Providers.Linode.Token = t
	}
	if t, ok := secrets["VULTR_TOKEN"]; ok && t != "" {
		cfg.Providers.Vultr.Token = t
	}
	return cfg, nil
}
