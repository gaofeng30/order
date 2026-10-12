ALTER TABLE categories
  ALTER COLUMN name_key TYPE BYTEA,
  ALTER COLUMN name_key SET NOT NULL,
  ADD CONSTRAINT uq_categories_name_key UNIQUE (name_key),
  ADD CONSTRAINT chk_categories_name CHECK (CHAR_LENGTH(name) > 0 AND OCTET_LENGTH(name)=OCTET_LENGTH(TRIM(name))),
  ADD CONSTRAINT chk_categories_record_version CHECK (record_version > 0);
