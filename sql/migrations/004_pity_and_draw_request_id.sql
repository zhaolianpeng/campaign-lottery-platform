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

ALTER TABLE draw_records
  ADD UNIQUE KEY uk_draw_records_request_id (request_id);
