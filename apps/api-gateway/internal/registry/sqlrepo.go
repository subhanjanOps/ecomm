package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type SQLRepository struct {
	db     *sql.DB
	schema string
}

// NewSQLRepository creates a repository using the provided schema (e.g., "gateway").
// If schema is empty, "public" will be used. Only [a-z_][a-z0-9_]* are allowed to prevent SQL injection.
func NewSQLRepository(db *sql.DB, schema string) *SQLRepository {
	if schema == "" {
		schema = "public"
	}
	valid := regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)
	if !valid.MatchString(schema) {
		schema = "public"
	}
	return &SQLRepository{db: db, schema: schema}
}

func (r *SQLRepository) table() string { return fmt.Sprintf("%s.gateway_services", r.schema) }

func (r *SQLRepository) Init() error {
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
	  protocol TEXT NOT NULL DEFAULT 'http',
	  grpc_target TEXT,
	  enabled BOOLEAN NOT NULL DEFAULT TRUE,
	  swagger_json JSONB,
	  last_refreshed_at TIMESTAMPTZ,
	  last_health_at TIMESTAMPTZ,
	  last_status TEXT,
	  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);`, r.table()))
	if err != nil {
		return err
	}
	// Ensure new columns exist on older tables
	if _, err := r.db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS protocol TEXT NOT NULL DEFAULT 'http'`, r.table())); err != nil {
		return err
	}
	if _, err := r.db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS grpc_target TEXT`, r.table())); err != nil {
		return err
	}
	// Routes table
	if _, err := r.db.Exec(fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s.gateway_routes (
	  id UUID PRIMARY KEY,
	  service_id UUID NOT NULL REFERENCES %s.gateway_services(id) ON DELETE CASCADE,
	  method TEXT NOT NULL,
	  path_pattern TEXT NOT NULL,
	  grpc_method TEXT NOT NULL,
	  query_mapping JSONB,
	  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	  UNIQUE(service_id, method, path_pattern)
	);`, r.schema, r.schema)); err != nil {
		return err
	}
	return nil
}

// --- Route methods ---

func (r *SQLRepository) ListRoutes(ctx context.Context, serviceID string) ([]*Route, error) {
	q := fmt.Sprintf(`SELECT id, service_id, method, path_pattern, grpc_method, COALESCE(query_mapping,'{}'::jsonb), created_at, updated_at FROM %s.gateway_routes WHERE service_id = $1 ORDER BY path_pattern ASC`, r.schema)
	rows, err := r.db.QueryContext(ctx, q, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Route
	for rows.Next() {
		var rt Route
		var qm json.RawMessage
		if err := rows.Scan(&rt.ID, &rt.ServiceID, &rt.Method, &rt.Path, &rt.GRPCMethod, &qm, &rt.CreatedAt, &rt.UpdatedAt); err != nil {
			return nil, err
		}
		if len(qm) > 0 {
			var m RouteQueryMapping
			_ = json.Unmarshal(qm, &m)
			rt.QueryMapping = m
		}
		list = append(list, &rt)
	}
	return list, nil
}

func (r *SQLRepository) GetRoute(ctx context.Context, serviceID, routeID string) (*Route, error) {
	q := fmt.Sprintf(`SELECT id, service_id, method, path_pattern, grpc_method, COALESCE(query_mapping,'{}'::jsonb), created_at, updated_at FROM %s.gateway_routes WHERE service_id=$1 AND id=$2`, r.schema)
	row := r.db.QueryRowContext(ctx, q, serviceID, routeID)
	var rt Route
	var qm json.RawMessage
	if err := row.Scan(&rt.ID, &rt.ServiceID, &rt.Method, &rt.Path, &rt.GRPCMethod, &qm, &rt.CreatedAt, &rt.UpdatedAt); err != nil {
		return nil, err
	}
	if len(qm) > 0 {
		var m RouteQueryMapping
		_ = json.Unmarshal(qm, &m)
		rt.QueryMapping = m
	}
	return &rt, nil
}

func (r *SQLRepository) CreateRoute(ctx context.Context, rt *Route) error {
	var qm any
	if rt.QueryMapping != nil {
		if b, err := json.Marshal(rt.QueryMapping); err == nil {
			qm = string(b)
		}
	}
	q := fmt.Sprintf(`INSERT INTO %s.gateway_routes (id, service_id, method, path_pattern, grpc_method, query_mapping) VALUES ($1,$2,$3,$4,$5,$6)`, r.schema)
	_, err := r.db.ExecContext(ctx, q, rt.ID, rt.ServiceID, strings.ToUpper(rt.Method), rt.Path, rt.GRPCMethod, qm)
	return err
}

func (r *SQLRepository) UpdateRoute(ctx context.Context, rt *Route) error {
	var qm any
	if rt.QueryMapping != nil {
		if b, err := json.Marshal(rt.QueryMapping); err == nil {
			qm = string(b)
		}
	}
	q := fmt.Sprintf(`UPDATE %s.gateway_routes SET method=$3, path_pattern=$4, grpc_method=$5, query_mapping=$6, updated_at=now() WHERE id=$1 AND service_id=$2`, r.schema)
	_, err := r.db.ExecContext(ctx, q, rt.ID, rt.ServiceID, strings.ToUpper(rt.Method), rt.Path, rt.GRPCMethod, qm)
	return err
}

func (r *SQLRepository) DeleteRoute(ctx context.Context, serviceID, routeID string) error {
	q := fmt.Sprintf(`DELETE FROM %s.gateway_routes WHERE service_id=$1 AND id=$2`, r.schema)
	_, err := r.db.ExecContext(ctx, q, serviceID, routeID)
	return err
}

func (r *SQLRepository) FindRoute(ctx context.Context, serviceID, method, path string) (*Route, error) {
	q := fmt.Sprintf(`SELECT id, service_id, method, path_pattern, grpc_method, COALESCE(query_mapping,'{}'::jsonb), created_at, updated_at FROM %s.gateway_routes WHERE service_id=$1 AND method=$2 AND path_pattern=$3`, r.schema)
	row := r.db.QueryRowContext(ctx, q, serviceID, strings.ToUpper(method), path)
	var rt Route
	var qm json.RawMessage
	if err := row.Scan(&rt.ID, &rt.ServiceID, &rt.Method, &rt.Path, &rt.GRPCMethod, &qm, &rt.CreatedAt, &rt.UpdatedAt); err != nil {
		return nil, err
	}
	if len(qm) > 0 {
		var m RouteQueryMapping
		_ = json.Unmarshal(qm, &m)
		rt.QueryMapping = m
	}
	return &rt, nil
}

func (r *SQLRepository) LoadEnabled(ctx context.Context) ([]*Service, error) {
	q := fmt.Sprintf(`SELECT id, name, COALESCE(description,''), public_prefix, base_url, swagger_url, protocol, COALESCE(grpc_target,''), enabled FROM %s WHERE enabled = TRUE`, r.table())
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Service
	for rows.Next() {
		var s Service
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.PublicPrefix, &s.BaseURL, &s.SwaggerURL, &s.Protocol, &s.GRPCTarget, &s.Enabled); err != nil {
			return nil, err
		}
		list = append(list, &s)
	}
	return list, nil
}

