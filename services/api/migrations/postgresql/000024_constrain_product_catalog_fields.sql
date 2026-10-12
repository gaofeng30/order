ALTER TABLE products
  ALTER COLUMN name_key SET NOT NULL,
  ALTER COLUMN images_json SET NOT NULL,
  ADD CONSTRAINT uq_products_name_key UNIQUE (name_key),
  ADD CONSTRAINT chk_products_name CHECK (char_length(name)>0 AND octet_length(name)=octet_length(trim(name))),
  ADD CONSTRAINT chk_products_price CHECK (price_cents>0),
  ADD CONSTRAINT chk_products_record_version CHECK (record_version>0),
  ADD CONSTRAINT chk_products_images CHECK (CASE WHEN jsonb_typeof(images_json)='array' THEN
    jsonb_array_length(images_json) BETWEEN 0 AND 3
    AND (jsonb_array_length(images_json)<1 OR COALESCE((jsonb_typeof(images_json->0)='object' AND images_json->0=jsonb_build_object('object_key',images_json->0->'object_key') AND jsonb_typeof(images_json->0->'object_key')='string' AND char_length(images_json->0->>'object_key') BETWEEN 1 AND 1024),FALSE))
    AND (jsonb_array_length(images_json)<2 OR COALESCE((jsonb_typeof(images_json->1)='object' AND images_json->1=jsonb_build_object('object_key',images_json->1->'object_key') AND jsonb_typeof(images_json->1->'object_key')='string' AND char_length(images_json->1->>'object_key') BETWEEN 1 AND 1024 AND images_json->1->>'object_key'<>images_json->0->>'object_key'),FALSE))
    AND (jsonb_array_length(images_json)<3 OR COALESCE((jsonb_typeof(images_json->2)='object' AND images_json->2=jsonb_build_object('object_key',images_json->2->'object_key') AND jsonb_typeof(images_json->2->'object_key')='string' AND char_length(images_json->2->>'object_key') BETWEEN 1 AND 1024 AND images_json->2->>'object_key'<>images_json->0->>'object_key' AND images_json->2->>'object_key'<>images_json->1->>'object_key'),FALSE))
    ELSE FALSE END);
