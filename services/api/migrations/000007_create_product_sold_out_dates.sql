CREATE TABLE product_sold_out_dates (
  service_date DATE NOT NULL,
  product_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (service_date, product_id),
  CONSTRAINT fk_product_sold_out_dates_product FOREIGN KEY (product_id) REFERENCES products (id) ON UPDATE RESTRICT ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
