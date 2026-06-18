package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultYAML is the built-in starter dashboard configuration, used both as
// the file written on first run and as the in-memory fallback when that write
// fails. It is the single source of truth for the default config.
//
//go:embed default.yaml
var DefaultYAML []byte

// WriteDefault writes the starter config to path, creating parent directories
// as needed. It refuses to overwrite an existing file.
func WriteDefault(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("creating config file: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(DefaultYAML); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}
