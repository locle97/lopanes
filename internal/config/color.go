package config

import (
	"fmt"
	"strconv"
	"strings"
)

// colorNames maps friendly color names to their ANSI 0–15 index (as a string).
var colorNames = map[string]string{
	"black": "0", "red": "1", "green": "2", "yellow": "3",
	"blue": "4", "magenta": "5", "cyan": "6", "white": "7",
	"bright-black": "8", "gray": "8", "grey": "8",
	"bright-red": "9", "bright-green": "10", "bright-yellow": "11",
	"bright-blue": "12", "bright-magenta": "13", "bright-cyan": "14",
	"bright-white": "15",
}

// parseColor validates a pane color spec and returns a canonical
// lipgloss-acceptable string (an ANSI index 0–255 or a hex literal). It accepts
// a friendly name, a bare 0–255 integer, or a #rgb / #rrggbb hex value. An empty
// spec returns def.
func parseColor(s, def string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return def, nil
	}
	if code, ok := colorNames[strings.ToLower(s)]; ok {
		return code, nil
	}
	if strings.HasPrefix(s, "#") {
		if isHexColor(s) {
			return s, nil
		}
		return "", fmt.Errorf("invalid hex color %q", s)
	}
	if n, err := strconv.Atoi(s); err == nil {
		if n < 0 || n > 255 {
			return "", fmt.Errorf("color index out of range 0-255: %q", s)
		}
		return s, nil
	}
	return "", fmt.Errorf("unknown color %q", s)
}

// isHexColor reports whether s is #rgb or #rrggbb (case-insensitive).
func isHexColor(s string) bool {
	hex := strings.TrimPrefix(s, "#")
	if len(hex) != 3 && len(hex) != 6 {
		return false
	}
	for _, r := range hex {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

// defaultColor is the canonical fallback when no color is configured anywhere.
const defaultColor = "7" // white
