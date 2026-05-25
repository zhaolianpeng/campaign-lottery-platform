-- v2 migration: add user_items and first_recharge_records tables
CREATE TABLE IF NOT EXISTS first_recharge_records (
  user_id VARCHAR(48) NOT NULL PRIMARY KEY,
  claimed JSON NOT NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS user_items (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id VARCHAR(48) NOT NULL,
  item_type VARCHAR(32) NOT NULL,
  quantity INT NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  UNIQUE KEY uk_user_item (user_id, item_type),
  KEY idx_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- add pity_config column if not exists
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS pity_config JSON NULL AFTER campaign_summary;
