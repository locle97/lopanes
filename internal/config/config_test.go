package config

import (
	"strings"
	"testing"
	"time"
)

func TestParseValidConfigAppliesDefaults(t *testing.T) {
	src := `
default_interval: 2s
default_timeout: 4s
rows:
  - height: 1fr
    widgets:
      - name: cpu
        script: "echo hi"
      - name: mem
        script: "free -h"
        interval: 9s
        width: 2fr
  - height: 10
    widgets:
      - {name: logs, title: "Syslog", script: "tail -n 5 /var/log/syslog", timeout: 1s}
`
	cfg, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultInterval != 2*time.Second || cfg.DefaultTimeout != 4*time.Second {
		t.Fatalf("defaults wrong: %+v", cfg)
	}
	if len(cfg.Rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(cfg.Rows))
	}
	// Row 0 weighted.
	if cfg.Rows[0].HeightWeight != 1 || cfg.Rows[0].HeightFixed != 0 {
		t.Errorf("row0 height: %+v", cfg.Rows[0])
	}
	// Row 1 fixed.
	if cfg.Rows[1].HeightFixed != 10 || cfg.Rows[1].HeightWeight != 0 {
		t.Errorf("row1 height: %+v", cfg.Rows[1])
	}
	cpu := cfg.Rows[0].Widgets[0]
	if cpu.Title != "cpu" { // defaults to name
		t.Errorf("cpu title = %q want cpu", cpu.Title)
	}
	if cpu.Interval != 2*time.Second { // default interval
		t.Errorf("cpu interval = %v want 2s", cpu.Interval)
	}
	if cpu.Timeout != 4*time.Second { // default timeout
		t.Errorf("cpu timeout = %v want 4s", cpu.Timeout)
	}
	if cpu.WidthWeight != 1 { // default weight
		t.Errorf("cpu width = %d want 1", cpu.WidthWeight)
	}
	mem := cfg.Rows[0].Widgets[1]
	if mem.Interval != 9*time.Second || mem.WidthWeight != 2 {
		t.Errorf("mem overrides: %+v", mem)
	}
	logs := cfg.Rows[1].Widgets[0]
	if logs.Title != "Syslog" || logs.Timeout != 1*time.Second {
		t.Errorf("logs: %+v", logs)
	}
}

func TestParseDefaultsWhenTopLevelOmitted(t *testing.T) {
	cfg, err := Parse([]byte("rows:\n  - widgets:\n      - {name: a, script: \"echo a\"}\n"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if cfg.DefaultInterval != 5*time.Second || cfg.DefaultTimeout != 10*time.Second {
		t.Fatalf("builtin defaults wrong: %+v", cfg)
	}
	if cfg.Rows[0].HeightWeight != 1 { // omitted height defaults to 1fr
		t.Fatalf("omitted height should default to 1fr, got %+v", cfg.Rows[0])
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantSub string
	}{
		{"invalid yaml", "rows: [oops", "parsing YAML"},
		{"unknown field", "rows:\n  - widgets:\n      - {name: a, script: x, bogus: 1}\n", "field bogus"},
		{"no rows", "default_interval: 1s\n", "no rows"},
		{"empty row", "rows:\n  - widgets: []\n", "no widgets"},
		{"missing name", "rows:\n  - widgets:\n      - {script: \"echo a\"}\n", "name is required"},
		{"missing script", "rows:\n  - widgets:\n      - {name: a}\n", "script is required"},
		{"bad interval", "rows:\n  - widgets:\n      - {name: a, script: x, interval: nope}\n", "interval"},
		{"bad height", "rows:\n  - height: huge\n    widgets:\n      - {name: a, script: x}\n", "height"},
		{"bad default", "default_interval: nope\nrows:\n  - widgets:\n      - {name: a, script: x}\n", "default_interval"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.src))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantSub)
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantSub)
			}
		})
	}
}
