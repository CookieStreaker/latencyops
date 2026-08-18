"use client";

import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, RefreshCw, ExternalLink, Globe, Wifi, WifiOff, Activity } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useSSE, type PingResult } from "@/lib/useSSE";

const WORKSPACE_ID = "ws_prod_default";

// Strict typing for our Domain Model
interface Endpoint {
  ID: string;
  WorkspaceID: string;
  TargetURL: string;
  Name: string;
  Interval: number;
  CreatedAt: string;
}

export default function EndpointsDashboard() {
  const queryClient = useQueryClient();
  const [newUrl, setNewUrl] = useState("");
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  // --- SSE Real-Time Stream ---
  const { latest: liveResults, history: liveHistory, status: sseStatus } = useSSE({
    workspaceId: WORKSPACE_ID,
  });

  // --- API Fetching ---
  const { data: endpoints = [], isLoading, isRefetching, refetch } = useQuery<Endpoint[]>({
    queryKey: ["endpoints"],
    queryFn: async () => {
      const res = await fetch("http://localhost:8080/api/v1/endpoints", {
        headers: { "X-Workspace-ID": WORKSPACE_ID },
      });
      if (!res.ok) throw new Error("Failed to fetch API targets");
      
      const data = await res.json();
      return data || [];
    },
  });

  // --- API Mutations ---
  const createEndpoint = useMutation({
    mutationFn: async (url: string) => {
      setFormError(null);
      const res = await fetch("http://localhost:8080/api/v1/endpoints", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Workspace-ID": WORKSPACE_ID,
        },
        body: JSON.stringify({ TargetURL: url }),
      });

      if (!res.ok) {
        const errorData = await res.json();
        throw new Error(errorData.error || "Failed to register endpoint");
      }
      return res.json();
    },
    onSuccess: () => {
      setNewUrl("");
      setIsDialogOpen(false);
      queryClient.invalidateQueries({ queryKey: ["endpoints"] });
    },
    onError: (err: Error) => {
      setFormError(err.message);
    },
  });

  return (
    <div className="max-w-6xl w-full mx-auto p-6 md:p-10 space-y-8">
      
      {/* Page Header */}
      <header className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-zinc-800/80 pb-6">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-zinc-100">API Probes</h1>
          <p className="text-sm text-zinc-400 mt-1">Real-time health monitoring with sub-millisecond telemetry.</p>
        </div>

        <div className="flex items-center space-x-3">
          {/* SSE Connection Status */}
          <SSEStatusBadge status={sseStatus} />

          <Button
            variant="outline"
            size="sm"
            onClick={() => refetch()}
            disabled={isRefetching}
            className="bg-transparent border-zinc-800 hover:bg-zinc-900 text-zinc-300 transition-all h-9"
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${isRefetching ? "animate-spin text-zinc-500" : ""}`} />
            Refresh
          </Button>

          <Button 
            size="sm" 
            onClick={() => setIsDialogOpen(true)}
            className="bg-zinc-100 text-zinc-950 hover:bg-white font-medium h-9"
          >
            <Plus className="h-4 w-4 mr-1.5" /> Add Probe
          </Button>

          {/* Dialog Modal */}
          <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
            <DialogContent className="bg-zinc-950 border-zinc-800 text-zinc-100 sm:max-w-md">
              <DialogHeader>
                <DialogTitle>Register Target URL</DialogTitle>
              </DialogHeader>
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  if (newUrl.trim()) createEndpoint.mutate(newUrl.trim());
                }}
                className="space-y-4 pt-4"
              >
                <div className="space-y-2">
                  <label className="text-xs font-mono text-zinc-400">Target Endpoint</label>
                  <Input
                    autoFocus
                    placeholder="https://api.startup.com/healthz"
                    value={newUrl}
                    onChange={(e) => setNewUrl(e.target.value)}
                    className="bg-zinc-900 border-zinc-800 focus-visible:ring-1 focus-visible:ring-zinc-700 h-10 text-sm font-mono placeholder:text-zinc-600"
                  />
                </div>

                {formError && (
                  <div className="p-3 rounded-md bg-red-500/10 border border-red-500/20 text-red-400 text-xs font-mono">
                    <span className="font-semibold block mb-0.5">SSRF Validation Failed:</span>
                    {formError}
                  </div>
                )}

                <div className="flex justify-end gap-2 pt-2">
                  <Button type="button" variant="ghost" size="sm" onClick={() => setIsDialogOpen(false)} className="text-zinc-400 hover:text-zinc-100 hover:bg-zinc-900">
                    Cancel
                  </Button>
                  <Button type="submit" size="sm" disabled={createEndpoint.isPending || !newUrl} className="bg-zinc-100 text-zinc-950 hover:bg-white">
                    {createEndpoint.isPending ? "Validating..." : "Register Probe"}
                  </Button>
                </div>
              </form>
            </DialogContent>
          </Dialog>
        </div>
      </header>

      {/* Grid Content */}
      {isLoading ? (
        <div className="flex items-center justify-center h-48 border border-dashed border-zinc-800 rounded-lg bg-zinc-950/50">
          <p className="text-sm font-mono text-zinc-500 animate-pulse">Fetching global edge states...</p>
        </div>
      ) : (!endpoints || endpoints.length === 0) ? (
        <div className="flex flex-col items-center justify-center h-64 border border-dashed border-zinc-800 rounded-lg bg-zinc-950/50 space-y-4">
          <Globe className="h-8 w-8 text-zinc-700" />
          <div className="text-center">
            <h3 className="text-sm font-medium text-zinc-200">No active probes</h3>
            <p className="text-sm text-zinc-500 mt-1 max-w-sm">Register your first API endpoint to begin capturing sub-millisecond telemetry.</p>
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4">
          {endpoints.map((ep) => {
            const live = liveResults.get(ep.ID);
            const sparkData = liveHistory.get(ep.ID) || [];
            return (
              <EndpointCard key={ep.ID} endpoint={ep} liveResult={live} sparkData={sparkData} />
            );
          })}
        </div>
      )}
    </div>
  );
}

// --- Endpoint Card with Live Data ---
function EndpointCard({
  endpoint,
  liveResult,
  sparkData,
}: {
  endpoint: Endpoint;
  liveResult?: PingResult;
  sparkData: PingResult[];
}) {
  const status = liveResult ? (liveResult.is_up ? "UP" : "DOWN") : undefined;
  const latencyMs = liveResult?.latency_ms;
  const statusCode = liveResult?.status_code;
  const lastChecked = liveResult?.timestamp;

  // Compute relative "last checked" time
  const timeAgo = useMemo(() => {
    if (!lastChecked) return null;
    const diff = Math.floor((Date.now() - new Date(lastChecked).getTime()) / 1000);
    if (diff < 5) return "just now";
    if (diff < 60) return `${diff}s ago`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    return `${Math.floor(diff / 3600)}h ago`;
  }, [lastChecked]);

  return (
    <Card className={`bg-zinc-900/40 border-zinc-800/80 hover:border-zinc-700 transition-all duration-300 overflow-hidden group ${
      liveResult && !liveResult.is_up ? "border-red-500/30 hover:border-red-500/50" : ""
    }`}>
      <div className="p-5 flex items-center justify-between">
        
        {/* Left Side: Status & URL */}
        <div className="flex items-center space-x-4">
          <StatusIndicator status={status} />
          <div>
            <div className="flex items-center space-x-2 mb-1">
              <span className="font-mono text-sm font-medium text-zinc-100 tracking-tight">{endpoint.TargetURL}</span>
              <a href={endpoint.TargetURL} target="_blank" rel="noreferrer" className="text-zinc-600 hover:text-zinc-300 transition-colors">
                <ExternalLink className="h-3.5 w-3.5" />
              </a>
            </div>
            <div className="flex items-center space-x-3 text-xs font-mono text-zinc-500">
              <span>ID: {endpoint.ID.split("-")[0]}</span>
              <span className="text-zinc-700">•</span>
              <span>Interval: {endpoint.Interval || 60}s</span>
              {timeAgo && (
                <>
                  <span className="text-zinc-700">•</span>
                  <span className="text-zinc-400">{timeAgo}</span>
                </>
              )}
            </div>
          </div>
        </div>

        {/* Right Side: Live Metrics */}
        <div className="flex items-center space-x-6">
          {/* Sparkline */}
          {sparkData.length > 1 && (
            <div className="hidden md:block">
              <LatencySparkline data={sparkData} />
            </div>
          )}

          {/* Latency */}
          <div className="hidden sm:flex flex-col items-end">
            <span className="text-[10px] font-mono text-zinc-500 uppercase">Latency</span>
            <span className={`text-sm font-mono font-medium transition-colors duration-300 ${
              latencyMs !== undefined
                ? latencyMs > 500
                  ? "text-red-400"
                  : latencyMs > 200
                  ? "text-amber-400"
                  : "text-emerald-400"
                : "text-zinc-500"
            }`}>
              {latencyMs !== undefined ? `${latencyMs}ms` : "-- ms"}
            </span>
          </div>

          {/* Status Code Badge */}
          <div className="flex items-center">
            <span className={`inline-flex items-center justify-center px-2 py-1 rounded-[4px] text-[10px] font-mono font-semibold border transition-all duration-300 ${
              statusCode !== undefined
                ? statusCode >= 200 && statusCode < 300
                  ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/30"
                  : statusCode >= 400
                  ? "bg-red-500/10 text-red-400 border-red-500/30"
                  : "bg-amber-500/10 text-amber-400 border-amber-500/30"
                : "bg-zinc-800 text-zinc-300 border-zinc-700"
            }`}>
              {statusCode !== undefined ? `HTTP ${statusCode}` : "HTTP ---"}
            </span>
          </div>
        </div>

      </div>
    </Card>
  );
}

// --- Sparkline Component ---
function LatencySparkline({ data }: { data: PingResult[] }) {
  const width = 120;
  const height = 32;
  const padding = 2;

  if (data.length < 2) return null;

  const values = data.map((d) => d.latency_ms);
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;

  const points = values
    .map((v, i) => {
      const x = padding + (i / (values.length - 1)) * (width - 2 * padding);
      const y = height - padding - ((v - min) / range) * (height - 2 * padding);
      return `${x},${y}`;
    })
    .join(" ");

  const latestIsUp = data[data.length - 1]?.is_up;
  const strokeColor = latestIsUp ? "rgb(52, 211, 153)" : "rgb(248, 113, 113)";

  return (
    <svg width={width} height={height} className="opacity-70 group-hover:opacity-100 transition-opacity">
      <polyline
        fill="none"
        stroke={strokeColor}
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        points={points}
      />
      {/* Glow dot on latest point */}
      <circle
        cx={padding + ((values.length - 1) / (values.length - 1)) * (width - 2 * padding)}
        cy={height - padding - ((values[values.length - 1] - min) / range) * (height - 2 * padding)}
        r="2.5"
        fill={strokeColor}
        className="animate-pulse"
      />
    </svg>
  );
}

// --- SSE Connection Status Badge ---
function SSEStatusBadge({ status }: { status: string }) {
  const config = {
    connected: { icon: Wifi, label: "Live", color: "text-emerald-400 bg-emerald-500/10 border-emerald-500/20" },
    connecting: { icon: Activity, label: "Connecting", color: "text-amber-400 bg-amber-500/10 border-amber-500/20" },
    disconnected: { icon: WifiOff, label: "Reconnecting", color: "text-amber-400 bg-amber-500/10 border-amber-500/20" },
    error: { icon: WifiOff, label: "Offline", color: "text-red-400 bg-red-500/10 border-red-500/20" },
  }[status] || { icon: WifiOff, label: "Unknown", color: "text-zinc-400 bg-zinc-800 border-zinc-700" };

  const Icon = config.icon;

  return (
    <div className={`inline-flex items-center space-x-1.5 px-2.5 py-1 rounded-full text-[10px] font-mono font-semibold border ${config.color}`}>
      <Icon className={`h-3 w-3 ${status === "connecting" ? "animate-pulse" : ""}`} />
      <span>{config.label}</span>
    </div>
  );
}

// --- Status Indicator Dot ---
function StatusIndicator({ status }: { status?: string }) {
  if (status === "DOWN") {
    return (
      <div className="relative flex h-3 w-3">
        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-20"></span>
        <span className="relative inline-flex rounded-full h-3 w-3 bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.6)]"></span>
      </div>
    );
  }
  if (status === "UP") {
    return (
      <div className="relative flex h-3 w-3">
        <span className="relative inline-flex rounded-full h-3 w-3 bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]"></span>
      </div>
    );
  }
  // Default: no data yet
  return (
    <div className="relative flex h-3 w-3">
      <span className="relative inline-flex rounded-full h-3 w-3 bg-zinc-600 animate-pulse"></span>
    </div>
  );
}