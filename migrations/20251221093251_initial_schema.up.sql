CREATE TABLE IF NOT EXISTS t_user (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS t_user_data (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES t_user(id) ON DELETE CASCADE,
    type INTEGER NOT NULL,
    name TEXT NOT NULL,
    encrypted_data BYTEA NOT NULL,
    metadata TEXT,
    version INTEGER DEFAULT 1 NOT NULL,
    updated_at BIGINT NOT NULL,
    deleted BOOLEAN DEFAULT FALSE,
    CONSTRAINT valid_type CHECK (type BETWEEN 0 AND 3),
    CONSTRAINT positive_version CHECK (version > 0)
);