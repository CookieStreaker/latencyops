# 1. Load variables from .env if it exists
if (Test-Path .env) {
    Get-Content .env | ForEach-Object {
        $line = $_.Trim()
        if ($line -and -not $line.StartsWith("#")) {
            $name, $value = $line.Split("=", 2)
            [System.Environment]::SetEnvironmentVariable($name.Trim(), $value.Trim(), [System.EnvironmentVariableTarget]::Process)
        }
    }
    Write-Host "✅ Loaded environment variables from .env" -ForegroundColor Green
} else {
    Write-Host "⚠️ Warning: .env file not found in root directory!" -ForegroundColor Yellow
}

# 2. Spin up Redis in Docker
docker-compose up -d
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Docker Compose failed. Please make sure Docker Desktop is open and running." -ForegroundColor Red
    exit 1
}
Write-Host "✅ Redis container running on port 6379" -ForegroundColor Green

# 3. Launch API and Worker concurrently
Write-Host "🚀 Starting LatencyOps Backend Cluster..." -ForegroundColor Cyan

$apiJob = Start-Job -ScriptBlock {
    param($db, $redis, $port)
    $env:DATABASE_URL = $db
    $env:REDIS_URL = $redis
    $env:APP_PORT = $port
    go run ./cmd/api/main.go
} -ArgumentList $env:DATABASE_URL, $env:REDIS_URL, $env:APP_PORT

Write-Host "🌐 API Server job started (Job ID: $($apiJob.Id))" -ForegroundColor Yellow

# Run Worker in the foreground
go run ./cmd/worker/main.go

# Cleanup API background job on shutdown
Stop-Job -Job $apiJob
Remove-Job -Job $apiJob