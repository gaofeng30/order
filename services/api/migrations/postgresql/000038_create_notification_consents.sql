CREATE SEQUENCE notification_consents_id_seq AS BIGINT;
CREATE TABLE notification_consents (
  id NUMERIC(20,0) DEFAULT nextval('notification_consents_id_seq') CHECK (id BETWEEN 0 AND 18446744073709551615) NOT NULL ,
  order_id NUMERIC(20,0) CHECK (order_id BETWEEN 0 AND 18446744073709551615) NOT NULL,
  user_id NUMERIC(20,0) CHECK (user_id BETWEEN 0 AND 18446744073709551615) NOT NULL,
  kind TEXT CHECK (kind IN ('READY','REFUND_RESULT')) NOT NULL,
  grant_sequence NUMERIC(20,0) CHECK (grant_sequence BETWEEN 0 AND 18446744073709551615) NOT NULL,
  decision TEXT CHECK (decision IN ('ACCEPTED','REJECTED')) NOT NULL,
  template_config_version NUMERIC(20,0) CHECK (template_config_version BETWEEN 0 AND 18446744073709551615) NOT NULL,
  idempotency_key_hash BYTEA CHECK (octet_length(idempotency_key_hash) = 32) NOT NULL,
  decided_at TIMESTAMPTZ(6) NOT NULL,
  consumed_at TIMESTAMPTZ(6) NULL,
  PRIMARY KEY (id),
  CONSTRAINT uq_notification_consents_grant UNIQUE (order_id,kind,grant_sequence),
  CONSTRAINT uq_notification_consents_user_idempotency UNIQUE (user_id,idempotency_key_hash),
  CONSTRAINT fk_notification_consents_order FOREIGN KEY (order_id) REFERENCES orders (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT fk_notification_consents_user FOREIGN KEY (user_id) REFERENCES miniprogram_users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT chk_notification_consents_values CHECK (grant_sequence>0 AND template_config_version>0 AND (decision='ACCEPTED' OR consumed_at IS NULL) AND (consumed_at IS NULL OR consumed_at>=decided_at))
);
ALTER SEQUENCE notification_consents_id_seq OWNED BY notification_consents.id;
