CREATE TABLE discount_settings (
  id TINYINT UNSIGNED NOT NULL,
  rate_percent TINYINT UNSIGNED NOT NULL,
  discount_version BIGINT UNSIGNED NOT NULL,
  whitelist_version BIGINT UNSIGNED NOT NULL,
  updated_at TIMESTAMP(6) NOT NULL,
  PRIMARY KEY (id),
  CONSTRAINT chk_discount_settings_singleton CHECK (id = 1),
  CONSTRAINT chk_discount_settings_rate CHECK (rate_percent >= 1 AND rate_percent <= 100),
  CONSTRAINT chk_discount_settings_versions CHECK (discount_version > 0 AND whitelist_version > 0)
) ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
