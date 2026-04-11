ALTER TABLE outbox_events
ADD COLUMN retry_count INT DEFAULT 0;