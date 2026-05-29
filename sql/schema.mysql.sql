-- Baseline schema for the migration runner.
-- The active database is selected by the migration command before this file runs.

CREATE TABLE IF NOT EXISTS users (
  id VARCHAR(32) NOT NULL,
  nickname VARCHAR(64) NOT NULL,
  mobile VARCHAR(32) NOT NULL DEFAULT '',
  status VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_users_mobile (mobile)
);

CREATE TABLE IF NOT EXISTS user_sessions (
  token VARCHAR(64) NOT NULL,
  user_id VARCHAR(32) NOT NULL,
  expires_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (token),
  KEY idx_user_sessions_user_id (user_id)
);

CREATE TABLE IF NOT EXISTS admin_users (
  id BIGINT NOT NULL AUTO_INCREMENT,
  username VARCHAR(64) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  display_name VARCHAR(64) NOT NULL DEFAULT '',
  status VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_admin_users_username (username)
);

CREATE TABLE IF NOT EXISTS campaigns (
  id VARCHAR(32) NOT NULL,
  name VARCHAR(128) NOT NULL,
  slug VARCHAR(64) NOT NULL,
  status VARCHAR(16) NOT NULL,
  starts_at DATETIME NOT NULL,
  ends_at DATETIME NOT NULL,
  daily_draw_limit INT NOT NULL DEFAULT 0,
  requires_phone_login TINYINT(1) NOT NULL DEFAULT 0,
  miss_weight INT NOT NULL DEFAULT 0,
  banner_image_url VARCHAR(255) NOT NULL DEFAULT '',
  campaign_summary VARCHAR(255) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_campaigns_slug (slug)
);

CREATE TABLE IF NOT EXISTS prizes (
  id VARCHAR(32) NOT NULL,
  campaign_id VARCHAR(32) NOT NULL,
  name VARCHAR(128) NOT NULL,
  level VARCHAR(16) NOT NULL,
  stock INT NOT NULL DEFAULT 0,
  probability_weight INT NOT NULL DEFAULT 0,
  status VARCHAR(16) NOT NULL DEFAULT 'active',
  image_url VARCHAR(255) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_prizes_campaign_id (campaign_id)
);

CREATE TABLE IF NOT EXISTS user_campaign_quotas (
  id BIGINT NOT NULL AUTO_INCREMENT,
  user_id VARCHAR(32) NOT NULL,
  campaign_id VARCHAR(32) NOT NULL,
  quota_date DATE NOT NULL,
  used_count INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_campaign_quotas (user_id, campaign_id, quota_date)
);

CREATE TABLE IF NOT EXISTS draw_records (
  id VARCHAR(32) NOT NULL,
  campaign_id VARCHAR(32) NOT NULL,
  user_id VARCHAR(32) NOT NULL,
  prize_id VARCHAR(32) NULL,
  prize_name VARCHAR(128) NOT NULL,
  result VARCHAR(16) NOT NULL,
  chance_after INT NOT NULL DEFAULT 0,
  request_id VARCHAR(64) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_draw_records_user_id (user_id),
  KEY idx_draw_records_campaign_id (campaign_id),
  KEY idx_draw_records_created_at (created_at)
);

CREATE TABLE IF NOT EXISTS prize_fulfillment_tasks (
  id BIGINT NOT NULL AUTO_INCREMENT,
  draw_record_id VARCHAR(32) NOT NULL,
  user_id VARCHAR(32) NOT NULL,
  prize_id VARCHAR(32) NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  payload_json JSON NULL,
  operator_note VARCHAR(255) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  fulfilled_at DATETIME NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_prize_fulfillment_draw_record_id (draw_record_id)
);

-- ============================================================
-- 盲盒扩展表
-- ============================================================

