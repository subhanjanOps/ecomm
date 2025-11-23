package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
)

type PostgresRepository struct {
	db     *sql.DB
	schema string
}

// NewPostgresRepository creates a repo using the provided schema (e.g., "gateway").
// If schema is empty, "public" will be used. Only [a-z_][a-z0-9_]* are allowed to prevent SQL injection.
func NewPostgresRepository(db *sql.DB, schema string) *PostgresRepository {
	if schema == "" {
		schema = "public"
	}
	valid := regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)
	if !valid.MatchString(schema) {
		schema = "public"
	}
	return &PostgresRepository{db: db, schema: schema}
}

func (r *PostgresRepository) table() string { return fmt.Sprintf("%s.gateway_services", r.schema) }

func (r *PostgresRepository) Init() error {
	if _, err := r.db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", r.schema)); err != nil {
		return err
	}
	_, err := r.db.Exec(fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
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
	);`, r.table()))
	return err
}

func (r *PostgresRepository) LoadEnabled(ctx context.Context) ([]*Service, error) {
	q := fmt.Sprintf(`SELECT id, name, COALESCE(description,''), public_prefix, base_url, swagger_url, enabled FROM %s WHERE enabled = TRUE`, r.table())
	rows, err := r.db.QueryContext(ctx, q)
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
	q := fmt.Sprintf(`SELECT id, name, description, public_prefix, base_url, swagger_url, enabled, COALESCE(last_refreshed_at, to_timestamp(0)), COALESCE(last_health_at, to_timestamp(0)), COALESCE(last_status,''), created_at, updated_at FROM %s ORDER BY created_at ASC`, r.table())
	rows, err := r.db.QueryContext(ctx, q)
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
	q := fmt.Sprintf(`SELECT id, name, description, public_prefix, base_url, swagger_url, enabled, COALESCE(swagger_json,'{}'::jsonb), COALESCE(last_refreshed_at, now()), COALESCE(last_health_at, to_timestamp(0)), COALESCE(last_status,''), created_at, updated_at FROM %s WHERE id = $1`, r.table())
	row := r.db.QueryRowContext(ctx, q, id)
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
	var jsonParam any
	if s.SwaggerJSON != nil {
		raw, _ = json.Marshal(s.SwaggerJSON)
		// use string for JSONB parameter to avoid passing bytea which can confuse the driver
		jsonParam = string(raw)
	} else {
		jsonParam = nil
	}
	q := fmt.Sprintf(`INSERT INTO %s (id, name, description, public_prefix, base_url, swagger_url, enabled, swagger_json, last_refreshed_at, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9, now(), now())`, r.table())
	_, err := r.db.ExecContext(ctx, q, s.ID, s.Name, s.Description, s.PublicPrefix, s.BaseURL, s.SwaggerURL, s.Enabled, jsonParam, s.LastRefreshed)
	return err
}

func (r *PostgresRepository) Update(ctx context.Context, s *Service) error {
	var raw []byte
	var jsonParam any
	if s.SwaggerJSON != nil {
		raw, _ = json.Marshal(s.SwaggerJSON)
		jsonParam = string(raw)
	} else {
		jsonParam = nil
	}
	q := fmt.Sprintf(`UPDATE %s SET name=$2, description=$3, public_prefix=$4, base_url=$5, swagger_url=$6, enabled=$7, swagger_json=$8, updated_at=now() WHERE id=$1`, r.table())
	_, err := r.db.ExecContext(ctx, q, s.ID, s.Name, s.Description, s.PublicPrefix, s.BaseURL, s.SwaggerURL, s.Enabled, jsonParam)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	q := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, r.table())
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}
