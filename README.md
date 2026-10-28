# Integration Guard

**Integration Guard** is a data integration validation and reconciliation tool built with Go, PostgreSQL, React, and TypeScript. It compares records across source and destination systems to detect transformation errors, missing records, unexpected records, duplicate keys, and field-level mismatches.

The goal is to identify integration defects before they reach production and provide structured diagnostics that help engineers investigate data quality problems.

## Features

- Configurable source-to-destination field mappings
- Required-field validation and explicit string normalization
- Batch record matching using configurable keys
- Type-sensitive matching for supported scalar identifiers
- Detection of duplicate keys, missing records, and unexpected records
- Structured field-level diagnostics with expected and actual values
- JSON validation reports and record-level outcomes
- Persistent integration configurations and validation history
- REST API for managing integrations and executing validations
- React + TypeScript dashboard for integrations, validation runs, and report inspection
- PostgreSQL persistence
- Docker Compose setup for the complete application
- Automated Go tests and GitHub Actions CI

## Technology stack

| Component | Technology |
|---|---|
| Validation engine and API | Go |
| Frontend | React, TypeScript, Vite |
| Database | PostgreSQL |
| API communication | REST, JSON |
| Frontend web server | Nginx |
| Containerization | Docker, Docker Compose |
| Automated verification | Go tests, GitHub Actions |

## Quick start: run the full application

### Requirements

- Docker Desktop with Docker Compose
- Git, if cloning the repository

### 1. Clone the repository

```bash
git clone https://github.com/sedki1223/IntegrationGuard.git
cd IntegrationGuard
```

### 2. Configure the environment

The application uses `local_dev_change_me` as a default password for local development. For a more explicit local configuration, copy the example environment file.

PowerShell:

```powershell
Copy-Item .env.example .env
```

Change `POSTGRES_PASSWORD` in `.env` before exposing the application beyond your local development environment.

The `.env` file is ignored by Git. Do not commit real credentials.

### 3. Build and start the services

From the repository root:

```bash
docker compose up --build -d
```

Docker Compose starts three services:

- **Frontend:** React dashboard served by Nginx.
- **API:** Go REST API and validation engine.
- **PostgreSQL:** persistent storage for integrations and validation runs.

The API waits for PostgreSQL to become healthy before starting. The frontend is served on port `5173`, the API on port `8080`, and PostgreSQL on port `5432`.

### 4. Open the dashboard

Visit:

**http://localhost:5173**

Use the dashboard to manage integrations, submit source and destination JSON payloads, execute validations, and inspect saved validation runs.

### 5. Verify the API

The health endpoint is available at:

```text
http://localhost:8080/health
```

A successful response contains a service status of `ok`.

### 6. Stop the application

```bash
docker compose down
```

This stops and removes the containers while preserving the named PostgreSQL data volume.

To remove the containers **and permanently delete the local database volume**, use:

```bash
docker compose down -v
```

Use the second command only when you intentionally want to reset the local database.

## Architecture

```text
                 ┌────────────────────────┐
                 │   React + TypeScript   │
                 │       Dashboard       │
                 └───────────┬────────────┘
                             │
                       HTTP / JSON
                             │
                 ┌───────────▼────────────┐
                 │       Go REST API      │
                 │                        │
                 │ Integration management │
                 │ Validation execution   │
                 │ Report retrieval       │
                 └────────┬─────────┬─────┘
                          │         │
                  Validation     Persistence
                    engine           │
                          │    ┌────▼────────┐
                          │    │ PostgreSQL  │
                          │    └─────────────┘
                          │
                 ┌────────▼─────────┐
                 │ Field comparison │
                 │ Batch matching   │
                 │ Reconciliation   │
                 └──────────────────┘
```

The Go validation engine is also available through the original command-line interface. The REST API exposes the engine's validation capabilities to the dashboard and persists integration configurations and execution reports in PostgreSQL.

## REST API

The API uses JSON request and response bodies.

| Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/health` | Check API health |
| `GET` | `/api/v1/integrations` | List integrations |
| `POST` | `/api/v1/integrations` | Create an integration |
| `GET` | `/api/v1/integrations/{id}` | Retrieve an integration |
| `PUT` | `/api/v1/integrations/{id}` | Update an integration |
| `DELETE` | `/api/v1/integrations/{id}` | Delete an integration when permitted |
| `POST` | `/api/v1/integrations/{id}/runs` | Execute a validation |
| `GET` | `/api/v1/runs` | List saved validation runs |
| `GET` | `/api/v1/runs/{id}` | Retrieve a validation run |

The run-list endpoint accepts an optional `limit` query parameter between 1 and 200; the default is 50.

Creating or updating an integration uses the configuration structure described below. Executing a validation submits `source` and `destination` JSON objects or arrays.

A successful validation returns HTTP `200`. A completed validation that detects invalid data returns HTTP `422`, with the saved run and validation report in the response. Invalid requests and operational failures use appropriate HTTP error responses.

The API currently has no authentication or authorization layer. Treat it as a local development application, not a publicly exposed production service.

## Original CLI

The original command-line validator remains available.

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

Create the report directory before writing a report if it does not already exist. Generated reports are local artifacts and are excluded from Git.

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

Supported normalization values:

- `none`
- `trim_lowercase`

An omitted normalization value uses the default behavior.

Source and destination field names may differ, allowing the validator to compare fields renamed during an integration.

## Batch matching and reconciliation

When a match key is configured, Integration Guard associates source and destination records before validating mapped fields.

The batch validator can detect:

- Missing source or destination match keys
- Unsupported match-key types
- Duplicate source keys
- Duplicate destination keys that make matching ambiguous
- Source records without a matching destination record
- Destination records without a matching source record
- Field-validation failures within matched records

Supported match keys are scalar identifiers, such as strings, numbers, and booleans. Missing keys, `null` values, arrays, and objects are not accepted as valid identifiers.

Matching is type-sensitive. For example, the string `"427"` and the number `427` are different identifiers. Identifiers with significant leading zeros should be represented as strings.

Duplicate destination keys are treated as ambiguous. Integration Guard does not silently select one duplicate record for field validation.

## Reports and validation history

The application saves validation runs and their reports in PostgreSQL. The dashboard provides a history of executions and access to individual run details.

A report can identify:

- The integration and run
- Overall validation status
- Source and destination record counts
- Passed and failed records
- Field-level validation issues
- Missing destination records
- Unexpected destination records

For example, a report may identify a `departmentCode` mismatch where the expected value is `"fin"` but the destination contains `"sales"`. It can also distinguish a source record with no destination match from an unexpected destination record with no source match.

The CLI report includes the integration name, input file paths, validation result, record counts, and detected issues.

The batch result distinguishes these counts:

- `source_records`: number of source records supplied
- `destination_records`: number of destination records supplied
- `matched_records`: source records matched to an unambiguous destination record and evaluated by the field validator
- `passed_records`: matched source records that passed field validation
- `failed_records`: source records that could not be validated successfully

`failed_records` is not the total number of reported issues. Unexpected destination records are reported as issues but do not increase the failed source-record count.

## Exit behavior

The CLI uses distinct exit codes:

| Exit code | Meaning |
|---|---|
| `0` | Validation passed |
| `1` | Validation completed but found invalid data |
| `2` | Operational error, such as invalid configuration or unreadable input |

A failed validation is not the same as a tool error: the comparison completed successfully as an operation but found defects in the data.

## Testing and CI

Run all Go tests:

```bash
go test ./...
```

Check Go formatting:

```bash
gofmt -l ./cmd ./internal
```

The formatting command should produce no output when all Go files are formatted correctly.

The GitHub Actions workflow checks Go formatting, runs the test suite, verifies the successful CLI example, and checks that the intentionally invalid example returns the expected validation failure.

The frontend is built as part of the Docker image build:

```bash
docker compose build frontend
```

## Repository structure

```text
.
├── .github/workflows/ci.yml
├── cmd/
│   ├── integration-guard/
│   └── integration-guard-api/
├── docs/
│   └── incidents/
├── frontend/
│   ├── public/
│   └── src/
├── internal/
│   ├── api/
│   ├── batch/
│   ├── config/
│   ├── store/
│   └── validator/
├── migrations/
├── testdata/
├── Dockerfile.api
├── Dockerfile.frontend
├── docker-compose.yml
├── go.mod
└── README.md
```

- `cmd/integration-guard`: CLI, input loading, report generation, and process exit behavior
- `cmd/integration-guard-api`: REST API entry point and database initialization
- `internal/api`: HTTP routes, request handling, and API responses
- `internal/batch`: record matching and reconciliation
- `internal/config`: configuration loading and validation
- `internal/store`: PostgreSQL persistence
- `internal/validator`: field-level comparison and normalization
- `migrations`: database schema migrations
- `frontend`: React + TypeScript dashboard
- `testdata`: valid and intentionally invalid sample integrations
- `docs/incidents`: documented integration incident scenarios

## Incident case study

The repository includes a simulated HR-to-Payroll incident case study documenting a department mismatch, a missing destination record, and an unexpected destination record.

The case study describes investigation steps, possible causes, remediation options, and regression testing. It does not claim a confirmed root cause or verified remediation because the scenario uses test fixtures rather than a live production system.

## Current scope and limitations

Integration Guard validates JSON payloads supplied through the CLI or dashboard. It does not currently connect directly to external HR systems, payroll platforms, databases, or message brokers.

It identifies validation defects but does not independently determine their underlying cause, inspect live system logs, repair records, or verify production deployments.

The application is currently intended for local development and demonstration. Authentication, authorization, production deployment hardening, and external system connectors remain outside the current scope.
