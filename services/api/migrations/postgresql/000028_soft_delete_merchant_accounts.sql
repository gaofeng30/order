ALTER TABLE merchant_accounts
  ADD COLUMN deleted_at TIMESTAMPTZ(6) NULL,
  ADD COLUMN deleted_by_account_id NUMERIC(20,0) CHECK (deleted_by_account_id BETWEEN 0 AND 18446744073709551615) NULL,
  ADD CONSTRAINT fk_merchant_accounts_deleted_by FOREIGN KEY (deleted_by_account_id) REFERENCES merchant_accounts (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  ADD CONSTRAINT chk_merchant_accounts_deleted_group CHECK ((deleted_at IS NULL AND deleted_by_account_id IS NULL) OR (deleted_at IS NOT NULL AND deleted_by_account_id IS NOT NULL AND enabled=FALSE AND bound_user_id IS NULL AND bound_at IS NULL));
