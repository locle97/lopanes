# quash-board

A terminal dashboard configured from YAML. Each widget runs a shell command and
displays its output in a bordered box. The grid reflows to your terminal size.

## Install

With a Go toolchain (1.25+):

```bash
go install github.com/locle97/quash-board@latest
```

This installs the `quash-board` binary into `$(go env GOPATH)/bin` (or `$GOBIN`
if set), which must be on your `$PATH`. Pin a specific version with, e.g.,
`go install github.com/locle97/quash-board@v0.1.0`.

### Build from source

```bash
git clone https://github.com/locle97/quash-board.git
cd quash-board
go build -o quash-board .
```

## Usage

```
quash-board [--config PATH] [--print] [--width N] [--no-color]
```

- `--config PATH` — config file. Default search order: `./quash-board.yaml`,
  then `~/.config/quash-board/config.yaml`.
- `--print` — run every widget once, print the dashboard, and exit (good for
  snapshots). Otherwise runs the interactive TUI.
- `--width N` — print-mode render width (useful when redirecting to a file).
- `--no-color` — strip ANSI escapes in print mode.

In interactive mode, press `q` or `Ctrl-C` to quit.

### Example

```bash
quash-board --config examples/quash-board.yaml
quash-board --print --config examples/quash-board.yaml --width 120 > snapshot.txt
```

## Configuration

```yaml
default_interval: 5s     # fallback per-widget refresh interval
default_timeout: 10s     # fallback per-widget script timeout
rows:
  - height: 1fr          # weight (Nfr) OR a fixed line count (e.g. 10)
    widgets:
      - name: cpu        # box title unless `title` is set
        script: "./scripts/cpu.sh"
        interval: 2s      # optional, else default_interval
        timeout: 3s       # optional, else default_timeout
        width: 1fr        # optional weight within the row; default equal share
      - name: mem
        script: "free -h"
```

Scripts run via `bash -c` and inherit your environment plus `WIDGET_W`,
`WIDGET_H` (the inner box size), `COLUMNS`, and `LINES`. stdout is the box body
(ANSI colors preserved); a non-zero exit or timeout shows an error indicator and
the stderr tail while keeping the last good output dimmed.
