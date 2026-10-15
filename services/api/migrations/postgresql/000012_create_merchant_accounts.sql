CREATE SEQUENCE merchant_accounts_id_seq AS BIGINT;
CREATE TABLE merchant_accounts (
  id NUMERIC(20,0) DEFAULT nextval('merchant_accounts_id_seq') CHECK (id BETWEEN 0 AND 18446744073709551615) NOT NULL ,
  phone BYTEA CHECK (octet_length(phone) <= 16) NOT NULL,
  name TEXT NOT NULL,
  role TEXT CHECK (role IN ('OWNER','SUBACCOUNT')) NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  record_version NUMERIC(20,0) CHECK (record_version BETWEEN 0 AND 18446744073709551615) NOT NULL DEFAULT 1,
  auth_version NUMERIC(20,0) CHECK (auth_version BETWEEN 0 AND 18446744073709551615) NOT NULL DEFAULT 1,
  bound_user_id NUMERIC(20,0) CHECK (bound_user_id BETWEEN 0 AND 18446744073709551615) NULL,
  bound_at TIMESTAMPTZ(6) NULL,
  created_at TIMESTAMPTZ(6) NOT NULL,
  updated_at TIMESTAMPTZ(6) NOT NULL,
  created_by NUMERIC(20,0) CHECK (created_by BETWEEN 0 AND 18446744073709551615) NULL,
  updated_by NUMERIC(20,0) CHECK (updated_by BETWEEN 0 AND 18446744073709551615) NULL,
  PRIMARY KEY (id),
  CONSTRAINT uq_merchant_accounts_phone UNIQUE (phone),
  CONSTRAINT uq_merchant_accounts_bound_user UNIQUE (bound_user_id),
  CONSTRAINT fk_merchant_accounts_bound_user FOREIGN KEY (bound_user_id) REFERENCES miniprogram_users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT chk_merchant_accounts_phone CHECK (
    (convert_from(phone,'UTF8') ~ '^[+][1-9][0-9]{0,14}$')
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
);
CREATE INDEX idx_merchant_accounts_role_enabled ON merchant_accounts (role, enabled);
ALTER SEQUENCE merchant_accounts_id_seq OWNED BY merchant_accounts.id;
