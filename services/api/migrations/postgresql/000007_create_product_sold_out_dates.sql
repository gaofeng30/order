CREATE TABLE product_sold_out_dates (
  service_date DATE NOT NULL,
  product_id NUMERIC(20,0) CHECK (product_id BETWEEN 0 AND 18446744073709551615) NOT NULL,
  PRIMARY KEY (service_date, product_id),
  CONSTRAINT fk_product_sold_out_dates_product FOREIGN KEY (product_id) REFERENCES products (id) ON UPDATE RESTRICT ON DELETE RESTRICT
);
