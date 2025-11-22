package registry

import (
	"context"
	"database/sql"
	"encoding/json"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository { return &PostgresRepository{db: db} }

func (r *PostgresRepository) Init() error {
	_, err := r.db.Exec(`
CREATE TABLE IF NOT EXISTS gateway_services (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT,
  public_prefix TEXT NOT NULL UNIQUE,
  base_url TEXT NOT NULL,
  swagger_url TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  swagger_json JSONB,
  last_refreshed_at TIMESTAMPTZ,
  last_health_at TIMESTAMPTZ,
  last_status TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`)
	return err
}

func (r *PostgresRepository) LoadEnabled(ctx context.Context) ([]*Service, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, COALESCE(description,''), public_prefix, base_url, swagger_url, enabled FROM gateway_services WHERE enabled = TRUE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Service
	for rows.Next() {
		var s Service
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.PublicPrefix, &s.BaseURL, &s.SwaggerURL, &s.Enabled); err != nil {
			return nil, err
		}
		list = append(list, &s)
	}
	return list, nil
}

func (r *PostgresRepository) List(ctx context.Context) ([]*Service, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, description, public_prefix, base_url, swagger_url, enabled, COALESCE(last_refreshed_at, to_timestamp(0)), COALESCE(last_health_at, to_timestamp(0)), COALESCE(last_status,''), created_at, updated_at FROM gateway_services ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Service
	for rows.Next() {
		var s Service
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.PublicPrefix, &s.BaseURL, &s.SwaggerURL, &s.Enabled, &s.LastRefreshed, &s.LastHealthAt, &s.LastStatus, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &s)
	}
	return list, nil
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (*Service, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, description, public_prefix, base_url, swagger_url, enabled, COALESCE(swagger_json,'{}'::jsonb), COALESCE(last_refreshed_at, now()), COALESCE(last_health_at, to_timestamp(0)), COALESCE(last_status,''), created_at, updated_at FROM gateway_services WHERE id = $1`, id)
	var s Service
	var raw json.RawMessage
	if err := row.Scan(&s.ID, &s.Name, &s.Description, &s.PublicPrefix, &s.BaseURL, &s.SwaggerURL, &s.Enabled, &raw, &s.LastRefreshed, &s.LastHealthAt, &s.LastStatus, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	if len(raw) > 0 {
		var v any
		_ = json.Unmarshal(raw, &v)
		s.SwaggerJSON = v
	}
	return &s, nil
}

func (r *PostgresRepository) Create(ctx context.Context, s *Service) error {
	var raw []byte
	if s.SwaggerJSON != nil {
		raw, _ = json.Marshal(s.SwaggerJSON)
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO gateway_services (id, name, description, public_prefix, base_url, swagger_url, enabled, swagger_json, last_refreshed_at, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9, now(), now())`, s.ID, s.Name, s.Description, s.PublicPrefix, s.BaseURL, s.SwaggerURL, s.Enabled, raw, s.LastRefreshed)
	return err
}

func (r *PostgresRepository) Update(ctx context.Context, s *Service) error {
	var raw []byte
	if s.SwaggerJSON != nil {
		raw, _ = json.Marshal(s.SwaggerJSON)
	}
	_, err := r.db.ExecContext(ctx, `UPDATE gateway_services SET name=$2, description=$3, public_prefix=$4, base_url=$5, swagger_url=$6, enabled=$7, swagger_json=$8, updated_at=now() WHERE id=$1`, s.ID, s.Name, s.Description, s.PublicPrefix, s.BaseURL, s.SwaggerURL, s.Enabled, raw)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM gateway_services WHERE id = $1`, id)
	return err
}
