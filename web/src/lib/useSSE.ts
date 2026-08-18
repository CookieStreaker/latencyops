"use client";

import { useState, useEffect, useRef, useCallback } from "react";

// Matches the Go domain.PingResult JSON serialization
export interface PingResult {
  workspace_id: string;
  endpoint_id: string;
  status_code: number;
  latency_ms: number;
  is_up: boolean;
  timestamp: string;
}

interface UseSSEOptions {
  workspaceId: string;
  apiBaseUrl?: string;
  maxHistoryPerEndpoint?: number;
}

interface SSEState {
  /** Latest result per endpoint */
  latest: Map<string, PingResult>;
  /** Rolling history per endpoint (for sparklines) */
  history: Map<string, PingResult[]>;
  /** Connection status */
  status: "connecting" | "connected" | "disconnected" | "error";
}

/**
 * useSSE connects to the Go API's SSE stream and maintains a reactive state
 * of live ping results. Includes automatic reconnection with exponential backoff.
 */
export function useSSE({ workspaceId, apiBaseUrl = "http://localhost:8080", maxHistoryPerEndpoint = 30 }: UseSSEOptions) {
  const [state, setState] = useState<SSEState>({
    latest: new Map(),
    history: new Map(),
    status: "connecting",
  });

  const retryCountRef = useRef(0);
  const maxRetries = 10;
  const abortRef = useRef<AbortController | null>(null);

  const connect = useCallback(() => {
    // Abort any existing connection
    if (abortRef.current) {
      abortRef.current.abort();
    }

    const controller = new AbortController();
    abortRef.current = controller;

    setState((prev) => ({ ...prev, status: "connecting" }));

    const url = `${apiBaseUrl}/api/v1/live-metrics?workspace_id=${encodeURIComponent(workspaceId)}`;

    fetch(url, {
      signal: controller.signal,
      headers: { Accept: "text/event-stream" },
    })
      .then((response) => {
        if (!response.ok) {
          throw new Error(`SSE connection failed: ${response.status}`);
        }

        const reader = response.body?.getReader();
        if (!reader) throw new Error("ReadableStream not supported");

        const decoder = new TextDecoder();
        retryCountRef.current = 0; // Reset on successful connection

        let buffer = "";

        const processStream = async () => {
          try {
            while (true) {
              const { done, value } = await reader.read();
              if (done) break;

              buffer += decoder.decode(value, { stream: true });

              // Parse SSE frames from the buffer
              const lines = buffer.split("\n");
              buffer = lines.pop() || ""; // Keep incomplete line in buffer

              let currentEvent = "";
              let currentData = "";

              for (const line of lines) {
                if (line.startsWith("event: ")) {
                  currentEvent = line.slice(7).trim();
                } else if (line.startsWith("data: ")) {
                  currentData = line.slice(6).trim();
                } else if (line === "" && currentData) {
                  // Empty line = end of event frame
                  if (currentEvent === "connected") {
                    setState((prev) => ({ ...prev, status: "connected" }));
                  } else if (currentEvent === "ping_result") {
                    try {
                      const result: PingResult = JSON.parse(currentData);
                      setState((prev) => {
                        const newLatest = new Map(prev.latest);
                        newLatest.set(result.endpoint_id, result);

                        const newHistory = new Map(prev.history);
                        const endpointHistory = [...(newHistory.get(result.endpoint_id) || []), result];
                        // Keep only the last N entries for sparkline rendering
                        newHistory.set(
                          result.endpoint_id,
                          endpointHistory.slice(-maxHistoryPerEndpoint)
                        );

                        return { latest: newLatest, history: newHistory, status: "connected" };
                      });
                    } catch {
                      console.error("[useSSE] Failed to parse ping_result:", currentData);
                    }
                  }
                  currentEvent = "";
                  currentData = "";
                }
              }
            }
          } catch (err: unknown) {
            if (err instanceof Error && err.name === "AbortError") return;
            throw err;
          }

          // Stream ended naturally — reconnect
          handleReconnect();
        };

        processStream().catch(() => handleReconnect());
      })
      .catch((err: Error) => {
        if (err.name === "AbortError") return;
        console.error("[useSSE] Connection error:", err.message);
        handleReconnect();
      });
  }, [workspaceId, apiBaseUrl, maxHistoryPerEndpoint]);

  const handleReconnect = useCallback(() => {
    if (retryCountRef.current >= maxRetries) {
      setState((prev) => ({ ...prev, status: "error" }));
      return;
    }

    setState((prev) => ({ ...prev, status: "disconnected" }));

    // Exponential backoff: 1s, 2s, 4s, 8s... capped at 30s
    const delay = Math.min(1000 * Math.pow(2, retryCountRef.current), 30000);
    retryCountRef.current += 1;

    console.log(`[useSSE] Reconnecting in ${delay}ms (attempt ${retryCountRef.current}/${maxRetries})`);
    setTimeout(connect, delay);
  }, [connect]);

  useEffect(() => {
    connect();
    return () => {
      abortRef.current?.abort();
    };
  }, [connect]);

  return state;
}