-- 用户库存（收集到的盲盒款式）
CREATE TABLE IF NOT EXISTS user_inventories (
  id VARCHAR(32) NOT NULL,
  user_id VARCHAR(32) NOT NULL,
  prize_id VARCHAR(32) NOT NULL,
  prize_name VARCHAR(128) NOT NULL,
  prize_level VARCHAR(16) NOT NULL DEFAULT 'common',
  campaign_id VARCHAR(32) NOT NULL,
  source VARCHAR(32) NOT NULL DEFAULT 'draw',  -- draw / exchange / redeem
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_user_inventories_user_id (user_id),
  KEY idx_user_inventories_campaign_id (campaign_id),
  KEY idx_user_inventories_prize_id (prize_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 交换市场挂单
CREATE TABLE IF NOT EXISTS exchange_offers (
  id VARCHAR(32) NOT NULL,
  user_id VARCHAR(32) NOT NULL,
  user_nickname VARCHAR(64) NOT NULL DEFAULT '',
  have_prize_id VARCHAR(32) NOT NULL,
  have_prize_name VARCHAR(128) NOT NULL,
  want_prize_id VARCHAR(32) NOT NULL,
  want_prize_name VARCHAR(128) NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'pending', -- pending / matched / completed / cancelled
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_exchange_offers_user_id (user_id),
  KEY idx_exchange_offers_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 用户会员/积分信息
CREATE TABLE IF NOT EXISTS user_members (
  user_id VARCHAR(32) NOT NULL,
  level VARCHAR(16) NOT NULL DEFAULT 'normal',  -- normal / silver / gold / diamond
  points BIGINT NOT NULL DEFAULT 0,
  total_draws BIGINT NOT NULL DEFAULT 0,
  total_spent BIGINT NOT NULL DEFAULT 0,  -- 累计消费（分）
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 积分变动记录
CREATE TABLE IF NOT EXISTS user_points_logs (
  id BIGINT NOT NULL AUTO_INCREMENT,
  user_id VARCHAR(32) NOT NULL,
  points BIGINT NOT NULL,              -- 变动数量（正=增加，负=消耗）
  balance BIGINT NOT NULL,             -- 变动后余额
  reason VARCHAR(32) NOT NULL,         -- draw / exchange / daily / redeem / admin
  remark VARCHAR(255) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_user_points_logs_user_id (user_id),
  KEY idx_user_points_logs_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 商店商品配置
CREATE TABLE IF NOT EXISTS shop_items (
  id VARCHAR(32) NOT NULL,
  name VARCHAR(128) NOT NULL,
  description VARCHAR(255) NOT NULL DEFAULT '',
  image_url VARCHAR(255) NOT NULL DEFAULT '',
  price_points INT NOT NULL DEFAULT 0,
  price_cash INT NOT NULL DEFAULT 0,
  item_type VARCHAR(32) NOT NULL,
  item_qty INT NOT NULL DEFAULT 1,
  stock INT NOT NULL DEFAULT -1,
  daily_limit INT NOT NULL DEFAULT 0,
  category VARCHAR(32) NOT NULL DEFAULT '',
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  expires_at DATETIME NULL,
  sort_order INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_shop_items_sort_order (sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 首充礼包配置
CREATE TABLE IF NOT EXISTS first_recharge_packs (
  id VARCHAR(32) NOT NULL,
  name VARCHAR(128) NOT NULL,
  price_points INT NOT NULL DEFAULT 0,
  cash_price INT NOT NULL DEFAULT 0,
  description VARCHAR(255) NOT NULL DEFAULT '',
  image_url VARCHAR(255) NOT NULL DEFAULT '',
  sort_order INT NOT NULL DEFAULT 0,
  items_json JSON NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_first_recharge_packs_sort_order (sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================
-- v1.1 迁移: campaigns 表加 pity_config JSON 字段
-- ============================================================
SET @schema_name := DATABASE();

SET @ddl := IF(
  EXISTS(
    SELECT 1
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @schema_name
      AND TABLE_NAME = 'campaigns'
      AND COLUMN_NAME = 'pity_config'
  ),
  'SELECT 1',
  "ALTER TABLE campaigns ADD COLUMN pity_config JSON NULL COMMENT '保底概率配置' AFTER campaign_summary"
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl := IF(
  EXISTS(
    SELECT 1
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @schema_name
      AND TABLE_NAME = 'prizes'
      AND COLUMN_NAME = 'image_url'
  ),
  'SELECT 1',
  "ALTER TABLE prizes ADD COLUMN image_url VARCHAR(255) NOT NULL DEFAULT '' AFTER status"
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl := IF(
  EXISTS(
    SELECT 1
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @schema_name
      AND TABLE_NAME = 'shop_items'
      AND COLUMN_NAME = 'image_url'
  ),
  'SELECT 1',
  "ALTER TABLE shop_items ADD COLUMN image_url VARCHAR(255) NOT NULL DEFAULT '' AFTER description"
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl := IF(
  EXISTS(
    SELECT 1
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @schema_name
      AND TABLE_NAME = 'first_recharge_packs'
      AND COLUMN_NAME = 'image_url'
  ),
  'SELECT 1',
  "ALTER TABLE first_recharge_packs ADD COLUMN image_url VARCHAR(255) NOT NULL DEFAULT '' AFTER description"
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- user_inventories 表加索引
SET @ddl := IF(
  EXISTS(
    SELECT 1
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = @schema_name
      AND TABLE_NAME = 'user_inventories'
      AND INDEX_NAME = 'idx_ui_user_campaign'
  ),
  'SELECT 1',
  'ALTER TABLE user_inventories ADD INDEX idx_ui_user_campaign (user_id, campaign_id)'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
