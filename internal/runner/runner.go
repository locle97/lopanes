// Package runner executes a widget's shell command with a timeout and captures
// a structured result. It has no Bubble Tea awareness.
package runner

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// RunSpec describes one command to run.
type RunSpec struct {
	Script  string
	Timeout time.Duration
	Env     map[string]string
}

// Result is the structured outcome of a run.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	TimedOut bool
	Err      error // non-nil only for failures that are not a non-zero exit or timeout
	Duration time.Duration
}

// Run executes spec.Script via `bash -c` under a timeout derived from ctx.
// The process environment is inherited and extended with spec.Env.
func Run(ctx context.Context, spec RunSpec) Result {
	start := time.Now()
	runCtx := ctx
	if spec.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, spec.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(runCtx, "bash", "-c", spec.Script)
	cmd.Env = os.Environ()
	for k, v := range spec.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	res := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(start),
	}

	if runCtx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
		res.ExitCode = -1
		return res
	}
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			res.ExitCode = ee.ExitCode()
		} else {
			res.Err = err
			res.ExitCode = -1
		}
	}
	return res
}

// WidgetEnv builds the environment variables injected into every widget script.
// widgetW/widgetH are the inner box dimensions; termW/termH the terminal size.
// In print mode widgetH and termH may be 0 (unknown / unbounded).
func WidgetEnv(widgetW, widgetH, termW, termH int) map[string]string {
	return map[string]string{
		"WIDGET_W": strconv.Itoa(widgetW),
		"WIDGET_H": strconv.Itoa(widgetH),
		"COLUMNS":  strconv.Itoa(termW),
		"LINES":    strconv.Itoa(termH),
	}
}
