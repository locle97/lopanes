// Command lopanes renders a YAML-configured TUI dashboard of shell-driven
// widgets, with an interactive mode and a one-shot print mode.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	path, err := resolveConfigPath(*cfgPath)
	if err != nil {
		fail(err)
	}
	cfg, err := config.Load(path)
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

// resolveConfigPath returns the config file to use. An explicit flag must
// exist; otherwise the default search order is ./lopanes.yaml then
// ~/.config/lopanes/config.yaml.
func resolveConfigPath(flagPath string) (string, error) {
	if flagPath != "" {
		if _, err := os.Stat(flagPath); err != nil {
			return "", fmt.Errorf("config %q: %w", flagPath, err)
		}
		return flagPath, nil
	}
	candidates := []string{"./lopanes.yaml"}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".config", "lopanes", "config.yaml"))
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("no config found (looked in %s)", strings.Join(candidates, ", "))
}
