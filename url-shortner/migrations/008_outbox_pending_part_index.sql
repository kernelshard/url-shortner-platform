-- Index on the next_retry_at column to efficiently query pending events
CREATE INDEX idx_outbox_pending
ON outbox_events (next_retry_at)
WHERE status = 'pending';
