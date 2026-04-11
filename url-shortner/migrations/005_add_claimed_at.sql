ALTER TABLE outbox_events
ADD COLUMN claimed_at TIMESTAMP NULL;
