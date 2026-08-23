ALTER TABLE merchant_accounts
  ADD COLUMN deleted_at TIMESTAMP(6) NULL AFTER bound_at,
  ADD COLUMN deleted_by_account_id BIGINT UNSIGNED NULL AFTER deleted_at,
  ADD CONSTRAINT fk_merchant_accounts_deleted_by FOREIGN KEY (deleted_by_account_id) REFERENCES merchant_accounts (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  ADD CONSTRAINT chk_merchant_accounts_deleted_group CHECK ((deleted_at IS NULL AND deleted_by_account_id IS NULL) OR (deleted_at IS NOT NULL AND deleted_by_account_id IS NOT NULL AND enabled=FALSE AND bound_user_id IS NULL AND bound_at IS NULL));
