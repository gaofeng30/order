-- The existing name/key become the user's independent default pickup name.
-- Preserve all saved values; phone/time and name/key are separate optional pairs.
ALTER TABLE miniprogram_users
  DROP CONSTRAINT chk_miniprogram_users_extra_group,
  ADD CONSTRAINT chk_miniprogram_users_extra_phone_pair CHECK ((extra_phone IS NULL AND extra_phone_set_at IS NULL) OR (extra_phone IS NOT NULL AND extra_phone_set_at IS NOT NULL)),
  ADD CONSTRAINT chk_miniprogram_users_default_name_pair CHECK ((extra_name IS NULL AND extra_name_key IS NULL) OR (extra_name IS NOT NULL AND extra_name_key IS NOT NULL));
