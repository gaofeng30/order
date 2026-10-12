ALTER TABLE action_audits
  DROP CONSTRAINT fk_merchant_action_audits_account,
  DROP CONSTRAINT chk_merchant_action_audits_snapshot,
  DROP CONSTRAINT chk_merchant_action_audits_target,
  DROP CONSTRAINT chk_merchant_action_audits_actor,
  DROP CONSTRAINT chk_merchant_action_audits_text;
DROP INDEX idx_merchant_action_audits_account_time;
DROP INDEX idx_merchant_action_audits_actor_time;
ALTER TABLE action_audits RENAME COLUMN merchant_account_id TO actor_account_id;
ALTER TABLE action_audits RENAME COLUMN account_id_snapshot TO actor_account_id_snapshot;
ALTER TABLE action_audits RENAME COLUMN role_snapshot TO actor_role_snapshot;
ALTER TABLE action_audits RENAME COLUMN auth_version_snapshot TO actor_auth_version_snapshot;
ALTER TABLE action_audits RENAME COLUMN reason TO reason_code;
ALTER TABLE action_audits RENAME COLUMN idempotency_key_hash TO operation_key_hash;
ALTER TABLE action_audits
  ALTER COLUMN actor_user_id DROP NOT NULL,
  ADD COLUMN entry_kind TEXT CHECK (entry_kind IN ('LEGACY_EVIDENCE','COMMAND_RECEIPT','SYSTEM_EVIDENCE')) NULL,
  ADD COLUMN actor_kind TEXT CHECK (actor_kind IN ('USER','MERCHANT','SYSTEM','PROVIDER')) NULL,
  ADD COLUMN actor_scope_hash BYTEA CHECK (octet_length(actor_scope_hash)=32) NULL,
  ADD COLUMN target_key_hash BYTEA CHECK (octet_length(target_key_hash)=32) NULL,
  ADD COLUMN request_id_hash BYTEA CHECK (octet_length(request_id_hash)=32) NULL,
  ADD COLUMN before_state_json JSONB NULL,
  ADD COLUMN after_state_json JSONB NULL,
  ADD COLUMN response_json JSONB NULL;
