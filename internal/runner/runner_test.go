package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunSuccess(t *testing.T) {
	res := Run(context.Background(), RunSpec{Script: "echo hello", Timeout: time.Second})
	if res.Err != nil || res.TimedOut || res.ExitCode != 0 {
		t.Fatalf("unexpected: %+v", res)
	}
	if strings.TrimSpace(res.Stdout) != "hello" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}

func TestRunNonZeroExit(t *testing.T) {
	res := Run(context.Background(), RunSpec{Script: "echo oops >&2; exit 3", Timeout: time.Second})
	if res.TimedOut {
		t.Fatalf("should not be timed out: %+v", res)
	}
	if res.ExitCode != 3 {
		t.Fatalf("exit code = %d want 3", res.ExitCode)
	}
	if strings.TrimSpace(res.Stderr) != "oops" {
		t.Fatalf("stderr = %q", res.Stderr)
	}
}

func TestRunTimeout(t *testing.T) {
	start := time.Now()
	res := Run(context.Background(), RunSpec{Script: "sleep 5", Timeout: 100 * time.Millisecond})
	if !res.TimedOut {
		t.Fatalf("expected timeout, got %+v", res)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("timeout took too long: %v", time.Since(start))
	}
}

func TestRunInjectsEnv(t *testing.T) {
	env := WidgetEnv(12, 4, 80, 24)
	res := Run(context.Background(), RunSpec{
		Script:  `echo "$WIDGET_W $WIDGET_H $COLUMNS $LINES"`,
		Timeout: time.Second,
		Env:     env,
	})
	if got := strings.TrimSpace(res.Stdout); got != "12 4 80 24" {
		t.Fatalf("env injection: stdout = %q", got)
	}
}

func TestWidgetEnv(t *testing.T) {
	env := WidgetEnv(1, 2, 3, 4)
	want := map[string]string{"WIDGET_W": "1", "WIDGET_H": "2", "COLUMNS": "3", "LINES": "4"}
	for k, v := range want {
		if env[k] != v {
			t.Errorf("env[%s] = %q want %q", k, env[k], v)
		}
	}
}
