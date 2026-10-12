CREATE TABLE notification_consents (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  order_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  kind ENUM('READY','REFUND_RESULT') NOT NULL,
  grant_sequence BIGINT UNSIGNED NOT NULL,
  decision ENUM('ACCEPTED','REJECTED') NOT NULL,
  template_config_version BIGINT UNSIGNED NOT NULL,
  idempotency_key_hash BINARY(32) NOT NULL,
  decided_at TIMESTAMP(6) NOT NULL,
  consumed_at TIMESTAMP(6) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_notification_consents_grant (order_id,kind,grant_sequence),
  UNIQUE KEY uq_notification_consents_user_idempotency (user_id,idempotency_key_hash),
  CONSTRAINT fk_notification_consents_order FOREIGN KEY (order_id) REFERENCES orders (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT fk_notification_consents_user FOREIGN KEY (user_id) REFERENCES miniprogram_users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT chk_notification_consents_values CHECK (grant_sequence>0 AND template_config_version>0 AND (decision='ACCEPTED' OR consumed_at IS NULL) AND (consumed_at IS NULL OR consumed_at>=decided_at))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
