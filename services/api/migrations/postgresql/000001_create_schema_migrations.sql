CREATE TABLE IF NOT EXISTS schema_migrations (
  version BIGINT PRIMARY KEY CHECK (version > 0),
  name VARCHAR(255) NOT NULL,
  checksum BYTEA NOT NULL CHECK (octet_length(checksum) = 32),
  dirty BOOLEAN NOT NULL,
  applied_at TIMESTAMPTZ(6) NULL
);
