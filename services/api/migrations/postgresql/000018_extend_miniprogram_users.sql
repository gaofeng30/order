ALTER TABLE miniprogram_users
  ADD COLUMN extra_phone BYTEA CHECK (octet_length(extra_phone) <= 16) NULL,
  ADD COLUMN extra_name VARCHAR(100) NULL,
  ADD COLUMN extra_name_key BYTEA CHECK (octet_length(extra_name_key) <= 400) NULL,
  ADD COLUMN extra_phone_set_at TIMESTAMPTZ(6) NULL,
  ADD COLUMN record_version NUMERIC(20,0) CHECK (record_version BETWEEN 0 AND 18446744073709551615) NOT NULL DEFAULT 1,
  ADD CONSTRAINT chk_miniprogram_users_primary_phone CHECK (primary_phone IS NULL OR (convert_from(primary_phone,'UTF8') ~ '^[+][1-9][0-9]{0,14}$')),
  ADD CONSTRAINT chk_miniprogram_users_extra_group CHECK ((extra_phone IS NULL AND extra_name IS NULL AND extra_name_key IS NULL AND extra_phone_set_at IS NULL) OR (extra_phone IS NOT NULL AND extra_name IS NOT NULL AND extra_name_key IS NOT NULL AND extra_phone_set_at IS NOT NULL)),
  ADD CONSTRAINT chk_miniprogram_users_extra_phone CHECK (extra_phone IS NULL OR (convert_from(extra_phone,'UTF8') ~ '^[+][1-9][0-9]{0,14}$')),
  ADD CONSTRAINT chk_miniprogram_users_extra_name CHECK (extra_name IS NULL OR (CHAR_LENGTH(extra_name) > 0 AND OCTET_LENGTH(extra_name_key) BETWEEN 1 AND 400)),
  ADD CONSTRAINT chk_miniprogram_users_record_version CHECK (record_version > 0);
