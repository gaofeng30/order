CREATE SEQUENCE notification_outbox_id_seq AS BIGINT;
CREATE TABLE notification_outbox (
  id NUMERIC(20,0) DEFAULT nextval('notification_outbox_id_seq') CHECK (id BETWEEN 0 AND 18446744073709551615) NOT NULL ,
  order_id NUMERIC(20,0) CHECK (order_id BETWEEN 0 AND 18446744073709551615) NOT NULL,
  consent_id NUMERIC(20,0) CHECK (consent_id BETWEEN 0 AND 18446744073709551615) NOT NULL,
  kind TEXT CHECK (kind IN ('READY','REFUND_RESULT')) NOT NULL,
  recipient_user_id NUMERIC(20,0) CHECK (recipient_user_id BETWEEN 0 AND 18446744073709551615) NOT NULL,
  immutable_message_json JSONB NOT NULL,
  template_config_version NUMERIC(20,0) CHECK (template_config_version BETWEEN 0 AND 18446744073709551615) NOT NULL,
  state TEXT CHECK (state IN ('PENDING','IN_FLIGHT','SENT','FAILED_PERMANENT')) NOT NULL,
  attempt_count INTEGER CHECK (attempt_count BETWEEN 0 AND 65535) NOT NULL DEFAULT 0,
  next_attempt_at TIMESTAMPTZ(6) NULL,
  lease_owner BYTEA CHECK (octet_length(lease_owner) = 16) NULL,
  lease_expires_at TIMESTAMPTZ(6) NULL,
  record_version NUMERIC(20,0) CHECK (record_version BETWEEN 0 AND 18446744073709551615) NOT NULL DEFAULT 1,
  provider_message_id BYTEA CHECK (octet_length(provider_message_id) <= 128) NULL,
  last_error_code VARCHAR(64) NULL,
  created_at TIMESTAMPTZ(6) NOT NULL,
  sent_at TIMESTAMPTZ(6) NULL,
  PRIMARY KEY (id),
  CONSTRAINT uq_notification_outbox_consent UNIQUE (consent_id),
  CONSTRAINT uq_notification_outbox_order_kind UNIQUE (order_id,kind),
  CONSTRAINT fk_notification_outbox_order FOREIGN KEY (order_id) REFERENCES orders (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT fk_notification_outbox_consent FOREIGN KEY (consent_id) REFERENCES notification_consents (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT fk_notification_outbox_user FOREIGN KEY (recipient_user_id) REFERENCES miniprogram_users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT chk_notification_outbox_values CHECK (jsonb_typeof(immutable_message_json)='object' AND template_config_version>0 AND record_version>0),
  CONSTRAINT chk_notification_outbox_state CHECK ((state='PENDING' AND next_attempt_at IS NOT NULL AND lease_owner IS NULL AND lease_expires_at IS NULL AND sent_at IS NULL AND provider_message_id IS NULL) OR (state='IN_FLIGHT' AND lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL AND sent_at IS NULL AND provider_message_id IS NULL) OR (state='SENT' AND sent_at IS NOT NULL AND lease_owner IS NULL AND lease_expires_at IS NULL AND last_error_code IS NULL) OR (state='FAILED_PERMANENT' AND last_error_code IS NOT NULL AND CHAR_LENGTH(TRIM(last_error_code))>0 AND lease_owner IS NULL AND lease_expires_at IS NULL AND sent_at IS NULL AND provider_message_id IS NULL))
);
CREATE INDEX idx_notification_outbox_due ON notification_outbox (state,next_attempt_at,id);
ALTER SEQUENCE notification_outbox_id_seq OWNED BY notification_outbox.id;
