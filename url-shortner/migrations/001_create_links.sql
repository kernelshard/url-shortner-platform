CREATE TABLE IF NOT EXISTS links (
    id UUID PRIMARY KEY,
    original_url TEXT UNIQUE NOT NULL,
    short_code TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP
);