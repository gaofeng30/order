UPDATE products SET name_key=CONVERT(name USING binary),images_json=JSON_ARRAY();
