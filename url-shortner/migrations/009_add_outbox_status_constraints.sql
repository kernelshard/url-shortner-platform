-- Add constraints to the status column
ALTER TABLE outbox_events
ADD CONSTRAINT status_valid
CHECK (status IN ('pending', 'processed', 'dead'));
