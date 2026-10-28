package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"integration-guard/internal/batch"
	"integration-guard/internal/validator"
)

type Config struct {
	Name     string                `json:"name"`
	Rules    []validator.FieldRule `json:"rules"`
	MatchKey batch.MatchKey        `json:"match_key,omitempty"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read configuration: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse configuration JSON: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Config{}, fmt.Errorf("configuration must contain exactly one JSON value")
		}
		return Config{}, fmt.Errorf("invalid trailing JSON data: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("configuration name is required")
	}
	if len(c.Rules) == 0 {
		return fmt.Errorf("at least one field rule is required")
	}

	for i, rule := range c.Rules {
		if strings.TrimSpace(rule.SourceField) == "" {
			return fmt.Errorf("rule %d: source_field is required", i+1)
		}
		if strings.TrimSpace(rule.DestinationField) == "" {
			return fmt.Errorf("rule %d: destination_field is required", i+1)
		}
		switch rule.Normalization {
		case "", validator.NormalizeNone, validator.NormalizeTrimLowercase:
		default:
			return fmt.Errorf("rule %d: unsupported normalization %q", i+1, rule.Normalization)
		}
	}
	return nil
}
