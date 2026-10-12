CREATE TABLE discount_settings (
  id SMALLINT CHECK (id BETWEEN 0 AND 255) NOT NULL,
  rate_percent SMALLINT CHECK (rate_percent BETWEEN 0 AND 255) NOT NULL,
  discount_version NUMERIC(20,0) CHECK (discount_version BETWEEN 0 AND 18446744073709551615) NOT NULL,
  whitelist_version NUMERIC(20,0) CHECK (whitelist_version BETWEEN 0 AND 18446744073709551615) NOT NULL,
  updated_at TIMESTAMPTZ(6) NOT NULL,
  PRIMARY KEY (id),
  CONSTRAINT chk_discount_settings_singleton CHECK (id = 1),
  CONSTRAINT chk_discount_settings_rate CHECK (rate_percent >= 1 AND rate_percent <= 100),
  CONSTRAINT chk_discount_settings_versions CHECK (discount_version > 0 AND whitelist_version > 0)
);
