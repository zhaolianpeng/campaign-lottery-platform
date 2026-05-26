CREATE TABLE IF NOT EXISTS pending_anonymous_wins (
  id VARCHAR(32) NOT NULL,
  anonymous_token VARCHAR(64) NOT NULL,
  campaign_id VARCHAR(64) NOT NULL,
  prize_id VARCHAR(64) NOT NULL,
  prize_name VARCHAR(255) NOT NULL,
  prize_level VARCHAR(32) NOT NULL,
  prize_image_url VARCHAR(512) NULL,
  drawn_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_pending_anonymous_wins_token (anonymous_token),
  KEY idx_pending_anonymous_wins_drawn_at (drawn_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;