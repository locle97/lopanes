// Command quash-board renders a YAML-configured TUI dashboard of shell-driven
// widgets, with an interactive mode and a one-shot print mode.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/locle97/quash-board/internal/config"
	"github.com/locle97/quash-board/internal/printer"
	"github.com/locle97/quash-board/internal/tui"
	"github.com/locle97/quash-board/internal/version"
)

func main() {
	cfgPath := flag.String("config", "", "config file path (default: ./quash-board.yaml or ~/.config/quash-board/config.yaml)")
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
	fmt.Fprintln(os.Stderr, "quash-board:", err)
	os.Exit(1)
}

// resolveConfigPath returns the config file to use. An explicit flag must
// exist; otherwise the default search order is ./quash-board.yaml then
// ~/.config/quash-board/config.yaml.
func resolveConfigPath(flagPath string) (string, error) {
	if flagPath != "" {
		if _, err := os.Stat(flagPath); err != nil {
			return "", fmt.Errorf("config %q: %w", flagPath, err)
		}
		return flagPath, nil
	}
	candidates := []string{"./quash-board.yaml"}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".config", "quash-board", "config.yaml"))
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("no config found (looked in %s)", strings.Join(candidates, ", "))
}
