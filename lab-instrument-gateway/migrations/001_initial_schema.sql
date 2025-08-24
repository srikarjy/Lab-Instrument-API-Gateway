-- Initial schema for Lab Instrument Gateway
-- Migration: 001_initial_schema.sql

-- Create extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create enum types
CREATE TYPE device_status AS ENUM (
    'unknown',
    'online', 
    'offline',
    'error',
    'maintenance',
    'connecting'
);

CREATE TYPE quality_code AS ENUM (
    'unknown',
    'good',
    'bad',
    'uncertain',
    'substituted'
);

CREATE TYPE command_status AS ENUM (
    'unknown',
    'pending',
    'executing',
    'completed',
    'failed',
    'timeout',
    'cancelled'
);

-- Devices table
CREATE TABLE devices (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    version VARCHAR(50),
    status device_status DEFAULT 'unknown',
    metadata JSONB DEFAULT '{}',
    capabilities TEXT[] DEFAULT '{}',
    last_seen TIMESTAMP WITH TIME ZONE,
    registered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Device sessions table for tracking active connections
CREATE TABLE device_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id VARCHAR(255) NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    session_id VARCHAR(255) NOT NULL UNIQUE,
    stream_id VARCHAR(255),
    connected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_heartbeat TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true
);

-- Measurements table (partitioned by time for performance)
CREATE TABLE measurements (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id VARCHAR(255) NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    type VARCHAR(100) NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    unit VARCHAR(50),
    quality quality_code DEFAULT 'unknown',
    metadata JSONB DEFAULT '{}',
    batch_id VARCHAR(255),
    sequence_number INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
) PARTITION BY RANGE (timestamp);

-- Create initial partitions for measurements (current month and next month)
CREATE TABLE measurements_current PARTITION OF measurements
    FOR VALUES FROM (DATE_TRUNC('month', NOW())) TO (DATE_TRUNC('month', NOW() + INTERVAL '1 month'));

CREATE TABLE measurements_next PARTITION OF measurements
    FOR VALUES FROM (DATE_TRUNC('month', NOW() + INTERVAL '1 month')) TO (DATE_TRUNC('month', NOW() + INTERVAL '2 months'));

-- Commands table
CREATE TABLE commands (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id VARCHAR(255) NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    command_id VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(100) NOT NULL,
    parameters JSONB DEFAULT '{}',
    status command_status DEFAULT 'pending',
    priority INTEGER DEFAULT 0,
    timeout_seconds INTEGER DEFAULT 30,
    result JSONB DEFAULT '{}',
    error_message TEXT,
    submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    executed_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    execution_time_ms DOUBLE PRECISION,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Alerts table for system notifications
CREATE TABLE alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id VARCHAR(255) REFERENCES devices(id) ON DELETE CASCADE,
    type VARCHAR(100) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    acknowledged BOOLEAN DEFAULT false,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    acknowledged_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for performance optimization

-- Device indexes
CREATE INDEX idx_devices_status ON devices(status);
CREATE INDEX idx_devices_type ON devices(type);
CREATE INDEX idx_devices_last_seen ON devices(last_seen);
CREATE INDEX idx_devices_metadata ON devices USING GIN(metadata);

-- Device sessions indexes
CREATE INDEX idx_device_sessions_device_id ON device_sessions(device_id);
CREATE INDEX idx_device_sessions_active ON device_sessions(is_active);
CREATE INDEX idx_device_sessions_last_heartbeat ON device_sessions(last_heartbeat);

-- Measurements indexes (will be inherited by partitions)
CREATE INDEX idx_measurements_device_timestamp ON measurements(device_id, timestamp DESC);
CREATE INDEX idx_measurements_type_timestamp ON measurements(type, timestamp DESC);
CREATE INDEX idx_measurements_timestamp ON measurements(timestamp DESC);
CREATE INDEX idx_measurements_batch_id ON measurements(batch_id);

-- Commands indexes
CREATE INDEX idx_commands_device_id ON commands(device_id);
CREATE INDEX idx_commands_status ON commands(status);
CREATE INDEX idx_commands_submitted_at ON commands(submitted_at DESC);
CREATE INDEX idx_commands_expires_at ON commands(expires_at);

-- Alerts indexes
CREATE INDEX idx_alerts_device_id ON alerts(device_id);
CREATE INDEX idx_alerts_type ON alerts(type);
CREATE INDEX idx_alerts_severity ON alerts(severity);
CREATE INDEX idx_alerts_acknowledged ON alerts(acknowledged);
CREATE INDEX idx_alerts_created_at ON alerts(created_at DESC);

-- Create functions for automatic timestamp updates
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for automatic timestamp updates
CREATE TRIGGER update_devices_updated_at BEFORE UPDATE ON devices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_commands_updated_at BEFORE UPDATE ON commands
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create function for automatic partition creation
CREATE OR REPLACE FUNCTION create_monthly_partition(table_name text, start_date date)
RETURNS void AS $$
DECLARE
    partition_name text;
    start_month date;
    end_month date;
BEGIN
    start_month := date_trunc('month', start_date);
    end_month := start_month + interval '1 month';
    partition_name := table_name || '_' || to_char(start_month, 'YYYY_MM');
    
    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF %I
                    FOR VALUES FROM (%L) TO (%L)',
                   partition_name, table_name, start_month, end_month);
END;
$$ LANGUAGE plpgsql;

-- Create initial data for testing (optional)
INSERT INTO devices (id, name, type, version, status, capabilities) VALUES
    ('device-001', 'Spectrometer Alpha', 'spectrometer', '1.2.3', 'offline', ARRAY['measurement', 'calibration']),
    ('device-002', 'Microscope Beta', 'microscope', '2.1.0', 'offline', ARRAY['imaging', 'measurement']),
    ('device-003', 'Analyzer Gamma', 'analyzer', '1.0.5', 'offline', ARRAY['analysis', 'reporting']);

-- Grant permissions (adjust as needed for your security model)
-- These would typically be more restrictive in production
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO lab_gateway_user;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO lab_gateway_user;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO lab_gateway_user;