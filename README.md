# 🚀 LatencyOps — Real-Time API Health & Rate-Limit Monitor

LatencyOps is a high-performance, real-time API health, uptime, and rate-limit alerting monitor built specifically for engineering teams and CTOs.

## 🛠️ Backend Tech Stack

- **Language:** Go (Golang 1.22+)
- **Architecture:** Clean Architecture Layout (`/cmd`, `/internal`)
- **Database:** PostgreSQL (Supabase) via `pgxpool`
- **Cache/State:** Redis 7+ via `go-redis/v9`
- **Security:** OWASP Top 10 API Security (SSRF URL validation, BOLA tenant isolation)

## ⚡ Quick Start (Local Cluster)

1. Clone the repository and navigate to the project root.
2. Copy `.env.example` to `.env` and fill in your Supabase connection strings:
   ```bash
   DATABASE_URL=your_supabase_pooler_url
   REDIS_URL=redis://localhost:6379
   APP_PORT=8080
