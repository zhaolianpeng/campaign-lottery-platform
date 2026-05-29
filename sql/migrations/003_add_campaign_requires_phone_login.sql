SET @dbname = DATABASE();
SET @preparedStatement = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM INFORMATION_SCHEMA.COLUMNS
      WHERE TABLE_SCHEMA = @dbname
        AND TABLE_NAME = 'campaigns'
        AND COLUMN_NAME = 'requires_phone_login'
    ),
    'SELECT 1',
    'ALTER TABLE campaigns ADD COLUMN requires_phone_login TINYINT(1) NOT NULL DEFAULT 0 AFTER daily_draw_limit'
  )
);
PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;
