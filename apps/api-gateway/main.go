package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type resp map[string]any

// Service represents a backend service managed by the gateway
type Service struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	PublicPrefix  string    `json:"public_prefix"`
	BaseURL       string    `json:"base_url"`
	SwaggerURL    string    `json:"swagger_url"`
	Enabled       bool      `json:"enabled"`
	SwaggerJSON   any       `json:"swagger_json,omitempty"`
	LastRefreshed time.Time `json:"last_refreshed_at,omitempty"`
	LastHealthAt  time.Time `json:"last_health_at,omitempty"`
	LastStatus    string    `json:"last_status,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

var (
	db  *sql.DB
	reg = &registry{byPrefix: map[string]*Service{}}
)

type registry struct {
	mu       sync.RWMutex
	byPrefix map[string]*Service
	// sorted prefixes by length desc for longest-prefix match
	order []string
}

func (r *registry) set(services []*Service) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byPrefix = map[string]*Service{}
	for _, s := range services {
		if s.Enabled {
			r.byPrefix[s.PublicPrefix] = s
		}
	}
	r.order = make([]string, 0, len(r.byPrefix))
	for p := range r.byPrefix {
		r.order = append(r.order, p)
	}
	sort.Slice(r.order, func(i, j int) bool { return len(r.order[i]) > len(r.order[j]) })
}

func (r *registry) match(path string) (*Service, string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.order {
		if strings.HasPrefix(path, p) {
			remainder := strings.TrimPrefix(path, p)
			if !strings.HasPrefix(remainder, "/") {
				remainder = "/" + remainder
			}
			return r.byPrefix[p], remainder, true
		}
	}
	return nil, "", false
}

func main() {
	port := getenv("PORT", "8080")

	// DB setup (Postgres)
	var err error
	dsn := getenv("DATABASE_URL", "")
	if dsn != "" {
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			log.Fatalf("db open: %v", err)
		}
		if err := initDB(); err != nil {
			log.Fatalf("db init: %v", err)
		}
		if err := loadRegistry(); err != nil {
			log.Printf("warn: load registry failed: %v", err)
		}
	} else {
		log.Printf("warning: DATABASE_URL not set; admin registry will be in-memory only")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp{"status": "ok"})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp{"status": "ready"})
	})

	// Dynamic reverse proxy for registered services (prefix-based)
	mux.HandleFunc("/api/", proxyDynamic)

	// Admin API (Phase 1 MVP)
	mux.HandleFunc("/admin/services", adminServicesHandler)
	mux.HandleFunc("/admin/services/", adminServiceByIDHandler)

	// Swagger/OpenAPI for gateway itself
	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(swaggerSpec(port))
	})
	// Proxied specs for downstream services (avoids CORS in browser)
	mux.HandleFunc("/specs/user", proxySwagger("user-service", "8081"))
	mux.HandleFunc("/specs/catalog", proxySwagger("catalog-service", "8082"))
	mux.HandleFunc("/specs/orders", proxySwagger("orders-service", "8083"))
	mux.HandleFunc("/specs/auth", proxySwagger("auth-service", "8084"))

	mux.HandleFunc("/swagger", swaggerUIHandler)
	mux.HandleFunc("/swagger/", swaggerUIHandler)

	srv := &http.Server{Addr: ":" + port, Handler: mux}
	log.Printf("api-gateway listening on :%s", port)
	log.Fatal(srv.ListenAndServe())
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// swaggerSpec returns a minimal OpenAPI 3.0 spec for this service.
func swaggerSpec(port string) map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "API Gateway",
			"version":     "1.0.0",
			"description": "Gateway surface endpoints and health checks.",
		},
		"servers": []map[string]any{{"url": "http://localhost:" + port}},
		"paths": map[string]any{
			"/healthz": map[string]any{
				"get": map[string]any{
					"summary":   "Liveness probe",
					"responses": map[string]any{"200": map[string]any{"description": "OK"}},
				},
			},
			"/readyz": map[string]any{
				"get": map[string]any{
					"summary":   "Readiness probe",
					"responses": map[string]any{"200": map[string]any{"description": "Ready"}},
				},
			},
		},
	}
}

func swaggerUIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <title>Swagger UI - API Gateway</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <style>body { margin: 0; }</style>
  </head>
<body>
  <div id="swagger-ui"></div>
	<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
	<script>
		window.onload = function () {
			window.ui = SwaggerUIBundle({
				url: '/swagger.json',
				dom_id: '#swagger-ui',
				validatorUrl: null,
				deepLinking: true,
				presets: [SwaggerUIBundle.presets.apis],
				layout: 'BaseLayout'
			});
		};
	</script>
</body>
</html>`))
}

