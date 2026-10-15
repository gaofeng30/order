CREATE TABLE meal_periods (
  code TEXT CHECK (code IN ('lunch','dinner')) NOT NULL,
  cutoff_time TIME NOT NULL,
  pickup_start_time TIME NOT NULL,
  pickup_end_time TIME NOT NULL,
  interval_minutes INTEGER CHECK (interval_minutes BETWEEN 0 AND 65535) NOT NULL,
  PRIMARY KEY (code),
  CONSTRAINT chk_meal_periods_cutoff_minute CHECK (cutoff_time >= '00:00:00' AND cutoff_time <= '23:59:00' AND EXTRACT(SECOND FROM cutoff_time) = 0),
  CONSTRAINT chk_meal_periods_start_minute CHECK (pickup_start_time >= '00:00:00' AND pickup_start_time <= '23:59:00' AND EXTRACT(SECOND FROM pickup_start_time) = 0),
  CONSTRAINT chk_meal_periods_end_minute CHECK (pickup_end_time >= '00:00:00' AND pickup_end_time <= '23:59:00' AND EXTRACT(SECOND FROM pickup_end_time) = 0),
  CONSTRAINT chk_meal_periods_interval CHECK (interval_minutes BETWEEN 1 AND 1440),
  CONSTRAINT chk_meal_periods_cutoff_order CHECK (cutoff_time <= pickup_start_time),
  CONSTRAINT chk_meal_periods_pickup_order CHECK (pickup_start_time <= pickup_end_time)
);
