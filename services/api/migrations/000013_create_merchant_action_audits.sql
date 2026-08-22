CREATE TABLE merchant_action_audits (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  merchant_account_id BIGINT UNSIGNED NULL,
  account_id_snapshot BIGINT UNSIGNED NULL,
  role_snapshot ENUM('OWNER','SUBACCOUNT') NULL,
  auth_version_snapshot BIGINT UNSIGNED NULL,
  actor_user_id BIGINT UNSIGNED NOT NULL,
  action VARCHAR(64) NOT NULL,
  result ENUM('SUCCEEDED','REJECTED') NOT NULL,
  reason VARCHAR(64) NOT NULL,
  target_type VARCHAR(64) NULL,
  target_id BIGINT UNSIGNED NULL,
  request_id VARBINARY(64) NOT NULL,
  idempotency_key_hash BINARY(32) NULL,
  state_before VARCHAR(64) NULL,
  state_after VARCHAR(64) NULL,
  occurred_at TIMESTAMP(6) NOT NULL,
  PRIMARY KEY (id),
  KEY idx_merchant_action_audits_account_time (merchant_account_id, occurred_at),
  KEY idx_merchant_action_audits_actor_time (actor_user_id, occurred_at),
  CONSTRAINT fk_merchant_action_audits_account FOREIGN KEY (merchant_account_id) REFERENCES merchant_accounts (id) ON UPDATE RESTRICT ON DELETE SET NULL,
  CONSTRAINT chk_merchant_action_audits_snapshot CHECK (
    (
      account_id_snapshot IS NULL
      AND role_snapshot IS NULL
      AND auth_version_snapshot IS NULL
    )
    OR (
      account_id_snapshot IS NOT NULL
      AND account_id_snapshot > 0
      AND role_snapshot IS NOT NULL
      AND auth_version_snapshot IS NOT NULL
      AND auth_version_snapshot > 0
    )
  ),
  CONSTRAINT chk_merchant_action_audits_target CHECK (
    (target_type IS NULL AND target_id IS NULL)
    OR (target_type IS NOT NULL AND CHAR_LENGTH(TRIM(target_type)) > 0 AND target_id IS NOT NULL AND target_id > 0)
  ),
  CONSTRAINT chk_merchant_action_audits_actor CHECK (actor_user_id > 0),
  CONSTRAINT chk_merchant_action_audits_text CHECK (
    CHAR_LENGTH(TRIM(action)) > 0
    AND CHAR_LENGTH(TRIM(reason)) > 0
    AND OCTET_LENGTH(request_id) BETWEEN 1 AND 64
    AND (state_before IS NULL OR CHAR_LENGTH(TRIM(state_before)) > 0)
    AND (state_after IS NULL OR CHAR_LENGTH(TRIM(state_after)) > 0)
  )
) ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
