package config

import "testing"

func TestParseColorNames(t *testing.T) {
	cases := map[string]string{
		"black": "0", "red": "1", "green": "2", "yellow": "3",
		"blue": "4", "magenta": "5", "cyan": "6", "white": "7",
		"bright-black": "8", "gray": "8", "grey": "8",
		"bright-red": "9", "bright-white": "15",
	}
	for in, want := range cases {
		got, err := parseColor(in, "7")
		if err != nil {
			t.Errorf("parseColor(%q) error: %v", in, err)
		}
		if got != want {
			t.Errorf("parseColor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseColorHexAnd256(t *testing.T) {
	for _, in := range []string{"#fff", "#ff8800", "#FF8800", "0", "255", "33"} {
		got, err := parseColor(in, "7")
		if err != nil {
			t.Errorf("parseColor(%q) error: %v", in, err)
		}
		if got != in {
			t.Errorf("parseColor(%q) = %q, want passthrough", in, got)
		}
	}
}

func TestParseColorEmptyReturnsDefault(t *testing.T) {
	got, err := parseColor("", "4")
	if err != nil || got != "4" {
		t.Fatalf("parseColor(\"\", \"4\") = %q, %v", got, err)
	}
}

func TestParseColorInvalid(t *testing.T) {
	for _, in := range []string{"reed", "256", "-1", "#gg0000", "#ff", "ff8800"} {
		if _, err := parseColor(in, "7"); err == nil {
			t.Errorf("parseColor(%q) expected error, got nil", in)
		}
	}
}
