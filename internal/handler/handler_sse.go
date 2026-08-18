package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// LiveMetricsHandler creates an SSE endpoint that subscribes to the Redis Pub/Sub
// channel for the requesting workspace and streams PingResult events to the client.
//
// Security: BOLA enforcement via workspace_id extraction.
// Protocol: Server-Sent Events (text/event-stream) with 15s keepalive heartbeat.
func LiveMetricsHandler(redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. BOLA Enforcement: Extract workspace_id from query param or header
		workspaceID := r.URL.Query().Get("workspace_id")
		if workspaceID == "" {
			workspaceID = r.Header.Get("X-Workspace-ID")
		}
		if workspaceID == "" {
			http.Error(w, `{"error": "missing workspace_id"}`, http.StatusUnauthorized)
			return
		}

		// 2. Verify the response writer supports streaming (flushing)
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, `{"error": "streaming not supported"}`, http.StatusInternalServerError)
			return
		}

		// 3. Set SSE response headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Workspace-ID")
		w.Header().Set("X-Accel-Buffering", "no") // Disable Nginx buffering if behind reverse proxy

		// 4. Subscribe to the per-workspace Redis Pub/Sub channel
		channel := fmt.Sprintf("health_checks:%s", workspaceID)
		sub := redisClient.Subscribe(r.Context(), channel)
		defer sub.Close()

		pubsubCh := sub.Channel()
		log.Printf("SSE: Client connected for workspace %s on channel %s", workspaceID, channel)

		// 5. Send initial connection event
		fmt.Fprintf(w, "event: connected\ndata: {\"workspace_id\":\"%s\",\"status\":\"streaming\"}\n\n", workspaceID)
		flusher.Flush()

		// 6. Heartbeat ticker to keep the connection alive through proxies/LBs
		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()

		// 7. Event loop: stream Redis messages as SSE events
		for {
			select {
			case <-r.Context().Done():
				// Client disconnected
				log.Printf("SSE: Client disconnected for workspace %s", workspaceID)
				return

			case msg, ok := <-pubsubCh:
				if !ok {
					// Redis channel closed unexpectedly
					log.Printf("ERROR: Redis Pub/Sub channel closed for workspace %s", workspaceID)
					return
				}

				// Validate JSON before forwarding to prevent malformed SSE frames
				if !json.Valid([]byte(msg.Payload)) {
					log.Printf("ERROR: invalid JSON received on channel %s: %s", channel, msg.Payload)
					continue
				}

				// Write SSE event frame
				fmt.Fprintf(w, "event: ping_result\ndata: %s\n\n", msg.Payload)
				flusher.Flush()

			case <-heartbeat.C:
				// SSE keepalive comment (ignored by EventSource clients)
				fmt.Fprintf(w, ": keepalive %d\n\n", time.Now().Unix())
				flusher.Flush()
			}
		}
	}
}