// proxySwagger returns a handler that fetches a swagger.json from a downstream
// service over the internal docker network and returns it to the browser from
// the gateway's origin to avoid CORS issues.
func proxySwagger(serviceHost, port string) http.HandlerFunc {
	client := &http.Client{}
	target := "http://" + serviceHost + ":" + port + "/swagger.json"
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for k, vs := range resp.Header {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}
}

// jsonOK is a small helper to write JSON responses
func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// --- Dynamic Reverse Proxy ---
func proxyDynamic(w http.ResponseWriter, r *http.Request) {
	svc, remainder, ok := reg.match(r.URL.Path)
	if !ok || svc == nil || !svc.Enabled {
		http.NotFound(w, r)
		return
	}
	target, err := url.Parse(svc.BaseURL)
	if err != nil {
		http.Error(w, "bad upstream", http.StatusBadGateway)
		return
	}
	// Build director to rewrite request to upstream
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		// join path correctly
		basePath := strings.TrimSuffix(target.Path, "/")
		upPath := basePath + remainder
		req.URL.Path = upPath
		req.URL.RawPath = upPath
		req.Host = target.Host
		// keep original headers (incl. Authorization)
	}
	rp := &httputil.ReverseProxy{Director: director}
	rp.ServeHTTP(w, r)
}

