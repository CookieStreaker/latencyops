-- Enable UUID extension for object enumeration protection (OWASP API1 / API7 mitigation)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Workspaces Table (Tenant Isolation)
CREATE TABLE IF NOT EXISTS workspaces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Endpoints Table
CREATE TABLE IF NOT EXISTS endpoints (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    target_url TEXT NOT NULL,
    interval_seconds INT NOT NULL DEFAULT 60,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Ping Results Table (Historical Metrics)
CREATE TABLE IF NOT EXISTS ping_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    status_code INT NOT NULL,
    latency_ms INT NOT NULL,
    is_up BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Performance Indexes
CREATE INDEX IF NOT EXISTS idx_endpoints_workspace_id ON endpoints(workspace_id);
CREATE INDEX IF NOT EXISTS idx_ping_results_endpoint_id ON ping_results(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_ping_results_created_at ON ping_results(created_at DESC);