package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var (
	db         *sql.DB
	rdb        *redis.Client
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
)

func main() {
	port := getenv("PORT", "8084")

	// Init envs
	jwtSecret = getenv("JWT_SECRET", "change-me-local")
	accessMin, _ := strconv.Atoi(getenv("ACCESS_TOKEN_MIN", "15"))
	refreshDays, _ := strconv.Atoi(getenv("REFRESH_TOKEN_DAYS", "7"))
	accessTTL = time.Duration(accessMin) * time.Minute
	refreshTTL = time.Duration(refreshDays) * 24 * time.Hour

	var err error
	db, err = sql.Open("postgres", getenv("DATABASE_URL", "postgres://ecomm:ecommpass@postgres:5432/ecomm?sslmode=disable"))
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(10)

	rdb = redis.NewClient(&redis.Options{
		Addr: getenv("REDIS_ADDR", "redis:6379"),
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { jsonOK(w, map[string]string{"status": "ok"}) })
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) { jsonOK(w, map[string]string{"status": "ready"}) })

	mux.HandleFunc("/signup", signupHandler)
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/refresh", refreshHandler)
	mux.HandleFunc("/logout", logoutHandler)
	mux.HandleFunc("/me", meHandler)

	// Swagger/OpenAPI
	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(swaggerSpec(getenv("PORT", "8084")))
	})
	mux.HandleFunc("/swagger", swaggerUIHandler)
	mux.HandleFunc("/swagger/", swaggerUIHandler)

	srv := &http.Server{Addr: ":" + port, Handler: mux}
	log.Printf("auth-service listening on :%s", port)
	log.Fatal(srv.ListenAndServe())
}

type creds struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var c creds
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if c.Email == "" || c.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}

	pw, err := hashPassword(c.Password)
	if err != nil {
		http.Error(w, "internal", http.StatusInternalServerError)
		return
	}

	// upsert user
	_, err = db.Exec("INSERT INTO users (email, password_hash, name) VALUES ($1,$2,$3) ON CONFLICT (email) DO NOTHING", c.Email, pw, c.Name)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]string{"status": "created"})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var c creds
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var id string
	var pwHash string
	err := db.QueryRow("SELECT id, password_hash FROM users WHERE email = $1", c.Email).Scan(&id, &pwHash)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if err := checkPassword(pwHash, c.Password); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	access, err := createAccessToken(id)
	if err != nil {
		http.Error(w, "token error", http.StatusInternalServerError)
		return
	}
	refresh, err := createRefreshToken(r.Context(), id)
	if err != nil {
		http.Error(w, "token error", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]string{"access_token": access, "refresh_token": refresh})
}

func refreshHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.RefreshToken == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	uid, err := rdb.Get(r.Context(), "refresh:"+body.RefreshToken).Result()
	if err != nil {
		http.Error(w, "invalid refresh", http.StatusUnauthorized)
		return
	}

	// rotate: delete old, create new
	_ = rdb.Del(r.Context(), "refresh:"+body.RefreshToken)
	access, _ := createAccessToken(uid)
	refresh, _ := createRefreshToken(r.Context(), uid)
	jsonOK(w, map[string]string{"access_token": access, "refresh_token": refresh})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.RefreshToken == "" {
		jsonOK(w, map[string]string{"status": "ok"})
		return
	}
	_ = rdb.Del(r.Context(), "refresh:"+body.RefreshToken)
	jsonOK(w, map[string]string{"status": "logged_out"})
}

func meHandler(w http.ResponseWriter, r *http.Request) {
	// simple introspection: expect Authorization: Bearer <token>
	tok := readBearer(r)
	if tok == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	claims, err := parseAccessToken(tok)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	jsonOK(w, map[string]any{"sub": claims.Subject})
}

func hashPassword(p string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	return string(b), err
}

func checkPassword(hash, pass string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass))
}

func createAccessToken(userID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(accessTTL)),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(jwtSecret))
}

func createRefreshToken(ctx context.Context, userID string) (string, error) {
	id := uuid.NewString()
	err := rdb.Set(ctx, "refresh:"+id, userID, refreshTTL).Err()
	return id, err
}

func parseAccessToken(token string) (*jwt.RegisteredClaims, error) {
	t, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected")
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := t.Claims.(*jwt.RegisteredClaims); ok && t.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid")
}

func readBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && h[:7] == "Bearer " {
		return h[7:]
	}
	return ""
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
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
			"title":       "Auth Service API",
			"version":     "1.0.0",
			"description": "Authentication endpoints for the ecomm platform.",
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
			"/signup": map[string]any{
				"post": map[string]any{
					"summary": "Create a new user",
					"requestBody": map[string]any{
						"required": true,
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{
									"$ref": "#/components/schemas/SignupRequest",
								},
							},
						},
					},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Created",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{"$ref": "#/components/schemas/Status"},
								},
							},
						},
					},
				},
			},
			"/login": map[string]any{
				"post": map[string]any{
					"summary": "Login with email and password",
					"requestBody": map[string]any{
						"required": true,
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{
									"$ref": "#/components/schemas/LoginRequest",
								},
							},
						},
					},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Tokens",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{"$ref": "#/components/schemas/Tokens"},
								},
							},
						},
						"401": map[string]any{"description": "Unauthorized"},
					},
				},
			},
			"/refresh": map[string]any{
				"post": map[string]any{
					"summary": "Refresh tokens",
					"requestBody": map[string]any{
						"required": true,
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{"$ref": "#/components/schemas/RefreshRequest"},
							},
						},
					},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "New tokens",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{"$ref": "#/components/schemas/Tokens"},
								},
							},
						},
					},
				},
			},
			"/logout": map[string]any{
				"post": map[string]any{
					"summary": "Logout and revoke refresh token",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Logged out",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{"$ref": "#/components/schemas/Status"},
								},
							},
						},
					},
				},
			},
			"/me": map[string]any{
				"get": map[string]any{
					"summary":  "Introspect access token",
					"security": []map[string]any{{"bearerAuth": []any{}}},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Subject info",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{"$ref": "#/components/schemas/Subject"},
								},
							},
						},
						"401": map[string]any{"description": "Unauthorized"},
					},
				},
			},
		},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"bearerAuth": map[string]any{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
			"schemas": map[string]any{
				"Status": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"status": map[string]any{"type": "string"},
					},
				},
				"SignupRequest": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"email":    map[string]any{"type": "string", "format": "email"},
						"password": map[string]any{"type": "string"},
						"name":     map[string]any{"type": "string"},
					},
					"required": []string{"email", "password"},
				},
				"LoginRequest": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"email":    map[string]any{"type": "string", "format": "email"},
						"password": map[string]any{"type": "string"},
					},
					"required": []string{"email", "password"},
				},
				"RefreshRequest": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"refresh_token": map[string]any{"type": "string"},
					},
					"required": []string{"refresh_token"},
				},
				"Tokens": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"access_token":  map[string]any{"type": "string"},
						"refresh_token": map[string]any{"type": "string"},
					},
				},
				"Subject": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"sub": map[string]any{"type": "string"},
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
  <title>Swagger UI - Auth Service</title>
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
