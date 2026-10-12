CREATE TABLE merchant_accounts (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  phone VARBINARY(16) NOT NULL,
  name TEXT NOT NULL,
  role ENUM('OWNER','SUBACCOUNT') NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  record_version BIGINT UNSIGNED NOT NULL DEFAULT 1,
  auth_version BIGINT UNSIGNED NOT NULL DEFAULT 1,
  bound_user_id BIGINT UNSIGNED NULL,
  bound_at TIMESTAMP(6) NULL,
  created_at TIMESTAMP(6) NOT NULL,
  updated_at TIMESTAMP(6) NOT NULL,
  created_by BIGINT UNSIGNED NULL,
  updated_by BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_merchant_accounts_phone (phone),
  UNIQUE KEY uq_merchant_accounts_bound_user (bound_user_id),
  KEY idx_merchant_accounts_role_enabled (role, enabled),
  CONSTRAINT fk_merchant_accounts_bound_user FOREIGN KEY (bound_user_id) REFERENCES miniprogram_users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT chk_merchant_accounts_phone CHECK (
    REGEXP_LIKE(CONVERT(phone USING ascii), '^\\+[1-9][0-9]{0,14}$', 'c')
  ),
  CONSTRAINT chk_merchant_accounts_name CHECK (
    CHAR_LENGTH(TRIM(name)) > 0
  ),
  CONSTRAINT chk_merchant_accounts_enabled CHECK (enabled IN (FALSE,TRUE)),
  CONSTRAINT chk_merchant_accounts_versions CHECK (record_version > 0 AND auth_version > 0),
  CONSTRAINT chk_merchant_accounts_binding_pair CHECK (
    (bound_user_id IS NULL AND bound_at IS NULL)
    OR (bound_user_id IS NOT NULL AND bound_at IS NOT NULL)
  )
) ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
