CREATE SEQUENCE staff_whitelist_id_seq AS BIGINT;
CREATE TABLE staff_whitelist (
  id NUMERIC(20,0) DEFAULT nextval('staff_whitelist_id_seq') CHECK (id BETWEEN 0 AND 18446744073709551615) NOT NULL ,
  phone BYTEA CHECK (octet_length(phone) <= 16) NOT NULL,
  name TEXT NOT NULL,
  name_key BYTEA CHECK (octet_length(name_key) <= 400) NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  record_version NUMERIC(20,0) CHECK (record_version BETWEEN 0 AND 18446744073709551615) NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ(6) NOT NULL,
  updated_at TIMESTAMPTZ(6) NOT NULL,
  PRIMARY KEY (id),
  CONSTRAINT uq_staff_whitelist_phone UNIQUE (phone),
  CONSTRAINT chk_staff_whitelist_phone CHECK (
    (convert_from(phone,'UTF8') ~ '^[+][1-9][0-9]{0,14}$')
  ),
  CONSTRAINT chk_staff_whitelist_name CHECK (
    CHAR_LENGTH(TRIM(name)) > 0
    AND OCTET_LENGTH(name) = OCTET_LENGTH(TRIM(name))
    AND OCTET_LENGTH(name_key) BETWEEN 1 AND 400
  ),
  CONSTRAINT chk_staff_whitelist_enabled CHECK (enabled IN (FALSE, TRUE)),
  CONSTRAINT chk_staff_whitelist_record_version CHECK (record_version > 0)
);
ALTER SEQUENCE staff_whitelist_id_seq OWNED BY staff_whitelist.id;
