ALTER TABLE products
  ADD COLUMN meal_period ENUM('all','lunch','dinner') NOT NULL DEFAULT 'all' AFTER is_listed,
  ADD KEY idx_products_menu (category_id, is_listed, meal_period, sort_order, id);
