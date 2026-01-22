-- PostgreSQL initialization for CDPP local development
-- Creates schemas and sample tables for local testing

-- Create test schema
CREATE SCHEMA IF NOT EXISTS test;

-- Grant privileges
GRANT ALL PRIVILEGES ON SCHEMA public TO cdpp;
GRANT ALL PRIVILEGES ON SCHEMA test TO cdpp;

-- Sample table for pipeline outputs
CREATE TABLE IF NOT EXISTS public.processed_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    source_topic VARCHAR(255),
    processed_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Index for efficient querying
CREATE INDEX IF NOT EXISTS idx_processed_events_type ON public.processed_events(event_type);
CREATE INDEX IF NOT EXISTS idx_processed_events_time ON public.processed_events(processed_at);

-- Test table for development
CREATE TABLE IF NOT EXISTS test.sample_data (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    value NUMERIC(10, 2),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert sample data
INSERT INTO test.sample_data (name, value) VALUES
    ('sample-1', 100.50),
    ('sample-2', 200.75),
    ('sample-3', 300.25)
ON CONFLICT DO NOTHING;

-- Audit log table for tracking pipeline runs
CREATE TABLE IF NOT EXISTS public.pipeline_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    package_name VARCHAR(255) NOT NULL,
    package_version VARCHAR(50) NOT NULL,
    run_id VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(50) NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    records_processed BIGINT DEFAULT 0,
    error_message TEXT,
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_pipeline_runs_package ON public.pipeline_runs(package_name);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_status ON public.pipeline_runs(status);
