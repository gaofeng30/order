CREATE SEQUENCE products_id_seq AS BIGINT;
CREATE TABLE products (
  id NUMERIC(20,0) DEFAULT nextval('products_id_seq') CHECK (id BETWEEN 0 AND 18446744073709551615) NOT NULL ,
  category_id NUMERIC(20,0) CHECK (category_id BETWEEN 0 AND 18446744073709551615) NOT NULL,
  name VARCHAR(100) NOT NULL,
  description VARCHAR(1000) NOT NULL DEFAULT '',
  specification VARCHAR(255) NOT NULL DEFAULT '',
  price_cents BIGINT CHECK (price_cents BETWEEN 0 AND 4294967295) NOT NULL,
  sort_order BIGINT CHECK (sort_order BETWEEN 0 AND 4294967295) NOT NULL DEFAULT 0,
  is_listed BOOLEAN NOT NULL DEFAULT TRUE,
  PRIMARY KEY (id),
  CONSTRAINT fk_products_category FOREIGN KEY (category_id) REFERENCES categories (id) ON UPDATE RESTRICT ON DELETE RESTRICT
);
CREATE INDEX idx_products_catalog ON products (category_id, is_listed, sort_order, id);
ALTER SEQUENCE products_id_seq OWNED BY products.id;
