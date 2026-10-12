UPDATE products SET name_key=convert_to(name,'UTF8'),images_json='[]'::jsonb;
