// Package config parses and validates the lopanes YAML configuration into
// typed structs with defaults applied.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultIntervalFallback = 5 * time.Second
	defaultTimeoutFallback  = 10 * time.Second
)

// Config is the fully validated dashboard configuration.
type Config struct {
	DefaultInterval time.Duration
	DefaultTimeout  time.Duration
	Rows            []Row
}

// Row is one full-width band of widgets.
//
// Exactly one of HeightWeight / HeightFixed is non-zero. HeightWeight > 0 means
// the row shares leftover vertical space by weight; HeightFixed > 0 means a
// fixed line count.
type Row struct {
	HeightWeight int
	HeightFixed  int
	Widgets      []Widget
}

// Widget is a single box driven by a shell script.
type Widget struct {
	Name        string
	Title       string // resolved: falls back to Name
	Script      string
	Interval    time.Duration
	Timeout     time.Duration
	WidthWeight int // >= 1
}

type rawConfig struct {
	DefaultInterval string   `yaml:"default_interval"`
	DefaultTimeout  string   `yaml:"default_timeout"`
	Rows            []rawRow `yaml:"rows"`
}

type rawRow struct {
	Height  string      `yaml:"height"`
	Widgets []rawWidget `yaml:"widgets"`
}

type rawWidget struct {
	Name     string `yaml:"name"`
	Title    string `yaml:"title"`
	Script   string `yaml:"script"`
	Interval string `yaml:"interval"`
	Timeout  string `yaml:"timeout"`
	Width    string `yaml:"width"`
}

// Load reads and parses the config file at path.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config: %w", err)
	}
	return Parse(data)
}

// Parse parses raw YAML bytes into a validated Config.
func Parse(data []byte) (Config, error) {
	var raw rawConfig
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&raw); err != nil {
		return Config{}, fmt.Errorf("parsing YAML: %w", err)
	}
	return raw.toConfig()
}

func (r rawConfig) toConfig() (Config, error) {
	di, err := parseDurationDefault(r.DefaultInterval, defaultIntervalFallback)
	if err != nil {
		return Config{}, fmt.Errorf("default_interval: %w", err)
	}
	dt, err := parseDurationDefault(r.DefaultTimeout, defaultTimeoutFallback)
	if err != nil {
		return Config{}, fmt.Errorf("default_timeout: %w", err)
	}
	if len(r.Rows) == 0 {
		return Config{}, errors.New("config has no rows")
	}

	cfg := Config{DefaultInterval: di, DefaultTimeout: dt}
	widgetCount := 0
	for ri, rr := range r.Rows {
		weight, fixed, err := parseSize(rr.Height)
		if err != nil {
			return Config{}, fmt.Errorf("rows[%d].height: %w", ri, err)
		}
		if weight == 0 && fixed == 0 {
			weight = 1 // omitted height defaults to 1fr
		}
		if len(rr.Widgets) == 0 {
			return Config{}, fmt.Errorf("rows[%d] has no widgets", ri)
		}
		row := Row{HeightWeight: weight, HeightFixed: fixed}
		for wi, rw := range rr.Widgets {
			if rw.Name == "" {
				return Config{}, fmt.Errorf("rows[%d].widgets[%d]: name is required", ri, wi)
			}
			if rw.Script == "" {
				return Config{}, fmt.Errorf("rows[%d].widgets[%d] (%s): script is required", ri, wi, rw.Name)
			}
			interval, err := parseDurationDefault(rw.Interval, di)
			if err != nil {
				return Config{}, fmt.Errorf("rows[%d].widgets[%d].interval: %w", ri, wi, err)
			}
			timeout, err := parseDurationDefault(rw.Timeout, dt)
			if err != nil {
				return Config{}, fmt.Errorf("rows[%d].widgets[%d].timeout: %w", ri, wi, err)
			}
			width, err := parseWeight(rw.Width)
			if err != nil {
				return Config{}, fmt.Errorf("rows[%d].widgets[%d].width: %w", ri, wi, err)
			}
			title := rw.Title
			if title == "" {
				title = rw.Name
			}
			row.Widgets = append(row.Widgets, Widget{
				Name:        rw.Name,
				Title:       title,
				Script:      rw.Script,
				Interval:    interval,
				Timeout:     timeout,
				WidthWeight: width,
			})
			widgetCount++
		}
		cfg.Rows = append(cfg.Rows, row)
	}
	if widgetCount == 0 {
		return Config{}, errors.New("config has no widgets")
	}
	return cfg, nil
}
