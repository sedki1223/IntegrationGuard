package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"integration-guard/internal/batch"
	"integration-guard/internal/config"
	"integration-guard/internal/validator"
)

type Report struct {
	Integration     string `json:"integration"`
	SourceFile      string `json:"source_file"`
	DestinationFile string `json:"destination_file"`
	Result          any    `json:"result"`
}

type ValidationError struct{}

func (e *ValidationError) Error() string {
	return "validation failed; report contains the detected issues"
}

func main() {
	configPath := flag.String("config", "testdata/integration-rules.json", "mapping configuration JSON")
	sourcePath := flag.String("source", "testdata/source.json", "source payload JSON")
	destinationPath := flag.String("destination", "testdata/destination.json", "destination payload JSON")
	reportPath := flag.String("report", "", "optional path to save the JSON report")
	flag.Parse()

	if err := run(*configPath, *sourcePath, *destinationPath, *reportPath); err != nil {
		fmt.Fprintln(os.Stderr, "Integration Guard:", err)

		var validationErr *ValidationError
		if errors.As(err, &validationErr) {
			os.Exit(1)
		}
		os.Exit(2)
	}
}

func run(configPath, sourcePath, destinationPath, reportPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	source, err := loadRecords(sourcePath)
	if err != nil {
		return fmt.Errorf("load source payload: %w", err)
	}

	destination, err := loadRecords(destinationPath)
	if err != nil {
		return fmt.Errorf("load destination payload: %w", err)
	}

	var result any
	var valid bool

	if cfg.MatchKey.SourceField != "" || cfg.MatchKey.DestinationField != "" {
		batchResult := batch.Compare(source, destination, cfg.MatchKey, cfg.Rules)
		result = batchResult
		valid = batchResult.Valid
	} else {
		if len(source) != 1 || len(destination) != 1 {
			return fmt.Errorf("multiple records require match_key.source_field and match_key.destination_field in configuration")
		}

		singleResult := validator.Compare(source[0], destination[0], cfg.Rules)
		result = singleResult
		valid = singleResult.Valid
	}

	report := Report{
		Integration:     cfg.Name,
		SourceFile:      sourcePath,
		DestinationFile: destinationPath,
		Result:          result,
	}

	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	reportJSON = append(reportJSON, '\n')

	if reportPath != "" {
		if err := os.MkdirAll(filepath.Dir(reportPath), 0755); err != nil {
			return fmt.Errorf("create report directory: %w", err)
		}
		if err := os.WriteFile(reportPath, reportJSON, 0644); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
	}

	if _, err := os.Stdout.Write(reportJSON); err != nil {
		return fmt.Errorf("write report to stdout: %w", err)
	}

	if !valid {
		return &ValidationError{}
	}

	return nil
}

func loadRecords(path string) ([]validator.Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse %q: %w", path, err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("%q must contain exactly one JSON value", path)
		}
		return nil, fmt.Errorf("invalid trailing JSON in %q: %w", path, err)
	}

	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("%q is empty", path)
	}

	switch trimmed[0] {
	case '{':
		var record validator.Record
		if err := json.Unmarshal(trimmed, &record); err != nil {
			return nil, fmt.Errorf("decode record in %q: %w", path, err)
		}
		return []validator.Record{record}, nil

	case '[':
		var records []validator.Record
		if err := json.Unmarshal(trimmed, &records); err != nil {
			return nil, fmt.Errorf("decode records in %q: %w", path, err)
		}
		if records == nil {
			records = []validator.Record{}
		}
		return records, nil

	default:
		return nil, fmt.Errorf("%q must contain a JSON object or array of objects", path)
	}
}
