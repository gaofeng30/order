CREATE SEQUENCE merchant_pc_sessions_id_seq AS BIGINT;
CREATE TABLE merchant_pc_sessions (
  id NUMERIC(20,0) DEFAULT nextval('merchant_pc_sessions_id_seq') CHECK (id BETWEEN 0 AND 18446744073709551615) NOT NULL ,
  approval_secret_hash BYTEA CHECK (octet_length(approval_secret_hash) = 32) NOT NULL,
  poll_secret_hash BYTEA CHECK (octet_length(poll_secret_hash) = 32) NOT NULL,
  state TEXT CHECK (state IN ('WAITING','APPROVED','CONSUMED')) NOT NULL,
  login_expires_at TIMESTAMPTZ(6) NOT NULL,
  approved_account_id NUMERIC(20,0) CHECK (approved_account_id BETWEEN 0 AND 18446744073709551615) NULL,
  approved_user_id NUMERIC(20,0) CHECK (approved_user_id BETWEEN 0 AND 18446744073709551615) NULL,
  approved_auth_version NUMERIC(20,0) CHECK (approved_auth_version BETWEEN 0 AND 18446744073709551615) NULL,
  approved_at TIMESTAMPTZ(6) NULL,
  consumed_at TIMESTAMPTZ(6) NULL,
  access_token_hash BYTEA CHECK (octet_length(access_token_hash) = 32) NULL,
  access_issued_at TIMESTAMPTZ(6) NULL,
  access_expires_at TIMESTAMPTZ(6) NULL,
  created_at TIMESTAMPTZ(6) NOT NULL,
  updated_at TIMESTAMPTZ(6) NOT NULL,
  PRIMARY KEY (id),
  CONSTRAINT uq_merchant_pc_sessions_approval_secret UNIQUE (approval_secret_hash),
  CONSTRAINT uq_merchant_pc_sessions_poll_secret UNIQUE (poll_secret_hash),
  CONSTRAINT uq_merchant_pc_sessions_access_token UNIQUE (access_token_hash),
  CONSTRAINT fk_merchant_pc_sessions_account FOREIGN KEY (approved_account_id) REFERENCES merchant_accounts (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT fk_merchant_pc_sessions_user FOREIGN KEY (approved_user_id) REFERENCES miniprogram_users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT chk_merchant_pc_sessions_login_expiry CHECK (login_expires_at=(created_at+INTERVAL '2 minute')),
  CONSTRAINT chk_merchant_pc_sessions_state CHECK ((state='WAITING' AND approved_account_id IS NULL AND approved_user_id IS NULL AND approved_auth_version IS NULL AND approved_at IS NULL AND consumed_at IS NULL AND access_token_hash IS NULL AND access_issued_at IS NULL AND access_expires_at IS NULL) OR (state='APPROVED' AND approved_account_id IS NOT NULL AND approved_user_id IS NOT NULL AND approved_auth_version IS NOT NULL AND approved_at IS NOT NULL AND approved_at BETWEEN created_at AND login_expires_at AND consumed_at IS NULL AND access_token_hash IS NULL AND access_issued_at IS NULL AND access_expires_at IS NULL) OR (state='CONSUMED' AND approved_account_id IS NOT NULL AND approved_user_id IS NOT NULL AND approved_auth_version IS NOT NULL AND approved_at IS NOT NULL AND approved_at BETWEEN created_at AND login_expires_at AND consumed_at IS NOT NULL AND consumed_at BETWEEN approved_at AND login_expires_at AND access_token_hash IS NOT NULL AND access_issued_at IS NOT NULL AND access_issued_at>=consumed_at AND access_expires_at IS NOT NULL AND access_expires_at=(access_issued_at+INTERVAL '12 hour'))),
  CONSTRAINT chk_merchant_pc_sessions_updated CHECK (updated_at>=created_at)
);
ALTER SEQUENCE merchant_pc_sessions_id_seq OWNED BY merchant_pc_sessions.id;
