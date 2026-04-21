ALTER TABLE outbox_events
ADD COLUMN status TEXT NOT NULL DEFAULT 'pending';
