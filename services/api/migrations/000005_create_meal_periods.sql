CREATE TABLE meal_periods (
  code ENUM('lunch','dinner') NOT NULL,
  cutoff_time TIME NOT NULL,
  pickup_start_time TIME NOT NULL,
  pickup_end_time TIME NOT NULL,
  interval_minutes SMALLINT UNSIGNED NOT NULL,
  PRIMARY KEY (code),
  CONSTRAINT chk_meal_periods_cutoff_minute CHECK (cutoff_time >= '00:00:00' AND cutoff_time <= '23:59:00' AND SECOND(cutoff_time) = 0),
  CONSTRAINT chk_meal_periods_start_minute CHECK (pickup_start_time >= '00:00:00' AND pickup_start_time <= '23:59:00' AND SECOND(pickup_start_time) = 0),
  CONSTRAINT chk_meal_periods_end_minute CHECK (pickup_end_time >= '00:00:00' AND pickup_end_time <= '23:59:00' AND SECOND(pickup_end_time) = 0),
  CONSTRAINT chk_meal_periods_interval CHECK (interval_minutes BETWEEN 1 AND 1440),
  CONSTRAINT chk_meal_periods_cutoff_order CHECK (cutoff_time <= pickup_start_time),
  CONSTRAINT chk_meal_periods_pickup_order CHECK (pickup_start_time <= pickup_end_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
