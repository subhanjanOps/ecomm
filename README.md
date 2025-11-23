**Overview**
- **Goal:** Production-ready e-commerce monorepo using Postgres, Redis, Golang microservices, React (Next.js), and Docker; later Kubernetes.
- **Why monorepo:** Easier cross-service changes, shared tooling, and consistent CI/CD while services remain independently deployable via separate Dockerfiles.

**System Architecture**
- **API Gateway (Go):** Single public entry. Centralizes auth, request routing, rate limiting, and observability. Hides internal topology, enabling service evolution without client churn.
- **Microservices (Go):**
  - **User Service:** Accounts, auth, profiles.
  - **Catalog Service:** Products, categories, inventory read model, search hooks.
  - **Orders Service:** Carts, orders, order state machine, checkout orchestration.
  - Future: Payments, Notifications, Search, Recommendation, Reporting.
- **Data Stores:**
  - **Postgres:** Primary source of truth for relational data with ACID guarantees. Strong schema and transactional integrity needed for orders/payments.
  - **Redis:** Low-latency cache for product reads, sessions, idempotency keys, rate limiting. Enables read scaling and smoother spikes.
- **Async (future):** Start with synchronous HTTP between services; evolve to event-driven with Kafka or Redis Streams for decoupling (e.g., inventory reserved, order placed events).
- **Networking:** Internal private network for services and data stores; only gateway exposed publicly. Health/readiness endpoints baked into every service.
- **Scalability:** Stateless services scale horizontally; DB scales read via caching and read replicas later. Compose demonstrates service separation; Kubernetes later adds autoscaling and service meshes.

**Key Decisions & Justifications**
- **Microservices vs monolith:** Bounded contexts (users, catalog, orders) map cleanly to separate services. This enables independent scaling and deployments, while monorepo preserves developer velocity early on.
- **Postgres:** Strong consistency and relational modeling for orders, payments, and inventory. Mature ecosystem (migrations, replication, extensions).
- **Redis:** Performance-critical reads and cross-cutting concerns (sessions, rate limits). Non-blocking addition that protects Postgres from hot paths.
- **API Gateway:** Contracts with clients remain stable as internals change; single place for auth, throttling, observability, and compat headers.
- **Docker Compose now, Kubernetes later:** Compose provides local parity and process isolation. The compose file is structured with networks/healthchecks and hints compatible with K8s (ports, envs, readiness) to ease migration.

**Repository Layout**
- `apps/` – independently deployable services and web app
  - `api-gateway/` – public edge (Go)
  - `user-service/` – users & auth (Go)
  - `catalog-service/` – products & categories (Go)
  - `orders-service/` – carts & orders (Go)
  - `web/` – Next.js app (to be added after backend)
- `db/` – bootstrap schema and seeds
  - `init/` – SQL executed on first Postgres start
- `scripts/` – helper scripts for Windows/macOS/Linux
- `docker-compose.yml` – local microservice cluster with Postgres/Redis

**Database Schema (Initial)**
- `users`: id, email, password_hash, name, created_at, updated_at
- `products`: id, sku, name, description, price_cents, currency, stock, created_at, updated_at
- `orders`: id, user_id, status, total_cents, currency, created_at, updated_at
- `order_items`: id, order_id, product_id, qty, price_cents
- `carts`: id, user_id, created_at, updated_at
- `cart_items`: id, cart_id, product_id, qty

**Local Development**
- Requirements: Docker Desktop, PowerShell (Windows) or bash (Unix).
- Quick start (Windows PowerShell):

```
pwsh -File .\scripts\compose-up.ps1
```

- Services:
  - Adminer: http://localhost:8088 (connect to host `postgres`, user/password from `.env`)
  - Redis: `localhost:6379`
  - API Gateway: http://localhost:8080/healthz | Swagger UI (aggregated): http://localhost:8080/swagger
  - User Service: http://localhost:8081/healthz | Swagger UI: http://localhost:8081/swagger
  - Catalog Service: http://localhost:8082/healthz | Swagger UI: http://localhost:8082/swagger
  - Orders Service: http://localhost:8083/healthz | Swagger UI: http://localhost:8083/swagger
  - Auth Service: http://localhost:8084/healthz | Swagger UI: http://localhost:8084/swagger

Swagger/OpenAPI
- Central entry: API Gateway Swagger UI aggregates all service specs (no CORS issues) and lets you switch between them.
- Each service also exposes its own OpenAPI 3.0 document at `/swagger.json` and a local Swagger UI at `/swagger`.
- Specs include JSON schemas for list endpoints. Expand `swaggerSpec` in each service as APIs evolve.

Generate Gateway Swagger docs
- To regenerate the API Gateway's `docs` package after changing annotations, run the helper script included in `scripts`:

```pwsh
pwsh -File .\scripts\generate-swagger.ps1
```

This installs the `swag` CLI (if needed) and runs `swag init -g cmd/api-gateway/main.go -o ./docs` inside `apps/api-gateway`.
After running the script commit the generated `apps/api-gateway/docs` folder so the generated Swagger UI is available in the container image.

To stop and clean:

```
pwsh -File .\scripts\compose-down.ps1
```

**Kubernetes Migration Notes**
- Compose services map 1:1 to Deployments + Services; Postgres and Redis become StatefulSets.
- Health endpoints (`/healthz`, `/readyz`) already exist for probes.
- Config moves from `.env` to `Secret`/`ConfigMap`.
- Per-service Dockerfiles are production-friendly multi-stage builds.

**Next Steps**
1) Validate DB + Redis start. 2) Flesh out service routes and repositories. 3) Add Next.js app and connect via gateway. 4) Add auth (JWT) and request tracing. 5) Introduce migrations tool and CI. 6) Expand Swagger specs with schemas and examples.
