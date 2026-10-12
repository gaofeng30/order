CREATE TABLE staff_whitelist (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  phone VARBINARY(16) NOT NULL,
  name TEXT NOT NULL,
  name_key VARBINARY(400) NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  record_version BIGINT UNSIGNED NOT NULL DEFAULT 1,
  created_at TIMESTAMP(6) NOT NULL,
  updated_at TIMESTAMP(6) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_staff_whitelist_phone (phone),
  CONSTRAINT chk_staff_whitelist_phone CHECK (
    REGEXP_LIKE(CONVERT(phone USING ascii), '^\\+[1-9][0-9]{0,14}$', 'c')
  ),
  CONSTRAINT chk_staff_whitelist_name CHECK (
    CHAR_LENGTH(TRIM(name)) > 0
    AND OCTET_LENGTH(name) = OCTET_LENGTH(TRIM(name))
    AND OCTET_LENGTH(name_key) BETWEEN 1 AND 400
  ),
  CONSTRAINT chk_staff_whitelist_enabled CHECK (enabled IN (FALSE, TRUE)),
  CONSTRAINT chk_staff_whitelist_record_version CHECK (record_version > 0)
) ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
