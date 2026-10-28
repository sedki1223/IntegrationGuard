package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
	return path
}

func testConfig(t *testing.T, dir string) string {
	t.Helper()

	return writeFixture(t, dir, "config.json", `{
		"name": "Test Integration",
		"match_key": {
			"source_field": "id",
			"destination_field": "code"
		},
		"rules": [
			{
				"source_field": "id",
				"destination_field": "code",
				"required": true
			},
			{
				"source_field": "email",
				"destination_field": "emailAddress",
				"required": true
			}
		]
	}`)
}

func TestRunWritesSuccessfulReport(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(t, dir)
	source := writeFixture(t, dir, "source.json",
		`[{"id":"A1","email":"user@example.com"}]`)
	destination := writeFixture(t, dir, "destination.json",
		`[{"code":"A1","emailAddress":"user@example.com"}]`)
	reportPath := filepath.Join(dir, "reports", "success.json")

	if err := run(cfg, source, destination, reportPath); err != nil {
		t.Fatalf("expected successful validation, got: %v", err)
	}

	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}

	var report map[string]any
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("report is not valid JSON: %v", err)
	}

	if report["integration"] != "Test Integration" {
		t.Fatalf("unexpected integration name: %v", report["integration"])
	}

	result, ok := report["result"].(map[string]any)
	if !ok {
		t.Fatalf("report result missing or invalid: %T", report["result"])
	}
	if result["valid"] != true {
		t.Fatalf("expected valid report, got: %v", result["valid"])
	}
}

func TestRunWritesFailureReportAndReturnsValidationError(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(t, dir)
	source := writeFixture(t, dir, "source.json",
		`[{"id":"A1","email":"source@example.com"}]`)
	destination := writeFixture(t, dir, "destination.json",
		`[{"code":"A1","emailAddress":"destination@example.com"}]`)
	reportPath := filepath.Join(dir, "reports", "failure.json")

	err := run(cfg, source, destination, reportPath)
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got: %v", err)
	}

	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("failure report was not written: %v", err)
	}

	report := string(data)
	for _, expected := range []string{
		`"valid": false`,
		`"value_mismatch"`,
		`"expected": "source@example.com"`,
		`"actual": "destination@example.com"`,
	} {
		if !strings.Contains(report, expected) {
			t.Errorf("failure report missing %q", expected)
		}
	}
}

func TestRunTreatsMalformedInputAsOperationalError(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(t, dir)
	source := writeFixture(t, dir, "source.json", `{invalid JSON`)
	destination := writeFixture(t, dir, "destination.json",
		`{"code":"A1","emailAddress":"user@example.com"}`)

	err := run(cfg, source, destination, "")
	var validationErr *ValidationError

	if err == nil {
		t.Fatal("expected an error for malformed input")
	}
	if errors.As(err, &validationErr) {
		t.Fatalf("malformed input must not be classified as a validation failure")
	}
}
