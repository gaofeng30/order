CREATE SEQUENCE categories_id_seq AS BIGINT;
CREATE TABLE categories (
  id NUMERIC(20,0) DEFAULT nextval('categories_id_seq') CHECK (id BETWEEN 0 AND 18446744073709551615) NOT NULL ,
  name VARCHAR(100) NOT NULL,
  sort_order BIGINT CHECK (sort_order BETWEEN 0 AND 4294967295) NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  PRIMARY KEY (id)
);
CREATE INDEX idx_categories_visibility ON categories (is_active, sort_order, id);
ALTER SEQUENCE categories_id_seq OWNED BY categories.id;
