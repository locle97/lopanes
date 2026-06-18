# Per-pane border color — design

**Date:** 2026-06-18
**Status:** Approved

## Goal

Let users configure a color per pane to emphasize panes visually. The color
applies to the pane's **border and title only**; the body output keeps its own
ANSI. Each pane defaults to a configurable top-level default, which itself
defaults to **white**.

## Config schema

Two new optional fields, mirroring the existing `default_interval` /
`default_timeout` pattern.

```yaml
default_color: gray        # top-level fallback for all panes (defaults to white)
rows:
  - widgets:
      - {name: cpu, color: cyan, script: ...}   # overrides default
      - {name: mem, script: ...}                # inherits default_color
```

- `config.Config` gains `DefaultColor string`.
- `config.Widget` gains `Color string`.
- `rawConfig` gains `DefaultColor string` (`yaml:"default_color"`).
- `rawWidget` gains `Color string` (`yaml:"color"`).
- **Precedence:** widget `color` → else `default_color` → else `white`.

Both stored values are the *canonical* color string (see below), not the raw
user input, so the render layer can pass them straight to lipgloss.

## Color parsing & validation

New file `internal/config/color.go`:

```go
func parseColor(s, def string) (string, error)
```

Validates at config-load time and returns a canonical, lipgloss-acceptable
string. lipgloss's `Color()` does **not** understand English names like `red`,
so names are mapped to ANSI indices here. Accepted input forms:

- **Names** → mapped to ANSI 0–15:
  - base (0–7): `black, red, green, yellow, blue, magenta, cyan, white`
  - bright (8–15): `bright-black` (alias `gray`/`grey`), `bright-red`,
    `bright-green`, `bright-yellow`, `bright-blue`, `bright-magenta`,
    `bright-cyan`, `bright-white`
  - `white` → `"7"`.
- **ANSI-256:** a bare integer `0`–`255` → stored as-is.
- **Hex:** `#rgb` or `#rrggbb` (case-insensitive) → stored as-is.
- Anything else → validation error, consistent with existing field errors,
  e.g. `rows[0].widgets[1].color: unknown color "reed"`.

`parseColor("", def)` returns `def`. The top-level `default_color` resolves an
empty value to the canonical `white` (`"7"`). Each widget resolves an empty
`color` to the (already-canonical) `default_color`.

## Rendering — `internal/widget`

- `View` gains `Color string` (the canonical value).
- `Theme` gains `Colorize func(s, color string) string`:
  - `DefaultTheme`: `lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(s)`;
    a no-op when `color == ""`.
  - `PlainTheme`: identity — so `--no-color` print mode strips it.
- `FromResult` gains a `color` parameter so the configured color survives every
  re-render (`tui.Update` rebuilds the `View` on each result).
- `Render` wraps the **top border (including the title), the bottom border, and
  each side `│`** in `theme.Colorize(..., v.Color)`. The body cells between the
  bars stay uncolored. lipgloss width measurement is ANSI-aware, so layout is
  unaffected.
- Color is applied **regardless of widget state** (including pending and error).
  The inner `⚠ <label>` indicator remains the sole error signal; the border is
  not recolored on error.

## Wiring

- `tui.New`: set `Color: w.Color` on the initial pending `View`; pass `w.Color`
  into `widget.FromResult` in `Update`.
- `printer.go`: pass `w.Color` into `widget.FromResult`. The existing
  `PlainTheme`-on-`--no-color` branch handles suppression.

## Testing

- `internal/config/color_test.go`: names (base + bright + gray/grey aliases),
  hex (`#rgb`, `#rrggbb`), ANSI-256 (`0`, `255`, reject `256`/`-1`), empty →
  default, invalid name → error.
- `internal/config/config_test.go`: `default_color` resolution and per-widget
  override/inherit precedence; default to `white` when omitted.
- `internal/widget/widget_test.go`: update `Render`/`FromResult` callers for the
  new `color` param; add a case asserting ANSI wraps the border lines and side
  bars but **not** the body, and that `PlainTheme` emits no ANSI.

## Docs & samples

- Add `default_color` and per-widget `color` to `internal/config/default.yaml`
  and `examples/lopanes.yaml`.
- Document the field, accepted formats, and the color-name list in `README.md`.

## Out of scope (YAGNI)

- No row-level color (only global default + per-pane).
- TUI mode keeps colors always-on; it has no `--no-color` path today and that
  is unchanged.
- Error state does not override the configured color.