func (r *SQLRepository) List(ctx context.Context) ([]*Service, error) {
	q := fmt.Sprintf(`SELECT id, name, description, public_prefix, base_url, swagger_url, protocol, COALESCE(grpc_target,''), enabled, COALESCE(last_refreshed_at, to_timestamp(0)), COALESCE(last_health_at, to_timestamp(0)), COALESCE(last_status,''), created_at, updated_at FROM %s ORDER BY created_at ASC`, r.table())
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Service
	for rows.Next() {
		var s Service
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.PublicPrefix, &s.BaseURL, &s.SwaggerURL, &s.Protocol, &s.GRPCTarget, &s.Enabled, &s.LastRefreshed, &s.LastHealthAt, &s.LastStatus, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &s)
	}
	return list, nil
}

func (r *SQLRepository) Get(ctx context.Context, id string) (*Service, error) {
	q := fmt.Sprintf(`SELECT id, name, description, public_prefix, base_url, swagger_url, protocol, COALESCE(grpc_target,''), enabled, COALESCE(swagger_json,'{}'::jsonb), COALESCE(last_refreshed_at, now()), COALESCE(last_health_at, to_timestamp(0)), COALESCE(last_status,''), created_at, updated_at FROM %s WHERE id = $1`, r.table())
	row := r.db.QueryRowContext(ctx, q, id)
	var s Service
	var raw json.RawMessage
	if err := row.Scan(&s.ID, &s.Name, &s.Description, &s.PublicPrefix, &s.BaseURL, &s.SwaggerURL, &s.Protocol, &s.GRPCTarget, &s.Enabled, &raw, &s.LastRefreshed, &s.LastHealthAt, &s.LastStatus, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	if len(raw) > 0 {
		var v any
		_ = json.Unmarshal(raw, &v)
		s.SwaggerJSON = v
	}
	return &s, nil
}

func (r *SQLRepository) Create(ctx context.Context, s *Service) error {
	var raw []byte
	var jsonParam any
	if s.SwaggerJSON != nil {
		raw, _ = json.Marshal(s.SwaggerJSON)
		jsonParam = string(raw)
	} else {
		jsonParam = nil
	}
	q := fmt.Sprintf(`INSERT INTO %s (id, name, description, public_prefix, base_url, swagger_url, protocol, grpc_target, enabled, swagger_json, last_refreshed_at, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11, now(), now())`, r.table())
	_, err := r.db.ExecContext(ctx, q, s.ID, s.Name, s.Description, s.PublicPrefix, s.BaseURL, s.SwaggerURL, s.Protocol, s.GRPCTarget, s.Enabled, jsonParam, s.LastRefreshed)
	return err
}

func (r *SQLRepository) Update(ctx context.Context, s *Service) error {
	var raw []byte
	var jsonParam any
	if s.SwaggerJSON != nil {
		raw, _ = json.Marshal(s.SwaggerJSON)
		jsonParam = string(raw)
	} else {
		jsonParam = nil
	}
	q := fmt.Sprintf(`UPDATE %s SET name=$2, description=$3, public_prefix=$4, base_url=$5, swagger_url=$6, protocol=$7, grpc_target=$8, enabled=$9, swagger_json=$10, updated_at=now() WHERE id=$1`, r.table())
	_, err := r.db.ExecContext(ctx, q, s.ID, s.Name, s.Description, s.PublicPrefix, s.BaseURL, s.SwaggerURL, s.Protocol, s.GRPCTarget, s.Enabled, jsonParam)
	return err
}

func (r *SQLRepository) Delete(ctx context.Context, id string) error {
	q := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, r.table())
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}
