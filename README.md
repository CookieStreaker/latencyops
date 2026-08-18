# 🚀 LatencyOps — Real-Time API Health & Rate-Limit Monitor

LatencyOps is a high-performance, real-time API health, uptime, and rate-limit alerting monitor built specifically for engineering teams and CTOs. It features a distributed Go worker engine capable of sub-millisecond telemetry and a reactive Next.js dashboard driven by Server-Sent Events (SSE).

**👉 See the [Detailed Benchmark & Performance Report](./BENCHMARK_RESULTS.md) for 10K+ concurrency metrics.**

---

## 🛠️ Tech Stack & Architecture

- **Backend & Worker Engine:** Go (Golang 1.22+)
- **Frontend Dashboard:** Next.js (React) + Tailwind CSS + Lucide Icons
- **Database (Historical Data):** PostgreSQL via `pgxpool` (Supabase)
- **Cache & Message Broker (Real-time Pub/Sub):** Redis 7+ via `go-redis/v9`
- **Streaming Protocol:** Server-Sent Events (SSE) via `fetch` & `ReadableStream`
- **Security:** OWASP Top 10 API Security (Strict SSRF URL validation, BOLA multi-tenant isolation via `WorkspaceID`, payload limits)

### Complete Data Flow

1. **Worker Pool:** A pool of goroutines continuously polls PostgreSQL for active endpoints and executes concurrent HTTP probes with strict timeout budgets and SSRF validation.
2. **Result Pipeline:** Probe results (`PingResult`) are pushed to a buffered channel and handled by a non-blocking 3-stage listener:
   - **Stage 1 (Postgres):** Parameterized `INSERT` for historical tracking.
   - **Stage 2 (Redis Pub/Sub):** Publishes to a tenant-isolated channel (`health_checks:<workspace_id>`).
   - **Stage 3 (Redis Cache):** Caches the latest snapshot for instant dashboard loads.
3. **SSE API Handler:** The Go API subscribes to the Redis Pub/Sub channel and streams JSON frames to connected browsers using `text/event-stream` with a 15s keepalive heartbeat.
4. **React Frontend:** A custom `useSSE` hook consumes the stream, parsing frames and updating reactive state to drive latency sparklines and dynamic HTTP status colors.

---

## ⚡ Quick Start (Local Cluster)

1. Clone the repository and navigate to the project root.
2. Copy `.env.example` to `.env` and fill in your Supabase connection strings:
   ```bash
   DATABASE_URL=your_supabase_pooler_url
   REDIS_URL=redis://localhost:6379
   APP_PORT=8080
   ```
3. Open **4 separate terminal windows** (this prevents background job swallowing on Windows):

   **Terminal 1: Start Redis**
   ```powershell
   docker-compose up -d
   ```

   **Terminal 2: Start the Go API Server**
   ```powershell
   # Load the .env file in PowerShell first
   Get-Content .env | ForEach-Object { $name, $value = $_.Split("=", 2); [System.Environment]::SetEnvironmentVariable($name.Trim(), $value.Trim(), "Process") }
   go run ./cmd/api/main.go
   ```

   **Terminal 3: Start the Go Worker Engine**
   ```powershell
   # Load the .env file in PowerShell first
   Get-Content .env | ForEach-Object { $name, $value = $_.Split("=", 2); [System.Environment]::SetEnvironmentVariable($name.Trim(), $value.Trim(), "Process") }
   go run ./cmd/worker/main.go
   ```

   **Terminal 4: Start the Next.js Dashboard**
   ```powershell
   cd web
   npm install
   npm run dev
   ```

4. Open [http://localhost:3000](http://localhost:3000) in your browser. Add an endpoint (like `https://httpbin.org/get`) and watch the telemetry flow in real-time.

---

## 🧪 Testing & Benchmarks

Run the unit test suite to validate the repository layers and SSRF protections:
```bash
go test ./... -v -count=1
```

Run the performance benchmarks isolating the worker pool throughput:
```bash
go test ./internal/service/... -bench=. -benchmem -benchtime=3s
```

Run the custom load generator (evaluates the full pipeline up to 10,000+ endpoints):
```bash
go run ./cmd/loadtest/main.go
```
