ALTER TABLE storefront_settings
  ADD COLUMN launch_image_object_key BYTEA CHECK (octet_length(launch_image_object_key) <= 1024) NULL,
  ADD COLUMN flavor_options_json JSONB NOT NULL DEFAULT ('[]'::jsonb),
  ADD COLUMN record_version NUMERIC(20,0) CHECK (record_version BETWEEN 0 AND 18446744073709551615) NOT NULL DEFAULT 1;
