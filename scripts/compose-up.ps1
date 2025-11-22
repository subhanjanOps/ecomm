param(
  [switch]$Build
)

Write-Host "Starting ecomm stack via Docker Compose..." -ForegroundColor Cyan
if ($Build) {
  docker compose up --build -d
} else {
  docker compose up -d
}

Write-Host "Services starting. Check health endpoints and Adminer at http://localhost:8088" -ForegroundColor Green
