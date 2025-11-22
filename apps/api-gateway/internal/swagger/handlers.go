package swagger

import (
	"io"
	"net/http"
)

// Spec returns a minimal OpenAPI 3 doc for the gateway
func Spec(port string) map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info":    map[string]any{"title": "API Gateway", "version": "1.0.0", "description": "Gateway endpoints and health."},
		"servers": []map[string]any{{"url": "http://localhost:" + port}},
		"paths": map[string]any{
			"/healthz": map[string]any{"get": map[string]any{"summary": "Liveness probe", "responses": map[string]any{"200": map[string]any{"description": "OK"}}}},
			"/readyz":  map[string]any{"get": map[string]any{"summary": "Readiness probe", "responses": map[string]any{"200": map[string]any{"description": "Ready"}}}},
		},
	}
}

// UIHandler serves a minimal Swagger UI wired to /swagger.json
func UIHandler(w http.ResponseWriter, r *http.Request) {
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

// ProxySpec proxies a swagger.json from an internal service to avoid CORS (optional)
func ProxySpec(serviceHost, port string) http.HandlerFunc {
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
