# LatencyOps Repository Instructions

## App Summary
LatencyOps is a real-time API health, uptime, and rate-limit alerting monitor built for developer teams.

## Stack & Versions
- Backend: Go (Golang) 1.22+
- Cache/State: Redis 7+
- Persistence: PostgreSQL 16+ (Supabase)
- Frontend: React / Next.js with Tailwind CSS & Shadcn UI

## Go Architecture Conventions
- Standard Clean Architecture layout (`/cmd`, `/internal/handler`, `/internal/service`, `/internal/repository`).
- Explicit error wrapping: `fmt.Errorf("failed to execute ping probe: %w", err)`. Never ignore errors with `_`.
- Use parameterized SQL queries (`pgx`) to eliminate SQL injection risks.

## OWASP Top 10 Security Rules
- Tenant Isolation: EVERY DB query fetching/modifying resources MUST include `WHERE id = $1 AND workspace_id = $2`.
- SSRF Protection: When pinging user-defined URLs, block requests targeting internal/private IP ranges (127.0.0.1, 10.0.0.0/8, 169.254.169.254).
- Auth: Store JWT tokens strictly in HttpOnly, Secure, SameSite=Strict cookies.
- Request Limits: Wrap handlers in `http.MaxBytesReader(w, r.Body, 1024*1024)` (1MB max).