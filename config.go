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
	Static StaticConfig `yaml:"static"`
}

type StaticConfig struct {
	Pipelines []PipelineConfig `yaml:"pipelines"`
}

type PipelineConfig struct {
	Match    string   `yaml:"match"`
	Commands []string `yaml:"commands"`
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
	errTemplateRequired     = errors.New("template is required")
	errOutputRequired       = errors.New("output is required")
	errOutputPathTraversal  = errors.New("output path must not escape output directory")
	errPipelineMatchEmpty   = errors.New("pipeline match pattern is required")
	errPipelineNoCommands   = errors.New("pipeline must have at least one command")
	errPipelineInvalidMatch = errors.New("pipeline match pattern is invalid")
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

	for i, p := range cfg.Static.Pipelines {
		if p.Match == "" {
			return nil, fmt.Errorf("static.pipelines[%d]: %w", i, errPipelineMatchEmpty)
		}

		if _, err := filepath.Match(p.Match, ""); err != nil {
			return nil, fmt.Errorf("static.pipelines[%d]: %w: %s", i, errPipelineInvalidMatch, p.Match)
		}

		if len(p.Commands) == 0 {
			return nil, fmt.Errorf("static.pipelines[%d]: %w", i, errPipelineNoCommands)
		}
	}

	return &cfg, nil
}
