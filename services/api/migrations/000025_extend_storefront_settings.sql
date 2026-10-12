ALTER TABLE storefront_settings
  ADD COLUMN launch_image_object_key VARBINARY(1024) NULL AFTER launch_png_url,
  ADD COLUMN flavor_options_json JSON NOT NULL DEFAULT (JSON_ARRAY()) AFTER aspect_ratio,
  ADD COLUMN record_version BIGINT UNSIGNED NOT NULL DEFAULT 1 AFTER flavor_options_json;
