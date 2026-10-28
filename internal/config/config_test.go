package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"integration-guard/internal/validator"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadValidConfiguration(t *testing.T) {
	path := writeConfig(t, `{
		"name": "HR integration",
		"rules": [{
			"source_field": "employee_id",
			"destination_field": "staffCode",
			"required": true,
			"normalization": "none"
		}]
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "HR integration" || len(cfg.Rules) != 1 {
		t.Fatalf("unexpected configuration: %+v", cfg)
	}
}

func TestLoadRejectsMalformedJSON(t *testing.T) {
	_, err := Load(writeConfig(t, `{"name": "broken",`))
	if err == nil || !strings.Contains(err.Error(), "parse configuration JSON") {
		t.Fatalf("expected JSON parsing error, got %v", err)
	}
}

func TestLoadRejectsUnknownProperty(t *testing.T) {
	_, err := Load(writeConfig(t, `{
		"name": "test",
		"unexpected": true,
		"rules": [{"source_field": "id", "destination_field": "id"}]
	}`))
	if err == nil {
		t.Fatal("expected unknown property error")
	}
}

func TestLoadRejectsMissingName(t *testing.T) {
	cfg := Config{Rules: []validator.FieldRule{
		{SourceField: "id", DestinationField: "id"},
	}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing name error")
	}
}

func TestLoadRejectsMissingRules(t *testing.T) {
	if err := (Config{Name: "test"}).Validate(); err == nil {
		t.Fatal("expected missing rules error")
	}
}

func TestLoadRejectsUnsupportedNormalization(t *testing.T) {
	cfg := Config{
		Name: "test",
		Rules: []validator.FieldRule{{
			SourceField:      "email",
			DestinationField: "email",
			Normalization:    "magic_conversion",
		}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected unsupported normalization error")
	}
}

func TestLoadRejectsMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("expected missing file error")
	}
}

func TestLoadRejectsTrailingJSON(t *testing.T) {
	_, err := Load(writeConfig(t, `{
		"name": "test",
		"rules": [{"source_field":"id","destination_field":"id"}]
	} {}`))
	if err == nil {
		t.Fatal("expected trailing JSON to be rejected")
	}
}
