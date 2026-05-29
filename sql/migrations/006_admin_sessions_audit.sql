-- Admin sessions and audit logs
CREATE TABLE IF NOT EXISTS admin_sessions (
  token VARCHAR(64) NOT NULL,
  admin_user_id BIGINT NOT NULL,
  expires_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (token),
  KEY idx_admin_sessions_admin_user_id (admin_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS admin_audit_logs (
  id BIGINT NOT NULL AUTO_INCREMENT,
  admin_user_id BIGINT NULL,
  admin_username VARCHAR(64) NOT NULL DEFAULT '',
  action VARCHAR(64) NOT NULL,
  resource_type VARCHAR(32) NOT NULL DEFAULT '',
  resource_id VARCHAR(64) NOT NULL DEFAULT '',
  request_id VARCHAR(64) NOT NULL DEFAULT '',
  payload_json JSON NULL,
  ip_address VARCHAR(64) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_admin_audit_logs_created_at (created_at),
  KEY idx_admin_audit_logs_action (action)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
