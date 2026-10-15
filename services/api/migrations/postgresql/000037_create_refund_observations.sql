CREATE SEQUENCE refund_observations_id_seq AS BIGINT;
CREATE TABLE refund_observations (
  id NUMERIC(20,0) DEFAULT nextval('refund_observations_id_seq') CHECK (id BETWEEN 0 AND 18446744073709551615) NOT NULL ,
  refund_id NUMERIC(20,0) CHECK (refund_id BETWEEN 0 AND 18446744073709551615) NOT NULL,
  dedupe_key BYTEA CHECK (octet_length(dedupe_key) = 32) NOT NULL,
  source TEXT CHECK (source IN ('CALLBACK','ACTIVE_QUERY')) NOT NULL,
  provider_event_id BYTEA CHECK (octet_length(provider_event_id) <= 128) NULL,
  out_refund_no BYTEA CHECK (octet_length(out_refund_no) <= 64) NOT NULL,
  provider_refund_id BYTEA CHECK (octet_length(provider_refund_id) <= 128) NULL,
  provider_state TEXT CHECK (provider_state IN ('PROCESSING','SUCCESS','CLOSED')) NOT NULL,
  validation TEXT CHECK (validation IN ('MATCH','MISMATCH')) NOT NULL,
  mismatch_code VARCHAR(64) NULL,
  success_time TIMESTAMPTZ(6) NULL,
  received_at TIMESTAMPTZ(6) NOT NULL,
  apply_state TEXT CHECK (apply_state IN ('NEW','APPLIED','DEFERRED')) NOT NULL,
  apply_reason_code VARCHAR(64) NULL,
  applied_at TIMESTAMPTZ(6) NULL,
  record_version NUMERIC(20,0) CHECK (record_version BETWEEN 0 AND 18446744073709551615) NOT NULL DEFAULT 1,
  PRIMARY KEY (id),
  CONSTRAINT uq_refund_observations_dedupe UNIQUE (dedupe_key),
  CONSTRAINT fk_refund_observations_refund FOREIGN KEY (refund_id) REFERENCES refunds (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT chk_refund_observations_identity CHECK (OCTET_LENGTH(out_refund_no) BETWEEN 1 AND 64 AND (provider_event_id IS NULL OR OCTET_LENGTH(provider_event_id) BETWEEN 1 AND 128) AND (provider_refund_id IS NULL OR OCTET_LENGTH(provider_refund_id) BETWEEN 1 AND 128)),
  CONSTRAINT chk_refund_observations_validation CHECK ((validation='MATCH' AND mismatch_code IS NULL) OR (validation='MISMATCH' AND mismatch_code IS NOT NULL AND CHAR_LENGTH(TRIM(mismatch_code))>0)),
  CONSTRAINT chk_refund_observations_success CHECK ((provider_state='SUCCESS' AND provider_refund_id IS NOT NULL AND success_time IS NOT NULL) OR (provider_state IN ('PROCESSING','CLOSED') AND success_time IS NULL)),
  CONSTRAINT chk_refund_observations_apply CHECK ((apply_state='NEW' AND apply_reason_code IS NULL AND applied_at IS NULL) OR (apply_state='APPLIED' AND apply_reason_code IS NULL AND applied_at IS NOT NULL) OR (apply_state='DEFERRED' AND apply_reason_code IS NOT NULL AND CHAR_LENGTH(TRIM(apply_reason_code))>0 AND applied_at IS NULL)),
  CONSTRAINT chk_refund_observations_record_version CHECK (record_version>0)
);
ALTER SEQUENCE refund_observations_id_seq OWNED BY refund_observations.id;
