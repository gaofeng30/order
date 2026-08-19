CREATE TABLE miniprogram_users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    openid VARBINARY(128) NOT NULL,
    created_at TIMESTAMP(6) NOT NULL,
    last_login_at TIMESTAMP(6) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_miniprogram_users_openid (openid)
) ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
