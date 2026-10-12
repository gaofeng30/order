CREATE TABLE miniprogram_sessions (
    token_hash BYTEA CHECK (octet_length(token_hash) = 32) NOT NULL,
    user_id NUMERIC(20,0) CHECK (user_id BETWEEN 0 AND 18446744073709551615) NOT NULL,
    issued_at TIMESTAMPTZ(6) NOT NULL,
    expires_at TIMESTAMPTZ(6) NOT NULL,
    PRIMARY KEY (token_hash),
    CONSTRAINT fk_miniprogram_sessions_user FOREIGN KEY (user_id) REFERENCES miniprogram_users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT chk_miniprogram_sessions_expiry CHECK (expires_at > issued_at)
);
CREATE INDEX idx_miniprogram_sessions_user_expiry ON miniprogram_sessions (user_id, expires_at);
