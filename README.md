
# Integration Guard

Integration Guard is a data integration validation tool written in Go. It compares records from source and destination systems to detect transformation errors, missing records, unexpected records, and field-level mismatches.

The goal is to identify integration defects before they reach production.

## Features

- Configurable source-to-destination field mappings
- Required-field validation
- Explicit string normalization
- Batch record matching using configurable keys
- Detection of missing, unexpected, and duplicate records
- Structured field-level diagnostics
- JSON reports written to standard output and optionally to a file
- Nonzero process exit status when validation fails
- Automated tests and GitHub Actions CI

## Requirements

- Go version specified in `go.mod`

## Quick start

Run the default successful example:

```bash
go run ./cmd/integration-guard
```

Save the report to a file:

```bash
go run ./cmd/integration-guard \
  -report ./reports/success.json
```

Run the intentionally invalid example:

```bash
go run ./cmd/integration-guard \
  -source ./testdata/source-failure.json \
  -destination ./testdata/destination-failure.json \
  -report ./reports/failure.json
```

The invalid example is expected to exit with status `1`. It demonstrates detection of:

- A field value mismatch
- A source record missing from the destination
- An unexpected destination record

## Configuration

The default configuration is `testdata/integration-rules.json`.

It defines the integration name, matching key, field mappings, required fields, and normalization rules.

Example:

```json
{
  "name": "HR-to-Payroll Integration",
  "match_key": {
    "source_field": "employee_id",
    "destination_field": "staffCode"
  },
  "rules": [
    {
      "source_field": "employee_id",
      "destination_field": "staffCode",
      "required": true,
      "normalization": "none"
    },
    {
      "source_field": "email",
      "destination_field": "emailAddress",
      "required": true,
      "normalization": "trim_lowercase"
    }
  ]
}
```

Supported normalization values are `none` and `trim_lowercase`. An omitted normalization value uses the default behavior.

## Command-line options

| Option | Default | Purpose |
|---|---|---|
| `-config` | `testdata/integration-rules.json` | Mapping configuration |
| `-source` | `testdata/source.json` | Source JSON payload |
| `-destination` | `testdata/destination.json` | Destination JSON payload |
| `-report` | Empty | Optional report output path |

Payload files can contain one JSON object or an array of objects.

## Exit behavior

| Exit code | Meaning |
|---|---|
| `0` | Validation passed |
| `1` | Validation completed but found invalid data |
| `2` | Operational error, such as invalid configuration or unreadable input |

A failed validation is not the same as a tool error: the former means the comparison completed and found defects.

## Run tests

```bash
go test ./...
```

GitHub Actions runs formatting checks, tests, and both demonstration scenarios on pushes and pull requests.

## Architecture

- `cmd/integration-guard`: CLI, input loading, report generation, and process exit behavior
- `internal/config`: configuration loading and validation
- `internal/validator`: field-level comparison and normalization
- `internal/batch`: matching and reconciliation across record collections
- `testdata`: valid and intentionally invalid sample integrations

## Current scope and limitations

Integration Guard currently compares JSON payloads supplied as files. It does not connect directly to HR systems, payroll platforms, databases, or message brokers.

The current scope focuses on the validation engine, batch reconciliation, diagnostic reporting, and automated verification. External connectors and a user interface can be considered after the core behavior is dependable.
