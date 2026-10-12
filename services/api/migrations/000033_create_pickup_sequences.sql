CREATE TABLE pickup_sequences (
  service_date DATE NOT NULL,
  last_number SMALLINT UNSIGNED NOT NULL,
  updated_at TIMESTAMP(6) NOT NULL,
  PRIMARY KEY (service_date),
  CONSTRAINT chk_pickup_sequences_number CHECK (last_number BETWEEN 0 AND 9999)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
