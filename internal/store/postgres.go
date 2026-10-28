package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"integration-guard/internal/config"
)

var ErrNotFound = errors.New("resource not found")

type Integration struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Config    config.Config `json:"config"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type Run struct {
	ID              string          `json:"id"`
	IntegrationID   string          `json:"integration_id"`
	IntegrationName string          `json:"integration_name"`
	Status          string          `json:"status"`
	Report          json.RawMessage `json:"report"`
	CreatedAt       time.Time       `json:"created_at"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Migrate(ctx context.Context) error {
	const migration = `
CREATE TABLE IF NOT EXISTS integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    config JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS integrations_updated_at_idx ON integrations (updated_at DESC);

CREATE TABLE IF NOT EXISTS validation_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    integration_id UUID NOT NULL REFERENCES integrations(id) ON DELETE RESTRICT,
    integration_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('passed', 'failed')),
    config_snapshot JSONB NOT NULL,
    report JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS validation_runs_created_at_idx ON validation_runs (created_at DESC);
CREATE INDEX IF NOT EXISTS validation_runs_integration_idx ON validation_runs (integration_id, created_at DESC);
`
	_, err := r.pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pgcrypto`)
	if err != nil {
		return fmt.Errorf("enable pgcrypto: %w", err)
	}
	if _, err := r.pool.Exec(ctx, migration); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

func (r *Repository) ListIntegrations(ctx context.Context) ([]Integration, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id::text, name, config, created_at, updated_at
FROM integrations ORDER BY updated_at DESC, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Integration, 0)
	for rows.Next() {
		var item Integration
		var raw []byte
		if err := rows.Scan(&item.ID, &item.Name, &raw, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &item.Config); err != nil {
			return nil, fmt.Errorf("decode integration config: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateIntegration(ctx context.Context, cfg config.Config) (Integration, error) {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return Integration{}, err
	}
	var item Integration
	var stored []byte
	err = r.pool.QueryRow(ctx, `
INSERT INTO integrations (name, config)
VALUES ($1, $2::jsonb)
RETURNING id::text, name, config, created_at, updated_at`, cfg.Name, raw).
		Scan(&item.ID, &item.Name, &stored, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return Integration{}, err
	}
	if err := json.Unmarshal(stored, &item.Config); err != nil {
		return Integration{}, err
	}
	return item, nil
}

func (r *Repository) GetIntegration(ctx context.Context, id string) (Integration, error) {
	var item Integration
	var raw []byte
	err := r.pool.QueryRow(ctx, `
SELECT id::text, name, config, created_at, updated_at
FROM integrations WHERE id = $1::uuid`, id).
		Scan(&item.ID, &item.Name, &raw, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Integration{}, ErrNotFound
	}
	if err != nil {
		return Integration{}, err
	}
	if err := json.Unmarshal(raw, &item.Config); err != nil {
		return Integration{}, err
	}
	return item, nil
}

func (r *Repository) UpdateIntegration(ctx context.Context, id string, cfg config.Config) (Integration, error) {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return Integration{}, err
	}
	var item Integration
	var stored []byte
	err = r.pool.QueryRow(ctx, `
UPDATE integrations
SET name = $2, config = $3::jsonb, updated_at = now()
WHERE id = $1::uuid
RETURNING id::text, name, config, created_at, updated_at`, id, cfg.Name, raw).
		Scan(&item.ID, &item.Name, &stored, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Integration{}, ErrNotFound
	}
	if err != nil {
		return Integration{}, err
	}
	if err := json.Unmarshal(stored, &item.Config); err != nil {
		return Integration{}, err
	}
	return item, nil
}

func (r *Repository) DeleteIntegration(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM integrations WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) CreateRun(ctx context.Context, integration Integration, status string, report any) (Run, error) {
	configJSON, err := json.Marshal(integration.Config)
	if err != nil {
		return Run{}, err
	}
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return Run{}, err
	}
	var item Run
	var stored []byte
	err = r.pool.QueryRow(ctx, `
INSERT INTO validation_runs (integration_id, integration_name, status, config_snapshot, report)
VALUES ($1::uuid, $2, $3, $4::jsonb, $5::jsonb)
RETURNING id::text, integration_id::text, integration_name, status, report, created_at`,
		integration.ID, integration.Name, status, configJSON, reportJSON).
		Scan(&item.ID, &item.IntegrationID, &item.IntegrationName, &item.Status, &stored, &item.CreatedAt)
	if err != nil {
		return Run{}, err
	}
	item.Report = json.RawMessage(stored)
	return item, nil
}

func (r *Repository) ListRuns(ctx context.Context, limit int) ([]Run, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id::text, integration_id::text, integration_name, status, report, created_at
FROM validation_runs
ORDER BY created_at DESC
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Run, 0)
	for rows.Next() {
		var item Run
		var raw []byte
		if err := rows.Scan(&item.ID, &item.IntegrationID, &item.IntegrationName, &item.Status, &raw, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Report = json.RawMessage(raw)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetRun(ctx context.Context, id string) (Run, error) {
	var item Run
	var raw []byte
	err := r.pool.QueryRow(ctx, `
SELECT id::text, integration_id::text, integration_name, status, report, created_at
FROM validation_runs WHERE id = $1::uuid`, id).
		Scan(&item.ID, &item.IntegrationID, &item.IntegrationName, &item.Status, &raw, &item.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Run{}, ErrNotFound
	}
	if err != nil {
		return Run{}, err
	}
	item.Report = json.RawMessage(raw)
	return item, nil
}
