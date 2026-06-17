package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseSize parses a row height spec. "Nfr" is a weight; a bare positive
// integer is a fixed line count. Empty returns (0, 0, nil) so the caller can
// apply a default.
func parseSize(s string) (weight, fixed int, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, nil
	}
	if strings.HasSuffix(s, "fr") {
		n, convErr := strconv.Atoi(strings.TrimSuffix(s, "fr"))
		if convErr != nil || n <= 0 {
			return 0, 0, fmt.Errorf("invalid weight %q", s)
		}
		return n, 0, nil
	}
	n, convErr := strconv.Atoi(s)
	if convErr != nil || n <= 0 {
		return 0, 0, fmt.Errorf("invalid size %q", s)
	}
	return 0, n, nil
}

// parseWeight parses a widget width spec as a weight. "Nfr" or a bare positive
// integer both yield weight N. Empty defaults to 1.
func parseWeight(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 1, nil
	}
	s = strings.TrimSuffix(s, "fr")
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid width %q", s)
	}
	return n, nil
}

// parseDurationDefault parses a Go duration string, returning def when empty.
// The duration must be strictly positive.
func parseDurationDefault(s string, def time.Duration) (time.Duration, error) {
	if strings.TrimSpace(s) == "" {
		return def, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	if d <= 0 {
		return 0, fmt.Errorf("duration must be positive: %q", s)
	}
	return d, nil
}
