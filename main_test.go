package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigPathExplicit(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "my.yaml")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := resolveConfigPath(p)
	if err != nil || got != p {
		t.Fatalf("explicit path: got %q err %v", got, err)
	}
}

func TestResolveConfigPathExplicitMissing(t *testing.T) {
	if _, err := resolveConfigPath("/no/such/file.yaml"); err == nil {
		t.Fatal("missing explicit path should error")
	}
}

func TestResolveConfigPathSearchesCwd(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("lopanes.yaml", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := resolveConfigPath("")
	if err != nil {
		t.Fatalf("search err: %v", err)
	}
	if filepath.Base(got) != "lopanes.yaml" {
		t.Fatalf("expected cwd config, got %q", got)
	}
}

func TestResolveConfigPathNoneFound(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", dir) // no ~/.config/lopanes/config.yaml here
	if _, err := resolveConfigPath(""); err == nil {
		t.Fatal("no config anywhere should error")
	}
}
