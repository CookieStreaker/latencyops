package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	// Replace "latencyops" with your actual go.mod module name if different
	"latencyops/internal/domain"
	"latencyops/internal/repository"
)

const (
	DefaultPort = "8080"
	MaxBodySize = 1024 * 1024 // 1MB payload limit for OWASP API4
)

func main() {
	// 1. Setup Context with Graceful Shutdown (SIGINT/SIGTERM)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("🚀 Starting LatencyOps API Server...")

	// 2. Load Environment Variables
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("FATAL: DATABASE_URL environment variable is required")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("FATAL: REDIS_URL environment variable is required")
	}

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = DefaultPort
	}

	// 3. Initialize Postgres Pool (Supabase)
	pgPool, err := repository.NewPostgresPool(ctx, dbURL)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize PostgreSQL pool: %v", err)
	}
	defer pgPool.Close()
	log.Println("✅ Connected to PostgreSQL (Supabase)")

	// 4. Initialize Redis Client
	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		redisOpts = &redis.Options{Addr: redisURL}
	}
	redisClient := redis.NewClient(redisOpts)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("FATAL: Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("✅ Connected to Redis")

	// 5. Instantiate Repositories
	endpointRepo := repository.NewPostgresEndpointRepo(pgPool)
	_ = repository.NewRedisStateRepo(redisClient) // Ready for future dashboard state fetching

	// 6. Setup Go 1.22 Router
	mux := http.NewServeMux()

	// 7. Register Routes
	mux.HandleFunc("GET /healthz", healthCheckHandler)
	
	// Inject dependencies into our route handlers using closures
	mux.HandleFunc("GET /api/v1/endpoints", getEndpointsHandler(endpointRepo))
	mux.HandleFunc("POST /api/v1/endpoints", createEndpointHandler(endpointRepo))

	// 8. Apply Global Middleware (Security & Headers)
	handler := applySecurityMiddleware(mux)

	// 9. Configure HTTP Server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 10. Start Server Concurrently
	go func() {
		log.Printf("🌐 API Server listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("FATAL: HTTP server error: %v", err)
		}
	}()

	// 11. Graceful Shutdown Listener
	<-ctx.Done()
	log.Println("⚠️  Shutdown signal received. Initiating graceful shutdown...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("ERROR: HTTP server shutdown failed: %v", fmt.Errorf("shutdown error: %w", err))
	}
	log.Println("🛑 LatencyOps API Server successfully shut down.")
}

// --- Route Handlers ---

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok", "service": "latencyops-api"}`))
}

func getEndpointsHandler(repo repository.EndpointRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// IN A REAL APP: Extract WorkspaceID from JWT context here.
		// For MVP, we simulate a secure tenant extraction.
		workspaceID := r.Header.Get("X-Workspace-ID")
		if workspaceID == "" {
			http.Error(w, `{"error": "missing workspace_id"}`, http.StatusUnauthorized)
			return
		}

		endpoints, err := repo.GetActiveEndpoints(r.Context())
		if err != nil {
			log.Printf("ERROR: failed to fetch endpoints: %v", fmt.Errorf("postgres read error: %w", err))
			http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(endpoints)
	}
}

func createEndpointHandler(repo repository.EndpointRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// IN A REAL APP: Extract WorkspaceID from JWT context here.
		workspaceID := r.Header.Get("X-Workspace-ID")
		if workspaceID == "" {
			http.Error(w, `{"error": "missing workspace_id"}`, http.StatusUnauthorized)
			return
		}

		var input domain.Endpoint
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, `{"error": "invalid json payload"}`, http.StatusBadRequest)
			return
		}

		// Apply OWASP API7 Mitigation: SSRF Validation using the correct TargetURL field
		if err := domain.ValidateURL(input.TargetURL); err != nil {
			log.Printf("SECURITY: SSRF attempt blocked for URL %s: %v", input.TargetURL, err)
			http.Error(w, fmt.Sprintf(`{"error": "invalid url: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		// Assign the verified tenant ID to prevent BOLA
		input.WorkspaceID = workspaceID

		// TODO: Save to Postgres using repo.Save(ctx, input) when implemented
		
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created", "id": input.ID})
	}
}

// --- Middleware ---

func applySecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. OWASP API4: Payload Defense (Prevent Resource Exhaustion)
		r.Body = http.MaxBytesReader(w, r.Body, MaxBodySize)

		// 2. Standard Secure JSON Headers
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// 3. CORS Configuration (Simplified for MVP)
		w.Header().Set("Access-Control-Allow-Origin", "*") 
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Workspace-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}