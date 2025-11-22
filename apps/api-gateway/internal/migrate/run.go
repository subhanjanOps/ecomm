package migrate

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// simple embedded migrations via in-memory list
var migrations = map[string]string{
	"000_schema.sql":           `CREATE SCHEMA IF NOT EXISTS {{schema}}; CREATE TABLE IF NOT EXISTS {{schema}}.schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ DEFAULT now());`,
	"001_gateway_services.sql": `CREATE TABLE IF NOT EXISTS {{schema}}.gateway_services (id UUID PRIMARY KEY, name TEXT NOT NULL, description TEXT, public_prefix TEXT NOT NULL UNIQUE, base_url TEXT NOT NULL, swagger_url TEXT NOT NULL, enabled BOOLEAN NOT NULL DEFAULT TRUE, swagger_json JSONB, last_refreshed_at TIMESTAMPTZ, last_health_at TIMESTAMPTZ, last_status TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now());`,
}

func Run(db *sql.DB, schema string) error {
	// ensure schema and migrations table
	if _, err := db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)); err != nil {
		return err
	}
	if _, err := db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ DEFAULT now())", schema)); err != nil {
		return err
	}
	// get applied
	applied := map[string]bool{}
	rows, err := db.Query(fmt.Sprintf("SELECT version FROM %s.schema_migrations", schema))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		_ = rows.Scan(&v)
		applied[v] = true
	}
	// sort keys
	keys := make([]string, 0, len(migrations))
	for k := range migrations {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// apply
	for _, k := range keys {
		if applied[k] {
			continue
		}
		sqlText := strings.ReplaceAll(migrations[k], "{{schema}}", schema)
		if _, err := db.Exec(sqlText); err != nil {
			return fmt.Errorf("apply %s: %w", k, err)
		}
		if _, err := db.Exec(fmt.Sprintf("INSERT INTO %s.schema_migrations (version) VALUES ($1)", schema), k); err != nil {
			return err
		}
	}
	return nil
}
