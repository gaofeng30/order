CREATE TABLE service_dates (
  service_date DATE NOT NULL,
  is_open BOOLEAN NOT NULL,
  record_version BIGINT UNSIGNED NOT NULL DEFAULT 1,
  updated_by_account_id BIGINT UNSIGNED NOT NULL,
  updated_at TIMESTAMP(6) NOT NULL,
  PRIMARY KEY (service_date),
  CONSTRAINT fk_service_dates_account FOREIGN KEY (updated_by_account_id) REFERENCES merchant_accounts (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT chk_service_dates_open CHECK (is_open IN (FALSE,TRUE)),
  CONSTRAINT chk_service_dates_version CHECK (record_version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
