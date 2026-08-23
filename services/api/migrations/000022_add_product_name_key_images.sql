ALTER TABLE products
  ADD COLUMN name_key VARBINARY(400) NULL AFTER name,
  ADD COLUMN images_json JSON NULL AFTER specification,
  ADD COLUMN record_version BIGINT UNSIGNED NOT NULL DEFAULT 1 AFTER meal_period;
