package ssssg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Global GlobalConfig `yaml:"global"`
	Pages  []PageConfig `yaml:"pages"`
}

type GlobalConfig struct {
	Layout string            `yaml:"layout"`
	Data   map[string]any    `yaml:"data"`
	Fetch  map[string]string `yaml:"fetch"`
}

type PageConfig struct {
	Template string            `yaml:"template"`
	Output   string            `yaml:"output"`
	Layout   string            `yaml:"layout"`
	Data     map[string]any    `yaml:"data"`
	Fetch    map[string]string `yaml:"fetch"`
}

var (
	errTemplateRequired    = errors.New("template is required")
	errOutputRequired      = errors.New("output is required")
	errOutputPathTraversal = errors.New("output path must not escape output directory")
)

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	for i, p := range cfg.Pages {
		if p.Template == "" {
			return nil, fmt.Errorf("pages[%d]: %w", i, errTemplateRequired)
		}

		if p.Output == "" {
			return nil, fmt.Errorf("pages[%d]: %w", i, errOutputRequired)
		}

		cleaned := filepath.Clean(p.Output)
		if filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "..") {
			return nil, fmt.Errorf("pages[%d]: %w: %s", i, errOutputPathTraversal, p.Output)
		}
	}

	return &cfg, nil
}
