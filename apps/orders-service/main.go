package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type resp map[string]any

func main() {
	port := getenv("PORT", "8083")

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp{"status": "ok"})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp{"status": "ready"})
	})
	mux.HandleFunc("/api/v1/orders", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]resp{})
	})

	// Swagger/OpenAPI
	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(swaggerSpec(port))
	})
	mux.HandleFunc("/swagger", swaggerUIHandler)
	mux.HandleFunc("/swagger/", swaggerUIHandler)

	srv := &http.Server{Addr: ":" + port, Handler: mux}
	log.Printf("orders-service listening on :%s", port)
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
			"title":       "Orders Service API",
			"version":     "1.0.0",
			"description": "Order management endpoints for the ecomm platform.",
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
			"/api/v1/orders": map[string]any{
				"get": map[string]any{
					"summary": "List orders",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "List of orders",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"type":  "array",
										"items": map[string]any{"$ref": "#/components/schemas/Order"},
									},
								},
							},
						},
					},
				},
			},
		},
		"components": map[string]any{
			"schemas": map[string]any{
				"Order": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":          map[string]any{"type": "string", "format": "uuid"},
						"user_id":     map[string]any{"type": "string", "format": "uuid"},
						"status":      map[string]any{"type": "string"},
						"total_cents": map[string]any{"type": "integer"},
						"currency":    map[string]any{"type": "string"},
						"created_at":  map[string]any{"type": "string", "format": "date-time"},
					},
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
  <title>Swagger UI - Orders Service</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <style>body { margin: 0; }</style>
  </head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
	window.ui = SwaggerUIBundle({
	  url: '/swagger.json',
	  dom_id: '#swagger-ui',
	  presets: [SwaggerUIBundle.presets.apis],
	  layout: 'BaseLayout'
	});
  </script>
</body>
</html>`))
}
