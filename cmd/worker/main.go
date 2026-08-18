package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"latencyops/internal/domain"
	"latencyops/internal/repository"
	"latencyops/internal/service"
)

const (
	TickerInterval   = 60 * time.Second
	ProbeTimeout     = 10 * time.Second
	ConcurrencyLimit = 50
)

func main() {
	// 1. Setup Context with Graceful Shutdown (SIGINT/SIGTERM)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("🚀 Starting LatencyOps Worker Engine...")

	// 2. Load Environment Variables
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("FATAL: DATABASE_URL environment variable is required")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("FATAL: REDIS_URL environment variable is required")
	}

	// 3. Initialize Postgres Pool (Supabase)
	pgPool, err := repository.NewPostgresPool(ctx, dbURL)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize PostgreSQL pool: %v", err)
	}
	defer pgPool.Close()
	log.Println("✅ Connected to PostgreSQL (Supabase)")

	// 4. Initialize Redis Client
	// ParseURL handles fully qualified URIs, fallback handles simple "localhost:6379"
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
	pingResultRepo := repository.NewPostgresPingResultRepo(pgPool)
	stateRepo := repository.NewRedisStateRepo(redisClient)

	// 6. Instantiate Services
	probeSvc := service.NewProbeService(ProbeTimeout)
	
	// Create a buffered channel for ping results to prevent blocking the worker pool
	resultsChan := make(chan domain.PingResult, 1000)
	workerPool := service.NewWorkerPool(ConcurrencyLimit, probeSvc, resultsChan)

	// 7. Start the WorkerPool
	// This spins up the goroutines that will sit idle until dispatched
	workerPool.Start(ctx)
	log.Printf("🛠️  Worker pool started (Concurrency: %d)", ConcurrencyLimit)

	// 8. Start Results Listener — The Full Data Pipeline
	// Each result flows through three independent stages:
	//   1. Postgres persistence (historical metrics)
	//   2. Redis Pub/Sub publish (SSE real-time stream)
	//   3. Redis state cache (dashboard snapshot)
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("🛑 Stopping results listener...")
				return
			case result := <-resultsChan:
				// Use a fresh background context here so DB saves aren't abruptly 
				// canceled mid-flight if the parent context closes.
				saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

				// Stage 1: Persist to PostgreSQL (historical record)
				if err := pingResultRepo.SavePingResult(saveCtx, result); err != nil {
					log.Printf("ERROR: failed to persist ping result to PostgreSQL for endpoint %s: %v",
						result.EndpointID, fmt.Errorf("postgres write error: %w", err))
				}

				// Stage 2: Publish to Redis Pub/Sub (SSE consumers)
				if err := stateRepo.PublishPingResult(saveCtx, result); err != nil {
					log.Printf("ERROR: failed to publish ping result to Redis Pub/Sub for endpoint %s: %v",
						result.EndpointID, fmt.Errorf("redis pubsub error: %w", err))
				}

				// Stage 3: Cache current state in Redis (dashboard snapshots)
				if err := stateRepo.SavePingState(saveCtx, result); err != nil {
					log.Printf("ERROR: failed to save ping state to Redis for endpoint %s: %v",
						result.EndpointID, fmt.Errorf("redis state error: %w", err))
				}

				cancel()
			}
		}
	}()

	// 9. Continuous Fetch & Dispatch Loop
	ticker := time.NewTicker(TickerInterval)
	defer ticker.Stop()

	// Execute the first cycle immediately so we don't wait 60 seconds on boot
	runDispatchCycle(ctx, endpointRepo, workerPool)

	for {
		select {
		case <-ctx.Done():
			log.Println("⚠️  Shutdown signal received. Initiating graceful shutdown...")
			
			// Allow a brief grace period for in-flight requests and Redis writes to finish
			time.Sleep(2 * time.Second)
			log.Println("🛑 LatencyOps Worker Engine successfully shut down.")
			return
		case <-ticker.C:
			runDispatchCycle(ctx, endpointRepo, workerPool)
		}
	}
}

// runDispatchCycle extracts active endpoints from Postgres and pushes them into the worker pipeline
func runDispatchCycle(ctx context.Context, repo repository.EndpointRepository, pool *service.WorkerPool) {
	log.Println("🔍 Fetching active endpoints from PostgreSQL...")
	
	fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	endpoints, err := repo.GetActiveEndpoints(fetchCtx)
	if err != nil {
		log.Printf("ERROR: failed to fetch active endpoints: %v", fmt.Errorf("postgres fetch error: %w", err))
		return
	}

	log.Printf("⚡ Dispatching %d endpoints to worker pool", len(endpoints))
	for _, ep := range endpoints {
		pool.Dispatch(ep)
	}
}