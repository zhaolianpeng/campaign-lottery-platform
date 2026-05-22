CREATE DATABASE IF NOT EXISTS campaign_lottery_platform
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_unicode_ci;

USE campaign_lottery_platform;

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
