CREATE SEQUENCE merchant_action_audits_id_seq AS BIGINT;
CREATE TABLE merchant_action_audits (
  id NUMERIC(20,0) DEFAULT nextval('merchant_action_audits_id_seq') CHECK (id BETWEEN 0 AND 18446744073709551615) NOT NULL ,
  merchant_account_id NUMERIC(20,0) CHECK (merchant_account_id BETWEEN 0 AND 18446744073709551615) NULL,
  account_id_snapshot NUMERIC(20,0) CHECK (account_id_snapshot BETWEEN 0 AND 18446744073709551615) NULL,
  role_snapshot TEXT CHECK (role_snapshot IN ('OWNER','SUBACCOUNT')) NULL,
  auth_version_snapshot NUMERIC(20,0) CHECK (auth_version_snapshot BETWEEN 0 AND 18446744073709551615) NULL,
  actor_user_id NUMERIC(20,0) CHECK (actor_user_id BETWEEN 0 AND 18446744073709551615) NOT NULL,
  action VARCHAR(64) NOT NULL,
  result TEXT CHECK (result IN ('SUCCEEDED','REJECTED')) NOT NULL,
  reason VARCHAR(64) NOT NULL,
  target_type VARCHAR(64) NULL,
  target_id NUMERIC(20,0) CHECK (target_id BETWEEN 0 AND 18446744073709551615) NULL,
  request_id BYTEA CHECK (octet_length(request_id) <= 64) NOT NULL,
  idempotency_key_hash BYTEA CHECK (octet_length(idempotency_key_hash) = 32) NULL,
  state_before VARCHAR(64) NULL,
  state_after VARCHAR(64) NULL,
  occurred_at TIMESTAMPTZ(6) NOT NULL,
  PRIMARY KEY (id),
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
);
CREATE INDEX idx_merchant_action_audits_account_time ON merchant_action_audits (merchant_account_id, occurred_at);
CREATE INDEX idx_merchant_action_audits_actor_time ON merchant_action_audits (actor_user_id, occurred_at);
ALTER SEQUENCE merchant_action_audits_id_seq OWNED BY merchant_action_audits.id;
