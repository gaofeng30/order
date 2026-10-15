CREATE TABLE service_dates (
  service_date DATE NOT NULL,
  is_open BOOLEAN NOT NULL,
  record_version NUMERIC(20,0) CHECK (record_version BETWEEN 0 AND 18446744073709551615) NOT NULL DEFAULT 1,
  updated_by_account_id NUMERIC(20,0) CHECK (updated_by_account_id BETWEEN 0 AND 18446744073709551615) NOT NULL,
  updated_at TIMESTAMPTZ(6) NOT NULL,
  PRIMARY KEY (service_date),
  CONSTRAINT fk_service_dates_account FOREIGN KEY (updated_by_account_id) REFERENCES merchant_accounts (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT chk_service_dates_open CHECK (is_open IN (FALSE,TRUE)),
  CONSTRAINT chk_service_dates_version CHECK (record_version > 0)
);
