latencyops/
├── .github/
│   └── copilot-instructions.md
├── cmd/
│   ├── api/          # Web API server entrypoint
│   └── worker/       # High-concurrency ping engine entrypoint
├── internal/
│   ├── domain/       # Core structs (Endpoint, PingResult, AlertRule)
│   ├── handler/      # HTTP & Webhook handlers
│   ├── service/      # Business logic (probe scheduler, threshold checker)
│   └── repository/   # PostgreSQL & Redis data access layers
├── docker-compose.yml
├── go.mod
└── go.sum
