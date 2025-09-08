package core

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadSecretsEnv reads $XDG_CONFIG_HOME/gaxx/secrets.env (or ~/.config/gaxx/secrets.env)
// and returns key/value pairs. Lines starting with # are ignored. Format: KEY=VALUE
func LoadSecretsEnv(path string) (map[string]string, error) {
	if path == "" {
		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			home, _ := os.UserHomeDir()
			base = filepath.Join(home, ".config")
		}
		path = filepath.Join(base, "gaxx", "secrets.env")
	}
	f, err := os.Open(path)
	if err != nil {
		return map[string]string{}, nil // not fatal if missing
	}
	defer f.Close()
	out := map[string]string{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.IndexByte(line, '='); i >= 0 {
			k := strings.TrimSpace(line[:i])
			v := strings.TrimSpace(line[i+1:])
			out[k] = v
		}
	}
	return out, nil
}
