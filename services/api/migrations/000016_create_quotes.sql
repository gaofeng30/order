CREATE TABLE quotes (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  contact_name_snapshot VARCHAR(64) NOT NULL,
  contact_phone_snapshot VARBINARY(16) NOT NULL,
  idempotency_key_hash BINARY(32) NOT NULL,
  request_digest BINARY(32) NOT NULL,
  identity_kind ENUM('STAFF','VISITOR') NOT NULL,
  identity_source_version BIGINT UNSIGNED NOT NULL,
  discount_rate_percent TINYINT UNSIGNED NOT NULL,
  discount_version BIGINT UNSIGNED NOT NULL,
  store_name_snapshot TEXT NOT NULL,
  store_address_snapshot TEXT NOT NULL,
  pickup_point_snapshot TEXT NOT NULL,
  pickup_date DATE NOT NULL,
  pickup_time TIME NOT NULL,
  meal_period ENUM('lunch','dinner') NOT NULL,
  order_note TEXT NOT NULL,
  item_count SMALLINT UNSIGNED NOT NULL,
  original_subtotal_cents BIGINT UNSIGNED NOT NULL,
  discount_cents BIGINT UNSIGNED NOT NULL,
  payable_cents BIGINT UNSIGNED NOT NULL,
  snapshot_digest BINARY(32) NOT NULL,
  created_at TIMESTAMP(6) NOT NULL,
  expires_at TIMESTAMP(6) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_quotes_user_idempotency (user_id, idempotency_key_hash),
  CONSTRAINT fk_quotes_user FOREIGN KEY (user_id) REFERENCES miniprogram_users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT chk_quotes_identity_version CHECK (identity_source_version > 0),
  CONSTRAINT chk_quotes_contact_name CHECK (
    CHAR_LENGTH(TRIM(contact_name_snapshot)) > 0
    AND OCTET_LENGTH(contact_name_snapshot) = OCTET_LENGTH(TRIM(contact_name_snapshot))
    AND OCTET_LENGTH(contact_name_snapshot) <= 64
  ),
  CONSTRAINT chk_quotes_contact_phone CHECK (
    REGEXP_LIKE(CONVERT(contact_phone_snapshot USING ascii), '^\\+[1-9][0-9]{0,14}$', 'c')
  ),
  CONSTRAINT chk_quotes_discount CHECK (discount_rate_percent >= 1 AND discount_rate_percent <= 100 AND discount_version > 0),
  CONSTRAINT chk_quotes_text_snapshots CHECK (
    CHAR_LENGTH(TRIM(store_name_snapshot)) > 0
    AND CHAR_LENGTH(TRIM(store_address_snapshot)) > 0
    AND CHAR_LENGTH(TRIM(pickup_point_snapshot)) > 0
  ),
  CONSTRAINT chk_quotes_pickup_minute CHECK (SECOND(pickup_time) = 0),
  CONSTRAINT chk_quotes_item_count CHECK (item_count > 0),
  CONSTRAINT chk_quotes_totals CHECK (original_subtotal_cents = discount_cents + payable_cents AND payable_cents > 0),
  CONSTRAINT chk_quotes_expiry CHECK (expires_at > created_at)
) ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
