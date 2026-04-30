CREATE TABLE IF NOT EXISTS email_deliveries (
    event_id UUID PRIMARY KEY,
    email TEXT NOT NULL CHECK (email <> ''),
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'sent', 'failed')),
    retry_count INT NOT NULL DEFAULT 0 CHECK (retry_count >= 0),
    last_error TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    sent_at TIMESTAMP
);


-- we will frequently query email deliveries by status, so let's add an index on that column
CREATE INDEX IF NOT EXISTS idx_email_deliveries_status
ON email_deliveries (status);
