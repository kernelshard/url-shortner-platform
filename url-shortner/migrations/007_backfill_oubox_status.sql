UPDATE outbox_events
SET status = 'processed'
WHERE processed = true;

UPDATE outbox_events
SET status = 'pending'
WHERE processed = false;
