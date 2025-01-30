-- servers.sql

CREATE SCHEMA IF NOT EXISTS loadbalancer;

CREATE TABLE IF NOT EXISTS loadbalancer.servers (
    id SERIAL PRIMARY KEY,
    url VARCHAR(255) UNIQUE NOT NULL,
    rate_limit INTEGER NOT NULL,
    burst_limit INTEGER NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Add index on is_active for faster queries
CREATE INDEX idx_servers_is_active ON servers(is_active);