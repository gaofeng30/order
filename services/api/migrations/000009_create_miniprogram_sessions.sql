CREATE TABLE miniprogram_sessions (
    token_hash BINARY(32) NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    issued_at TIMESTAMP(6) NOT NULL,
    expires_at TIMESTAMP(6) NOT NULL,
    PRIMARY KEY (token_hash),
    KEY idx_miniprogram_sessions_user_expiry (user_id, expires_at),
    CONSTRAINT fk_miniprogram_sessions_user FOREIGN KEY (user_id) REFERENCES miniprogram_users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT chk_miniprogram_sessions_expiry CHECK (expires_at > issued_at)
) ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
