ALTER TABLE products
  ADD COLUMN meal_period TEXT CHECK (meal_period IN ('all','lunch','dinner')) NOT NULL DEFAULT 'all';
CREATE INDEX idx_products_menu ON products (category_id, is_listed, meal_period, sort_order, id);
