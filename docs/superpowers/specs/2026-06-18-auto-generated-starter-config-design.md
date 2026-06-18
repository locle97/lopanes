# Auto-generated starter config

## Problem

A fresh `go install` user who runs `lopanes` from an arbitrary directory hits a
hard error:

```
lopanes: no config found (looked in ./lopanes.yaml, /home/<user>/.config/lopanes/config.yaml)
```

There is nothing to fall back to. The bundled `examples/lopanes.yaml` references
repo-relative script paths (`./examples/scripts/clock.sh`) and only works from
the repo root, so it is no help to an installed user. First-time use is
confusing: the tool tells you what it looked for but not how to proceed.

## Goal

When no config exists, lopanes should generate a working starter config on
disk, tell the user where it landed, and run it — so the first run always shows
a live dashboard instead of an error.

## Behavior

When `resolveConfigPath` is called with no explicit `--config` flag and finds no
config at either default location (`./lopanes.yaml`, then
`~/.config/lopanes/config.yaml`):

1. Write a built-in starter config to `~/.config/lopanes/config.yaml`, creating
   `~/.config/lopanes/` as needed.
2. Print to stderr:
   `lopanes: no config found; wrote a starter config to <path> — edit it to customize.`
3. Load and run that config. Applies to **both** interactive and `--print`
   modes.

This is the only behavioral change. The two unchanged paths:

- **Explicit `--config PATH`**: a missing path still errors
  (`config "<path>": <stat error>`). No generation.
- **Config already present**: if either default location has a config, it is
  used as before. Generation only triggers when *neither* exists.

### Write-failure fallback

If writing the starter file fails (no home directory, read-only filesystem,
permission denied), lopanes does not hard-fail. It falls back to loading the
embedded default **in memory**, prints a warning to stderr
(`lopanes: no config found and could not write starter config (<err>); using built-in default`),
and continues. First run never hard-fails.

## The starter config

Self-contained and portable — inline shell only, no external script files:

```yaml
default_interval: 2s
default_timeout: 5s
rows:
  - height: 1fr
    widgets:
      - {name: clock, title: "Clock", script: "date '+%H:%M:%S'", interval: 1s}
      - {name: uptime, title: "Uptime", script: "uptime"}
  - height: 2fr
    widgets:
      - {name: disk, title: "Disk", script: "df -h / 2>/dev/null | head -n 5", width: 2fr}
      - {name: mem, title: "Memory", script: "free -h 2>/dev/null || vm_stat"}
```

The `mem` widget uses `free -h` with a `vm_stat` fallback so it produces output
on both Linux and macOS.

## Implementation

- **Embed the template.** Store the starter YAML as
  `internal/config/default.yaml` and embed it via `go:embed` into an exported
  `config.DefaultYAML []byte`. Single source of truth: the same bytes are
  written to disk and used for the in-memory fallback, and the parser tests
  validate it.
- **`config.WriteDefault(path string) error`.** Creates the parent directory
  (`os.MkdirAll`), then writes `DefaultYAML` to `path` with mode `0644`. Returns
  an error if the target file already exists (use `os.OpenFile` with
  `O_CREATE|O_EXCL|O_WRONLY`) so generation never clobbers existing content.
- **`main.go` wiring.** `resolveConfigPath` gains the generation branch in the
  no-config case: compute the global target path, call `WriteDefault`, and on
  success return that path (loaded normally via `config.Load`). On write
  failure, signal the caller to parse `config.DefaultYAML` directly instead of
  reading a file. The cleanest shape is for the resolve step to return both the
  resolved path (possibly empty) and the config to use, or for `main` to call a
  helper that returns a loaded `*config.Config`. Exact signature decided during
  implementation; the observable behavior above is what matters.

## Testing

- **`default.yaml` parses.** A test in the config package calls
  `config.Parse(config.DefaultYAML)` and asserts no error and the expected row
  and widget counts.
- **`WriteDefault`.** Writes the file with the embedded content; creates missing
  parent directories; returns an error when the target already exists.
- **Config resolution.** With a temp `HOME` and empty cwd, the no-config path
  generates and returns the global config file. An explicit missing `--config`
  path still errors (generation not triggered).

## Out of scope

- Helper script files (clock.sh/counter.sh) — the starter config is inline-only.
- An explicit `lopanes init` subcommand.
- Overwriting or migrating an existing config.
