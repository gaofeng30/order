ALTER TABLE categories
  ADD COLUMN name_key BYTEA CHECK (octet_length(name_key) <= 400) NULL,
  ADD COLUMN record_version NUMERIC(20,0) CHECK (record_version BETWEEN 0 AND 18446744073709551615) NOT NULL DEFAULT 1;
