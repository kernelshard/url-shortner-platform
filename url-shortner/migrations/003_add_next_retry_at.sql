ALTER TABLE outbox_events
ADD COLUMN next_retry_at TIMESTAMP DEFAULT  now();
