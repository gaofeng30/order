CREATE TABLE storefront_settings (
  id TINYINT UNSIGNED NOT NULL,
  store_name TEXT NOT NULL,
  store_address TEXT NOT NULL,
  pickup_point TEXT NOT NULL,
  announcement TEXT NOT NULL,
  business_status ENUM('open','closed','cutoff') NOT NULL,
  launch_png_url TEXT NULL,
  center_x DOUBLE NULL,
  center_y DOUBLE NULL,
  width_ratio DOUBLE NULL,
  aspect_ratio DOUBLE NULL,
  PRIMARY KEY (id),
  CONSTRAINT chk_storefront_settings_singleton CHECK (id = 1),
  CONSTRAINT chk_storefront_settings_store_name CHECK (CHAR_LENGTH(TRIM(store_name)) > 0 AND OCTET_LENGTH(store_name) = OCTET_LENGTH(TRIM(store_name))),
  CONSTRAINT chk_storefront_settings_store_address CHECK (CHAR_LENGTH(TRIM(store_address)) > 0 AND OCTET_LENGTH(store_address) = OCTET_LENGTH(TRIM(store_address))),
  CONSTRAINT chk_storefront_settings_pickup_point CHECK (CHAR_LENGTH(TRIM(pickup_point)) > 0 AND OCTET_LENGTH(pickup_point) = OCTET_LENGTH(TRIM(pickup_point))),
  CONSTRAINT chk_storefront_settings_announcement CHECK (CHAR_LENGTH(announcement) <= 1000),
  CONSTRAINT chk_storefront_settings_launch_group CHECK (
    (launch_png_url IS NULL AND center_x IS NULL AND center_y IS NULL AND width_ratio IS NULL AND aspect_ratio IS NULL)
    OR
    (launch_png_url IS NOT NULL AND center_x IS NOT NULL AND center_y IS NOT NULL AND width_ratio IS NOT NULL AND aspect_ratio IS NOT NULL)
  ),
  CONSTRAINT chk_storefront_settings_center_x CHECK (center_x BETWEEN 0 AND 1),
  CONSTRAINT chk_storefront_settings_center_y CHECK (center_y BETWEEN 0 AND 1),
  CONSTRAINT chk_storefront_settings_width_ratio CHECK (width_ratio > 0 AND width_ratio <= 1),
  CONSTRAINT chk_storefront_settings_aspect_ratio CHECK (aspect_ratio > 0)
) ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
