Write-Host "Stopping ecomm stack and removing containers/volumes..." -ForegroundColor Cyan
docker compose down -v
Write-Host "Done." -ForegroundColor Green
