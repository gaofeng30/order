ALTER TABLE storefront_settings
  DROP CHECK chk_storefront_settings_launch_group,
  DROP COLUMN launch_png_url,
  ADD CONSTRAINT chk_storefront_settings_launch_object_group CHECK ((launch_image_object_key IS NULL AND center_x IS NULL AND center_y IS NULL AND width_ratio IS NULL AND aspect_ratio IS NULL) OR (launch_image_object_key IS NOT NULL AND OCTET_LENGTH(launch_image_object_key) BETWEEN 1 AND 1024 AND center_x IS NOT NULL AND center_y IS NOT NULL AND width_ratio IS NOT NULL AND aspect_ratio IS NOT NULL)),
  ADD CONSTRAINT chk_storefront_settings_flavors CHECK (JSON_TYPE(flavor_options_json)='ARRAY'),
  ADD CONSTRAINT chk_storefront_settings_record_version CHECK (record_version > 0);
