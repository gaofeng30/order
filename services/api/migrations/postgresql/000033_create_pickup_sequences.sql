CREATE TABLE pickup_sequences (
  service_date DATE NOT NULL,
  last_number INTEGER CHECK (last_number BETWEEN 0 AND 65535) NOT NULL,
  updated_at TIMESTAMPTZ(6) NOT NULL,
  PRIMARY KEY (service_date),
  CONSTRAINT chk_pickup_sequences_number CHECK (last_number BETWEEN 0 AND 9999)
);
