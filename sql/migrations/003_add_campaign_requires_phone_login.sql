ALTER TABLE campaigns
  ADD COLUMN requires_phone_login TINYINT(1) NOT NULL DEFAULT 0 AFTER daily_draw_limit;
