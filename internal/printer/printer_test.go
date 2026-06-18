package printer

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/locle97/lopanes/internal/config"
)

// lipglossWidth is the printed cell width of a line, ignoring ANSI escapes.
func lipglossWidth(s string) int { return lipgloss.Width(s) }

func oneWidgetCfg(script string) config.Config {
	return config.Config{
		DefaultInterval: time.Second,
		DefaultTimeout:  2 * time.Second,
		Rows: []config.Row{
			{HeightWeight: 1, Widgets: []config.Widget{
				{Name: "a", Title: "a", Script: script, Timeout: 2 * time.Second, WidthWeight: 1},
			}},
		},
	}
}

func TestRenderRunsScriptAndShowsOutput(t *testing.T) {
	out := Render(oneWidgetCfg("echo hello-print"), Options{Width: 30, NoColor: true})
	if !strings.Contains(out, "hello-print") {
		t.Fatalf("output missing script result:\n%s", out)
	}
	if !strings.Contains(out, "┌") || !strings.Contains(out, "└") {
		t.Fatalf("output missing box borders:\n%s", out)
	}
	// Every line should be exactly the render width.
	for _, ln := range strings.Split(out, "\n") {
		if w := lipglossWidth(ln); w != 30 {
			t.Fatalf("line width %d want 30: %q", w, ln)
		}
	}
}

func TestRenderDefaultWidth(t *testing.T) {
	out := Render(oneWidgetCfg("echo hi"), Options{NoColor: true}) // width 0 -> default 80
	first := strings.SplitN(out, "\n", 2)[0]
	if lipglossWidth(first) != 80 {
		t.Fatalf("default width not 80: %d", lipglossWidth(first))
	}
}

func TestRenderErrorState(t *testing.T) {
	out := Render(oneWidgetCfg("echo boom >&2; exit 7"), Options{Width: 30, NoColor: true})
	if !strings.Contains(out, "exit 7") {
		t.Fatalf("error label missing:\n%s", out)
	}
	if !strings.Contains(out, "boom") {
		t.Fatalf("stderr tail missing:\n%s", out)
	}
}

func TestRenderNoColorStripsANSI(t *testing.T) {
	// Script emits a red ANSI sequence; --no-color must strip it.
	out := Render(oneWidgetCfg(`printf '\033[31mRED\033[0m\n'`), Options{Width: 30, NoColor: true})
	if regexp.MustCompile("\x1b\\[").MatchString(out) {
		t.Fatalf("ANSI not stripped:\n%q", out)
	}
	if !strings.Contains(out, "RED") {
		t.Fatalf("text content lost:\n%s", out)
	}
}

func TestRenderConcurrentMultipleWidgets(t *testing.T) {
	cfg := config.Config{
		DefaultInterval: time.Second,
		DefaultTimeout:  2 * time.Second,
		Rows: []config.Row{
			{HeightWeight: 1, Widgets: []config.Widget{
				{Name: "a", Title: "a", Script: "echo AAA", Timeout: 2 * time.Second, WidthWeight: 1},
				{Name: "b", Title: "b", Script: "echo BBB", Timeout: 2 * time.Second, WidthWeight: 1},
			}},
		},
	}
	out := Render(cfg, Options{Width: 40, NoColor: true})
	if !strings.Contains(out, "AAA") || !strings.Contains(out, "BBB") {
		t.Fatalf("both widgets should render:\n%s", out)
	}
}
