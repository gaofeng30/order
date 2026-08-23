ALTER TABLE miniprogram_users
  ADD COLUMN extra_phone VARBINARY(16) NULL AFTER primary_phone_bound_at,
  ADD COLUMN extra_name VARCHAR(100) NULL AFTER extra_phone,
  ADD COLUMN extra_name_key VARBINARY(400) NULL AFTER extra_name,
  ADD COLUMN extra_phone_set_at TIMESTAMP(6) NULL AFTER extra_name_key,
  ADD COLUMN record_version BIGINT UNSIGNED NOT NULL DEFAULT 1 AFTER extra_phone_set_at,
  ADD CONSTRAINT chk_miniprogram_users_primary_phone CHECK (primary_phone IS NULL OR REGEXP_LIKE(CONVERT(primary_phone USING ascii), '^\\+[1-9][0-9]{0,14}$', 'c')),
  ADD CONSTRAINT chk_miniprogram_users_extra_group CHECK ((extra_phone IS NULL AND extra_name IS NULL AND extra_name_key IS NULL AND extra_phone_set_at IS NULL) OR (extra_phone IS NOT NULL AND extra_name IS NOT NULL AND extra_name_key IS NOT NULL AND extra_phone_set_at IS NOT NULL)),
  ADD CONSTRAINT chk_miniprogram_users_extra_phone CHECK (extra_phone IS NULL OR REGEXP_LIKE(CONVERT(extra_phone USING ascii), '^\\+[1-9][0-9]{0,14}$', 'c')),
  ADD CONSTRAINT chk_miniprogram_users_extra_name CHECK (extra_name IS NULL OR (CHAR_LENGTH(extra_name) > 0 AND OCTET_LENGTH(extra_name_key) BETWEEN 1 AND 400)),
  ADD CONSTRAINT chk_miniprogram_users_record_version CHECK (record_version > 0);
