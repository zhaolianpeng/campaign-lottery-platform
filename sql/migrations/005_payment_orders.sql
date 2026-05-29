-- Payment orders persisted in MySQL (authoritative for production)
CREATE TABLE IF NOT EXISTS payment_orders (
  order_no VARCHAR(64) NOT NULL,
  user_id VARCHAR(32) NOT NULL,
  client_request_id VARCHAR(64) NOT NULL,
  channel VARCHAR(16) NOT NULL,
  presentation VARCHAR(32) NOT NULL DEFAULT '',
  subject VARCHAR(128) NOT NULL,
  body VARCHAR(255) NOT NULL DEFAULT '',
  business_type VARCHAR(32) NOT NULL,
  business_id VARCHAR(64) NOT NULL DEFAULT '',
  product_snapshot JSON NULL,
  amount_cents INT NOT NULL,
  currency VARCHAR(8) NOT NULL DEFAULT 'CNY',
  status VARCHAR(16) NOT NULL DEFAULT 'created',
  channel_trade_no VARCHAR(64) NOT NULL DEFAULT '',
  paid_at DATETIME NULL,
  fulfilled_at DATETIME NULL,
  expire_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (order_no),
  UNIQUE KEY uk_payment_orders_user_request (user_id, client_request_id),
  KEY idx_payment_orders_user_id (user_id),
  KEY idx_payment_orders_status (status),
  KEY idx_payment_orders_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS payment_notify_logs (
  id BIGINT NOT NULL AUTO_INCREMENT,
  order_no VARCHAR(64) NOT NULL,
  channel VARCHAR(16) NOT NULL,
  notify_id VARCHAR(128) NOT NULL DEFAULT '',
  raw_body MEDIUMTEXT NOT NULL,
  verified TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_payment_notify_idempotent (channel, notify_id),
  KEY idx_payment_notify_order_no (order_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
