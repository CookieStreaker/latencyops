"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, RefreshCw, ExternalLink, Globe } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

// Strict typing for our Domain Model
interface Endpoint {
  id: string;
  workspace_id: string;
  target_url: string;
  check_interval: number;
  status: "UP" | "DEGRADED" | "DOWN";
}

export default function EndpointsDashboard() {
  const queryClient = useQueryClient();
  const [newUrl, setNewUrl] = useState("");
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  // --- API Fetching ---
  const { data: endpoints = [], isLoading, isRefetching, refetch } = useQuery<Endpoint[]>({
    queryKey: ["endpoints"],
    queryFn: async () => {
      const res = await fetch("http://localhost:8080/api/v1/endpoints", {
        headers: { "X-Workspace-ID": "ws_prod_default" },
      });
      if (!res.ok) throw new Error("Failed to fetch API targets");
      
      const data = await res.json();
      // FIX: Ensure Go's `null` response becomes a safe empty array `[]`
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
          "X-Workspace-ID": "ws_prod_default",
        },
        body: JSON.stringify({ target_url: url }),
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
          <p className="text-sm text-zinc-400 mt-1">Manage target URLs and monitor TTFB latency.</p>
        </div>

        <div className="flex items-center space-x-3">
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

          {/* Standalone Button Triggering Dialog Manually */}
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
        // FIX: Bulletproof null/empty check applies here
        <div className="flex flex-col items-center justify-center h-64 border border-dashed border-zinc-800 rounded-lg bg-zinc-950/50 space-y-4">
          <Globe className="h-8 w-8 text-zinc-700" />
          <div className="text-center">
            <h3 className="text-sm font-medium text-zinc-200">No active probes</h3>
            <p className="text-sm text-zinc-500 mt-1 max-w-sm">Register your first API endpoint to begin capturing sub-millisecond telemetry.</p>
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4">
          {endpoints.map((ep) => (
            <Card key={ep.id} className="bg-zinc-900/40 border-zinc-800/80 hover:border-zinc-700 transition-colors duration-200 overflow-hidden group">
              <div className="p-5 flex items-center justify-between">
                
                {/* Left Side: Status & URL */}
                <div className="flex items-center space-x-4">
                  <StatusIndicator status={ep.status} />
                  <div>
                    <div className="flex items-center space-x-2 mb-1">
                      <span className="font-mono text-sm font-medium text-zinc-100 tracking-tight">{ep.target_url}</span>
                      <a href={ep.target_url} target="_blank" rel="noreferrer" className="text-zinc-600 hover:text-zinc-300 transition-colors">
                        <ExternalLink className="h-3.5 w-3.5" />
                      </a>
                    </div>
                    <div className="flex items-center space-x-3 text-xs font-mono text-zinc-500">
                      <span>ID: {ep.id.split("-")[0]}</span>
                      <span className="text-zinc-700">•</span>
                      <span>Interval: {ep.check_interval || 60}s</span>
                    </div>
                  </div>
                </div>

                {/* Right Side: Metrics/Actions */}
                <div className="flex items-center space-x-6">
                  <div className="hidden sm:flex flex-col items-end">
                    <span className="text-[10px] font-mono text-zinc-500 uppercase">P99 Latency</span>
                    <span className="text-sm font-mono font-medium text-zinc-300">-- ms</span>
                  </div>
                  <div className="flex items-center">
                    <span className="inline-flex items-center justify-center px-2 py-1 rounded-[4px] text-[10px] font-mono font-semibold bg-zinc-800 text-zinc-300 border border-zinc-700">
                      HTTP 200
                    </span>
                  </div>
                </div>

              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

// Strictly minimal Vercel-style glowing status dots
function StatusIndicator({ status }: { status?: string }) {
  if (status === "DOWN") {
    return (
      <div className="relative flex h-3 w-3">
        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-20"></span>
        <span className="relative inline-flex rounded-full h-3 w-3 bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.6)]"></span>
      </div>
    );
  }
  if (status === "DEGRADED") {
    return (
      <div className="relative flex h-3 w-3">
        <span className="relative inline-flex rounded-full h-3 w-3 bg-amber-500 shadow-[0_0_8px_rgba(245,158,11,0.6)]"></span>
      </div>
    );
  }
  // Default UP
  return (
    <div className="relative flex h-3 w-3">
      <span className="relative inline-flex rounded-full h-3 w-3 bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]"></span>
    </div>
  );
}