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

# Start API Server in the background (Compatible with Windows PowerShell 5.1)
$apiJob = Start-Job -Name "LatencyOpsAPI" -ScriptBlock {
    param($projectPath)
    
    # Move into the project directory BEFORE running Go
    Set-Location -Path $projectPath

    # Import the environment variables from the parent terminal session
    $env:DATABASE_URL = $using:env:DATABASE_URL
    $env:REDIS_URL = $using:env:REDIS_URL
    $env:APP_PORT = $using:env:APP_PORT
    
    go run ./cmd/api/main.go
} -ArgumentList $PSScriptRoot

Write-Host "🌐 API Server job started (Job Name: LatencyOpsAPI, Job ID: $($apiJob.Id))" -ForegroundColor Yellow
Write-Host "💡 Note: To view live API Server logs, open a new PowerShell tab and run: Receive-Job -Name LatencyOpsAPI -Keep" -ForegroundColor DarkGray
Write-Host "--------------------------------------------------------" -ForegroundColor DarkGray

# Run Worker in the foreground
go run ./cmd/worker/main.go

# Cleanup API background job on shutdown (when you press Ctrl+C)
Write-Host "🛑 Shutting down backend cluster..." -ForegroundColor Yellow
Stop-Job -Job $apiJob
Remove-Job -Job $apiJob
Write-Host "✅ All jobs terminated cleanly." -ForegroundColor Green