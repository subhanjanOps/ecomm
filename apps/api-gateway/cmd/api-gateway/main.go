package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	_ "ecomm/api-gateway/docs"

	_ "github.com/lib/pq"

	"ecomm/api-gateway/internal/app"
	mg "ecomm/api-gateway/internal/migrate"
	"ecomm/api-gateway/internal/registry"

	"github.com/redis/go-redis/v9"
)

// @title Ecomm API Gateway
// @version 1.0
// @description The Ecomm API Gateway provides a unified entrypoint that proxies requests to onboarded backend services.
// The gateway supports service onboarding and lifecycle management via the Admin API, dynamic routing to
// registered backends based on `public_prefix`, background health checks that update service status,
// and an embedded Swagger UI for human-readable API discovery. Admin endpoints require JWT Bearer authentication.
// Services are persisted in Postgres (set `DATABASE_URL`) and stored under the schema configured by
// `GATEWAY_DB_SCHEMA`. On startup the gateway runs embedded migrations to create the necessary schema and tables.
//
// Environment variables of interest:
// - `DATABASE_URL` (required): Postgres connection string used as the source-of-truth for services.
// - `GATEWAY_DB_SCHEMA` (optional, default "gateway"): Postgres schema where gateway tables are stored.
// - `JWT_SECRET` (optional for local dev): HMAC secret used to validate Admin JWT Bearer tokens.
// - `HEALTH_CHECK_SECONDS` (optional): Interval in seconds for background health probes.
//
// @termsOfService https://example.com/terms/
// @contact.name Ecomm Platform Team
// @contact.url https://example.com/support
// @contact.email ops@example.com
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
// @host localhost:8080
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @tag.name admin
// @tag.description Admin operations for onboarding and managing backend services (create, update, delete, refresh)
// @tag.name proxy
// @tag.description Proxy endpoints that forward incoming traffic under `/api/` to onboarded services using
// longest-prefix matching on `public_prefix`.
// @tag.name system
// @tag.description System endpoints for health, readiness, and operational status

func main() {
	port := getenv("PORT", "8080")

	// DB setup (Postgres)
	dsn := getenv("DATABASE_URL", "postgres://ecomm:ecommpass@localhost:5432/ecomm?sslmode=disable")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required for api-gateway")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	schema := getenv("GATEWAY_DB_SCHEMA", "gateway")
	if err := mg.Run(db, schema); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	var repo registry.Repository = registry.NewSQLRepository(db, schema)
	if err := repo.Init(); err != nil {
		log.Fatalf("db init: %v", err)
	}

	// Optional Redis caching layer for repository reads
	if addr := getenv("REDIS_ADDR", ""); addr != "" {
		rdb := redis.NewClient(&redis.Options{Addr: addr})
		repo = registry.NewCachingRepository(repo, rdb, 15*time.Second)
	}

	// Build server via app wiring (hexagonal adapters + ports)
	jwtSecret := getenv("JWT_SECRET", "")
	secStr := getenv("HEALTH_CHECK_SECONDS", "30")
	sec, _ := strconv.Atoi(secStr)
	if sec <= 0 {
		sec = 30
	}
	reg := registry.New()
	srv, err := app.NewServer(app.Options{
		Port:           port,
		Repo:           repo,
		Registry:       reg,
		JWTSecret:      jwtSecret,
		HealthInterval: time.Duration(sec) * time.Second,
	})
	if err != nil {
		log.Fatalf("server init: %v", err)
	}

	log.Printf("api-gateway listening on :%s", port)
	log.Fatal(srv.ListenAndServe())
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
