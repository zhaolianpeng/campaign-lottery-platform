-- Pity state persistence and draw idempotency
CREATE TABLE IF NOT EXISTS user_pity_state (
  id BIGINT NOT NULL AUTO_INCREMENT,
  user_id VARCHAR(32) NOT NULL,
  campaign_id VARCHAR(32) NOT NULL,
  soft_pity_count INT NOT NULL DEFAULT 0,
  hard_pity_count INT NOT NULL DEFAULT 0,
  up_pool_guarantee TINYINT(1) NOT NULL DEFAULT 0,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_pity_state (user_id, campaign_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Legacy rows use DEFAULT '' for request_id; multiple '' values violate UNIQUE.
UPDATE draw_records
SET request_id = CONCAT('legacy_', id)
WHERE request_id IS NULL OR request_id = '';

SET @dbname = DATABASE();
SET @preparedStatement = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM INFORMATION_SCHEMA.STATISTICS
      WHERE TABLE_SCHEMA = @dbname
        AND TABLE_NAME = 'draw_records'
        AND INDEX_NAME = 'uk_draw_records_request_id'
    ),
    'SELECT 1',
    'ALTER TABLE draw_records ADD UNIQUE KEY uk_draw_records_request_id (request_id)'
  )
);
PREPARE addUniqueIfNotExists FROM @preparedStatement;
EXECUTE addUniqueIfNotExists;
DEALLOCATE PREPARE addUniqueIfNotExists;
