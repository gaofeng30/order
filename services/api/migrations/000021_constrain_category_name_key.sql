ALTER TABLE categories
  MODIFY COLUMN name_key VARBINARY(400) NOT NULL,
  ADD UNIQUE KEY uq_categories_name_key (name_key),
  ADD CONSTRAINT chk_categories_name CHECK (CHAR_LENGTH(name) > 0 AND OCTET_LENGTH(name)=OCTET_LENGTH(TRIM(name))),
  ADD CONSTRAINT chk_categories_record_version CHECK (record_version > 0);
