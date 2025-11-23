Param(
    [string]$GatewayPath = "apps/api-gateway"
)

Write-Host "Installing swag CLI (local user)..."
go install github.com/swaggo/swag/cmd/swag@latest

# Determine where the `swag` binary was installed and run it directly so users don't need to
# modify their PATH. `GOBIN` may be empty which means binaries go to `$(go env GOPATH)/bin`.
$gobin = (go env GOBIN)
if (-not $gobin -or $gobin -eq "") {
    $gobin = Join-Path (go env GOPATH) 'bin'
}

Push-Location $GatewayPath
try {
    Write-Host "Running swag init in $PWD..."
    $swagExe = Join-Path $gobin 'swag.exe'
    if (-not (Test-Path $swagExe)) {
        # On some systems the executable may be named without .exe
        $swagExe = Join-Path $gobin 'swag'
    }
    if (Test-Path $swagExe) {
        & $swagExe init -g cmd/api-gateway/main.go -o ./docs
    } else {
        Write-Host "swag binary not found in $gobin; falling back to 'go run' (no PATH changes required)"
        go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api-gateway/main.go -o ./docs
    }
    Write-Host "Swagger docs generated in $GatewayPath/docs"
} finally {
    Pop-Location
}

Write-Host "Done. Commit the generated files (apps/api-gateway/docs) to the repository if happy."
