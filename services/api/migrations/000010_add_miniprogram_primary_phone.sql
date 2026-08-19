ALTER TABLE miniprogram_users
  ADD COLUMN primary_phone VARBINARY(16) NULL,
  ADD COLUMN primary_phone_bound_at TIMESTAMP(6) NULL,
  ADD UNIQUE KEY uq_miniprogram_users_primary_phone (primary_phone),
  ADD CONSTRAINT chk_miniprogram_users_primary_phone_pair CHECK (
    (primary_phone IS NULL AND primary_phone_bound_at IS NULL)
    OR (primary_phone IS NOT NULL AND primary_phone_bound_at IS NOT NULL)
  );
