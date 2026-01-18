-- 1. Define ENUM Types
-- Using ENUMs saves space and prevents invalid values
CREATE TYPE service_status_enum AS ENUM ('UP', 'DOWN', 'DEGRADED');
CREATE TYPE event_type_enum AS ENUM ('WENT_DOWN', 'CAME_UP');

-- 2. Services Table (Inventory)
CREATE TABLE IF NOT EXISTS services (
    id SERIAL PRIMARY KEY,
    service_name VARCHAR(255) NOT NULL,
    service_url VARCHAR(255),
    region VARCHAR(50),
    
    -- Use the ENUM here
    status service_status_enum DEFAULT 'UP',
    
    last_heartbeat TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 3. Incident Log Table (History)
CREATE TABLE IF NOT EXISTS incident_log (
    id SERIAL PRIMARY KEY,
    service_id INT REFERENCES services(id),
    
    -- Use the ENUM here
    event_type event_type_enum NOT NULL,
    
    event_time TIMESTAMP DEFAULT NOW(),
    details TEXT
);

-- 4. Indexes for Performance
-- Index on service_id (standard FK index)
CREATE INDEX idx_incident_service ON incident_log(service_id);

-- Index on the ENUM column
-- This makes queries like "WHERE event_type = 'WENT_DOWN'" extremely fast
CREATE INDEX idx_incident_type ON incident_log(event_type);