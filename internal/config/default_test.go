package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultYAMLParses(t *testing.T) {
	cfg, err := Parse(DefaultYAML)
	if err != nil {
		t.Fatalf("default config does not parse: %v", err)
	}
	if len(cfg.Rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(cfg.Rows))
	}
	if len(cfg.Rows[0].Widgets) != 2 || len(cfg.Rows[1].Widgets) != 2 {
		t.Fatalf("want 2 widgets per row, got %d and %d",
			len(cfg.Rows[0].Widgets), len(cfg.Rows[1].Widgets))
	}
}

func TestWriteDefaultCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.yaml")
	if err := WriteDefault(path); err != nil {
		t.Fatalf("WriteDefault: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(got) != string(DefaultYAML) {
		t.Fatal("written content does not match DefaultYAML")
	}
}

func TestWriteDefaultRefusesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteDefault(path); err == nil {
		t.Fatal("WriteDefault should error when target exists")
	}
}