// --- Admin API Handlers ---
// POST /admin/services  (create); GET /admin/services (list)
func adminServicesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := listServices(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, list)
	case http.MethodPost:
		var body struct {
			Name         string `json:"name"`
			Description  string `json:"description"`
			PublicPrefix string `json:"public_prefix"`
			SwaggerURL   string `json:"swagger_url"`
			BaseURL      string `json:"base_url"`
			Enabled      *bool  `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body.PublicPrefix == "" || body.SwaggerURL == "" {
			http.Error(w, "public_prefix and swagger_url required", http.StatusBadRequest)
			return
		}
		// fetch swagger
		swJSON, inferredBase, err := fetchSwagger(r.Context(), body.SwaggerURL)
		if err != nil {
			http.Error(w, "failed to fetch swagger: "+err.Error(), http.StatusBadGateway)
			return
		}
		base := body.BaseURL
		if base == "" {
			base = inferredBase
		}
		if base == "" {
			http.Error(w, "base_url missing and not derivable from swagger servers", http.StatusBadRequest)
			return
		}
		en := true
		if body.Enabled != nil {
			en = *body.Enabled
		}
		svc := &Service{
			ID:            newUUID(),
			Name:          firstNonEmpty(body.Name, guessNameFromURL(base)),
			Description:   body.Description,
			PublicPrefix:  normalizePrefix(body.PublicPrefix),
			BaseURL:       strings.TrimRight(base, "/"),
			SwaggerURL:    body.SwaggerURL,
			Enabled:       en,
			SwaggerJSON:   swJSON,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			LastRefreshed: time.Now(),
		}
		if err := createService(r.Context(), svc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// reload registry
		_ = loadRegistry()
		jsonOK(w, svc)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

// GET/PUT/DELETE /admin/services/{id} ; POST /admin/services/{id}/refresh via subpath
func adminServiceByIDHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/admin/services/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	// refresh action
	if strings.HasSuffix(id, "/refresh") && r.Method == http.MethodPost {
		id = strings.TrimSuffix(id, "/refresh")
		svc, err := getService(r.Context(), id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		swJSON, inferredBase, err := fetchSwagger(r.Context(), svc.SwaggerURL)
		if err != nil {
			http.Error(w, "failed to fetch swagger: "+err.Error(), http.StatusBadGateway)
			return
		}
		if svc.BaseURL == "" && inferredBase != "" {
			svc.BaseURL = inferredBase
		}
		svc.SwaggerJSON = swJSON
		svc.LastRefreshed = time.Now()
		if err := updateService(r.Context(), svc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = loadRegistry()
		jsonOK(w, svc)
		return
	}
	switch r.Method {
	case http.MethodGet:
		svc, err := getService(r.Context(), id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		jsonOK(w, svc)
	case http.MethodPut:
		var body Service
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		body.ID = id
		if err := updateService(r.Context(), &body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = loadRegistry()
		jsonOK(w, body)
	case http.MethodDelete:
		if err := deleteService(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = loadRegistry()
		jsonOK(w, resp{"deleted": id})
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

// --- Registry persistence (Postgres) ---
func initDB() error {
	_, err := db.Exec(`
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

func loadRegistry() error {
	rows, err := db.Query(`SELECT id, name, COALESCE(description,''), public_prefix, base_url, swagger_url, enabled FROM gateway_services WHERE enabled = TRUE`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var list []*Service
	for rows.Next() {
		var s Service
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.PublicPrefix, &s.BaseURL, &s.SwaggerURL, &s.Enabled); err != nil {
			return err
		}
		list = append(list, &s)
	}
	reg.set(list)
	return nil
}

func listServices(ctx any) ([]*Service, error) {
	if db == nil {
		// no db: return currently loaded
		reg.mu.RLock()
		defer reg.mu.RUnlock()
		var list []*Service
		for _, p := range reg.order {
			list = append(list, reg.byPrefix[p])
		}
		return list, nil
	}
	rows, err := db.Query(`SELECT id, name, description, public_prefix, base_url, swagger_url, enabled, COALESCE(last_refreshed_at, to_timestamp(0)), COALESCE(last_health_at, to_timestamp(0)), COALESCE(last_status,''), created_at, updated_at FROM gateway_services ORDER BY created_at ASC`)
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

func getService(ctx any, id string) (*Service, error) {
	row := db.QueryRow(`SELECT id, name, description, public_prefix, base_url, swagger_url, enabled, COALESCE(swagger_json,'{}'::jsonb), COALESCE(last_refreshed_at, now()), COALESCE(last_health_at, to_timestamp(0)), COALESCE(last_status,''), created_at, updated_at FROM gateway_services WHERE id = $1`, id)
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

func createService(ctx any, s *Service) error {
	var raw []byte
	if s.SwaggerJSON != nil {
		raw, _ = json.Marshal(s.SwaggerJSON)
	}
	_, err := db.Exec(`INSERT INTO gateway_services (id, name, description, public_prefix, base_url, swagger_url, enabled, swagger_json, last_refreshed_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, s.ID, s.Name, s.Description, s.PublicPrefix, s.BaseURL, s.SwaggerURL, s.Enabled, raw, s.LastRefreshed)
	return err
}

func updateService(ctx any, s *Service) error {
	var raw []byte
	if s.SwaggerJSON != nil {
		raw, _ = json.Marshal(s.SwaggerJSON)
	}
	_, err := db.Exec(`UPDATE gateway_services SET name=$2, description=$3, public_prefix=$4, base_url=$5, swagger_url=$6, enabled=$7, swagger_json=$8, updated_at=now() WHERE id=$1`, s.ID, s.Name, s.Description, s.PublicPrefix, s.BaseURL, s.SwaggerURL, s.Enabled, raw)
	return err
}

func deleteService(ctx any, id string) error {
	_, err := db.Exec(`DELETE FROM gateway_services WHERE id = $1`, id)
	return err
}

// --- Helpers ---
func fetchSwagger(ctx any, urlStr string) (any, string, error) {
	req, _ := http.NewRequest(http.MethodGet, urlStr, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, "", fmtError("status %d", resp.StatusCode)
	}
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, "", err
	}
	// infer base url from servers[0].url if present
	base := ""
	if v, ok := m["servers"].([]any); ok && len(v) > 0 {
		if first, ok := v[0].(map[string]any); ok {
			if u, ok := first["url"].(string); ok {
				base = strings.TrimRight(u, "/")
			}
		}
	}
	return m, base, nil
}

func newUUID() string {
	// Simple UUIDv4 compatible using crypto/rand would be ideal; reuse google/uuid would add dep.
	// For now, use a timestamp-based pseudo id for MVP; replace with proper UUID next.
	return time.Now().Format("20060102150405.000000")
}

func normalizePrefix(p string) string {
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if !strings.HasSuffix(p, "/") {
		p = p + "/"
	}
	return p
}

func guessNameFromURL(u string) string {
	if strings.HasPrefix(u, "http") {
		if parsed, err := url.Parse(u); err == nil {
			host := parsed.Hostname()
			if host != "" {
				return host
			}
		}
	}
	return u
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// small printf-style error helper
func fmtError(format string, args ...any) error {
	return &stringError{msg: fmt.Sprintf(format, args...)}
}

type stringError struct{ msg string }

func (e *stringError) Error() string { return e.msg }
