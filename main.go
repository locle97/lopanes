// Command lopanes renders a YAML-configured TUI dashboard of shell-driven
// widgets, with an interactive mode and a one-shot print mode.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/locle97/lopanes/internal/config"
	"github.com/locle97/lopanes/internal/printer"
	"github.com/locle97/lopanes/internal/tui"
	"github.com/locle97/lopanes/internal/version"
)

func main() {
	cfgPath := flag.String("config", "", "config file path (default: ./lopanes.yaml or ~/.config/lopanes/config.yaml)")
	printMode := flag.Bool("print", false, "render once to stdout and exit")
	width := flag.Int("width", 0, "print-mode render width (default: terminal width or 80)")
	noColor := flag.Bool("no-color", false, "strip ANSI escapes in print mode")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.Version())
		return
	}

	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		fail(err)
	}

	if *printMode {
		out := printer.Render(cfg, printer.Options{Width: *width, NoColor: *noColor})
		fmt.Println(out)
		return
	}

	m := tui.New(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "lopanes:", err)
	os.Exit(1)
}

// loadConfig resolves and loads the configuration. An explicit flagPath must
// exist. Otherwise the default search order is ./lopanes.yaml then
// ~/.config/lopanes/config.yaml; when neither exists, a starter config is
// written to the latter and loaded. If that write fails, the embedded default
// is parsed in memory so the first run never hard-fails.
func loadConfig(flagPath string) (config.Config, error) {
	if flagPath != "" {
		if _, err := os.Stat(flagPath); err != nil {
			return config.Config{}, fmt.Errorf("config %q: %w", flagPath, err)
		}
		return config.Load(flagPath)
	}

	candidates := []string{"./lopanes.yaml"}
	var globalPath string
	if home, err := os.UserHomeDir(); err == nil {
		globalPath = filepath.Join(home, ".config", "lopanes", "config.yaml")
		candidates = append(candidates, globalPath)
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return config.Load(c)
		}
	}

	if globalPath == "" {
		fmt.Fprintln(os.Stderr, "lopanes: no config found and no home directory; using built-in default")
		return config.Parse(config.DefaultYAML)
	}
	if err := config.WriteDefault(globalPath); err != nil {
		fmt.Fprintf(os.Stderr, "lopanes: no config found and could not write starter config (%v); using built-in default\n", err)
		return config.Parse(config.DefaultYAML)
	}
	fmt.Fprintf(os.Stderr, "lopanes: no config found; wrote a starter config to %s — edit it to customize.\n", globalPath)
	return config.Load(globalPath)
}
