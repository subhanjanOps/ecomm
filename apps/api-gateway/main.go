package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
)

type resp map[string]any

func main() {
	port := getenv("PORT", "8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp{"status": "ok"})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp{"status": "ready"})
	})

	// Placeholder: later route /api/* to internal services

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
