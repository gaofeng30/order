CREATE SEQUENCE quotes_id_seq AS BIGINT;
CREATE TABLE quotes (
  id NUMERIC(20,0) DEFAULT nextval('quotes_id_seq') CHECK (id BETWEEN 0 AND 18446744073709551615) NOT NULL ,
  user_id NUMERIC(20,0) CHECK (user_id BETWEEN 0 AND 18446744073709551615) NOT NULL,
  contact_name_snapshot VARCHAR(64) NOT NULL,
  contact_phone_snapshot BYTEA CHECK (octet_length(contact_phone_snapshot) <= 16) NOT NULL,
  idempotency_key_hash BYTEA CHECK (octet_length(idempotency_key_hash) = 32) NOT NULL,
  request_digest BYTEA CHECK (octet_length(request_digest) = 32) NOT NULL,
  identity_kind TEXT CHECK (identity_kind IN ('STAFF','VISITOR')) NOT NULL,
  identity_source_version NUMERIC(20,0) CHECK (identity_source_version BETWEEN 0 AND 18446744073709551615) NOT NULL,
  discount_rate_percent SMALLINT CHECK (discount_rate_percent BETWEEN 0 AND 255) NOT NULL,
  discount_version NUMERIC(20,0) CHECK (discount_version BETWEEN 0 AND 18446744073709551615) NOT NULL,
  store_name_snapshot TEXT NOT NULL,
  store_address_snapshot TEXT NOT NULL,
  pickup_point_snapshot TEXT NOT NULL,
  pickup_date DATE NOT NULL,
  pickup_time TIME NOT NULL,
  meal_period TEXT CHECK (meal_period IN ('lunch','dinner')) NOT NULL,
  order_note TEXT NOT NULL,
  item_count INTEGER CHECK (item_count BETWEEN 0 AND 65535) NOT NULL,
  original_subtotal_cents NUMERIC(20,0) CHECK (original_subtotal_cents BETWEEN 0 AND 18446744073709551615) NOT NULL,
  discount_cents NUMERIC(20,0) CHECK (discount_cents BETWEEN 0 AND 18446744073709551615) NOT NULL,
  payable_cents NUMERIC(20,0) CHECK (payable_cents BETWEEN 0 AND 18446744073709551615) NOT NULL,
  snapshot_digest BYTEA CHECK (octet_length(snapshot_digest) = 32) NOT NULL,
  created_at TIMESTAMPTZ(6) NOT NULL,
  expires_at TIMESTAMPTZ(6) NOT NULL,
  PRIMARY KEY (id),
  CONSTRAINT uq_quotes_user_idempotency UNIQUE (user_id, idempotency_key_hash),
  CONSTRAINT fk_quotes_user FOREIGN KEY (user_id) REFERENCES miniprogram_users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT chk_quotes_identity_version CHECK (identity_source_version > 0),
  CONSTRAINT chk_quotes_contact_name CHECK (
    CHAR_LENGTH(TRIM(contact_name_snapshot)) > 0
    AND OCTET_LENGTH(contact_name_snapshot) = OCTET_LENGTH(TRIM(contact_name_snapshot))
    AND OCTET_LENGTH(contact_name_snapshot) <= 64
  ),
  CONSTRAINT chk_quotes_contact_phone CHECK (
    (convert_from(contact_phone_snapshot,'UTF8') ~ '^[+][1-9][0-9]{0,14}$')
  ),
  CONSTRAINT chk_quotes_discount CHECK (discount_rate_percent >= 1 AND discount_rate_percent <= 100 AND discount_version > 0),
  CONSTRAINT chk_quotes_text_snapshots CHECK (
    CHAR_LENGTH(TRIM(store_name_snapshot)) > 0
    AND CHAR_LENGTH(TRIM(store_address_snapshot)) > 0
    AND CHAR_LENGTH(TRIM(pickup_point_snapshot)) > 0
  ),
  CONSTRAINT chk_quotes_pickup_minute CHECK (EXTRACT(SECOND FROM pickup_time) = 0),
  CONSTRAINT chk_quotes_item_count CHECK (item_count > 0),
  CONSTRAINT chk_quotes_totals CHECK (original_subtotal_cents = discount_cents + payable_cents AND payable_cents > 0),
  CONSTRAINT chk_quotes_expiry CHECK (expires_at > created_at)
);
ALTER SEQUENCE quotes_id_seq OWNED BY quotes.id;
