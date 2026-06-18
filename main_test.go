package main

import (
	"os"
	"path/filepath"
	"testing"
)

const validYAML = `
rows:
  - height: 1fr
    widgets:
      - {name: a, script: "echo hi"}
`

func TestLoadConfigExplicit(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "my.yaml")
	if err := os.WriteFile(p, []byte(validYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(p)
	if err != nil {
		t.Fatalf("explicit path: err %v", err)
	}
	if len(cfg.Rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(cfg.Rows))
	}
}

func TestLoadConfigExplicitMissing(t *testing.T) {
	if _, err := loadConfig("/no/such/file.yaml"); err == nil {
		t.Fatal("missing explicit path should error")
	}
}

func TestLoadConfigSearchesCwd(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("lopanes.yaml", []byte(validYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(""); err != nil {
		t.Fatalf("search err: %v", err)
	}
}

func TestLoadConfigGeneratesWhenNoneFound(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", dir) // no ~/.config/lopanes/config.yaml here yet

	cfg, err := loadConfig("")
	if err != nil {
		t.Fatalf("expected generation to succeed, got err %v", err)
	}
	if len(cfg.Rows) == 0 {
		t.Fatal("generated config should have rows")
	}
	gen := filepath.Join(dir, ".config", "lopanes", "config.yaml")
	if _, err := os.Stat(gen); err != nil {
		t.Fatalf("expected starter config written at %s: %v", gen, err)
	}
}
