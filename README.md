# Integration Guard

Integration Guard is a data integration validation tool written in Go. It compares records from source and destination systems to detect transformation errors, missing records, unexpected records, duplicate keys, and field-level mismatches.

The goal is to identify integration defects before they reach production.

## Features

- Configurable source-to-destination field mappings
- Required-field validation
- Explicit string normalization
- Batch record matching using configurable keys
- Type-sensitive matching for supported scalar identifiers
- Rejection of unsupported match-key types
- Detection of missing source and destination match keys
- Detection of duplicate source keys and ambiguous duplicate destination keys
- Detection of missing destination records and unexpected destination records
- Structured field-level diagnostics
- JSON reports written to standard output and optionally to a file
- Distinct exit codes for validation failures and operational errors
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

Generated reports are written to the specified path. Create the report directory first if it does not exist.

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

Supported normalization values are:

- `none`
- `trim_lowercase`

An omitted normalization value uses the default behavior.

The source and destination field names may differ, allowing the validator to compare fields that have been renamed during an integration.

## Command-line options

| Option | Default | Purpose |
|---|---|---|
| `-config` | `testdata/integration-rules.json` | Mapping configuration |
| `-source` | `testdata/source.json` | Source JSON payload |
| `-destination` | `testdata/destination.json` | Destination JSON payload |
| `-report` | Empty | Optional report output path |

Payload files can contain one JSON object or an array of objects.

## Batch matching and reconciliation

When a match key is configured, Integration Guard uses the specified source and destination fields to associate records before validating their mapped fields.

The batch validator reports conditions including:

- Missing source or destination match keys
- Unsupported match-key types
- Duplicate source keys
- Duplicate destination keys that make matching ambiguous
- Source records without a matching destination record
- Destination records without a matching source record
- Field-validation failures within matched records

Supported match keys are scalar identifiers, such as strings, numbers, and booleans. Missing keys, `null` values, arrays, and objects are not accepted as valid identifiers.

Matching is type-sensitive. For example, the string `"427"` and the number `427` are treated as different identifiers. Identifiers with leading zeros should be represented as strings when those zeros are significant.

Duplicate destination keys are treated as ambiguous. Integration Guard does not silently select one of the duplicate records for field validation.

## Reports and record counts

The JSON report includes the integration name, input file paths, validation result, record counts, and detected issues.

The batch result distinguishes these counts:

- `source_records`: number of source records supplied
- `destination_records`: number of destination records supplied
- `matched_records`: source records matched to an unambiguous destination record and evaluated by the field validator
- `passed_records`: matched source records that passed field validation
- `failed_records`: source records that could not be validated successfully

**`failed_records` is not the total number of reported issues.** Unexpected destination records are reported as issues but do not increase the failed source-record count.

## Exit behavior

| Exit code | Meaning |
|---|---|
| `0` | Validation passed |
| `1` | Validation completed but found invalid data |
| `2` | Operational error, such as invalid configuration or unreadable input |

A failed validation is not the same as a tool error: the former means the comparison completed and found defects.

## Run tests

Run all Go tests:

```bash
go test ./...
```

Check Go formatting:

```bash
gofmt -l ./cmd ./internal
```

The formatting command should produce no output when all Go files are formatted correctly.

GitHub Actions checks formatting, runs the test suite, verifies the successful demo, and verifies that the intentionally invalid demo is rejected with the expected validation errors.

## Architecture

- `cmd/integration-guard`: CLI, input loading, report generation, and process exit behavior
- `internal/config`: configuration loading and validation
- `internal/validator`: field-level comparison and normalization
- `internal/batch`: matching and reconciliation across record collections
- `testdata`: valid and intentionally invalid sample integrations
- `docs/incidents`: documented integration incident scenarios

## Incident case study

The repository includes a simulated HR-to-Payroll incident case study documenting a department mismatch, a missing destination record, and an unexpected destination record.

The case study describes investigation steps, possible causes, remediation options, and regression testing. It does not claim a confirmed root cause or verified remediation because the scenario uses test fixtures rather than a live production system.

## Current scope and limitations

Integration Guard currently compares JSON payloads supplied as files. It does not connect directly to HR systems, payroll platforms, databases, or message brokers.

It identifies validation defects but does not independently determine their underlying cause, inspect live system logs, repair records, or verify production deployments.

The current scope focuses on the validation engine, batch reconciliation, diagnostic reporting, and automated verification. External connectors and a user interface can be considered after the core behavior is dependable.
