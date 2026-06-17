package config

import (
	"testing"
	"time"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		in         string
		wantWeight int
		wantFixed  int
		wantErr    bool
	}{
		{"", 0, 0, false},
		{"1fr", 1, 0, false},
		{"3fr", 3, 0, false},
		{"10", 0, 10, false},
		{"  2fr ", 2, 0, false},
		{"0fr", 0, 0, true},
		{"-1", 0, 0, true},
		{"abc", 0, 0, true},
		{"1.5fr", 0, 0, true},
	}
	for _, tt := range tests {
		w, f, err := parseSize(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseSize(%q) err=%v wantErr=%v", tt.in, err, tt.wantErr)
			continue
		}
		if err == nil && (w != tt.wantWeight || f != tt.wantFixed) {
			t.Errorf("parseSize(%q) = (w=%d,f=%d) want (w=%d,f=%d)", tt.in, w, f, tt.wantWeight, tt.wantFixed)
		}
	}
}

func TestParseWeight(t *testing.T) {
	tests := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"", 1, false},
		{"1fr", 1, false},
		{"4fr", 4, false},
		{"2", 2, false},
		{"0fr", 0, true},
		{"x", 0, true},
	}
	for _, tt := range tests {
		got, err := parseWeight(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseWeight(%q) err=%v wantErr=%v", tt.in, err, tt.wantErr)
			continue
		}
		if err == nil && got != tt.want {
			t.Errorf("parseWeight(%q) = %d want %d", tt.in, got, tt.want)
		}
	}
}

func TestParseDurationDefault(t *testing.T) {
	def := 5 * time.Second
	got, err := parseDurationDefault("", def)
	if err != nil || got != def {
		t.Fatalf("empty: got %v err %v want %v", got, err, def)
	}
	got, err = parseDurationDefault("2s", def)
	if err != nil || got != 2*time.Second {
		t.Fatalf("2s: got %v err %v", got, err)
	}
	if _, err := parseDurationDefault("0s", def); err == nil {
		t.Fatal("0s should error (must be positive)")
	}
	if _, err := parseDurationDefault("nope", def); err == nil {
		t.Fatal("invalid duration should error")
	}
}
