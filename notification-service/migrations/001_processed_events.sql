CREATE TABLE IF NOT EXISTS processed_events (
    event_id UUID PRIMARY KEY,
    created_at TIMESTAMP DEFAULT NOW()
);
