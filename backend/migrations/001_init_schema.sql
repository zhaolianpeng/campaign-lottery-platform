-- ============================================================
-- 盲盒抽奖平台数据库 Schema v1.0
-- MySQL 8.0+
-- ============================================================

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id          VARCHAR(48)  NOT NULL PRIMARY KEY,
    nickname    VARCHAR(64)  NOT NULL DEFAULT '',
    avatar_url  VARCHAR(512) NOT NULL DEFAULT '',
    status      VARCHAR(16)  NOT NULL DEFAULT 'active',
    created_at  DATETIME(3)  NOT NULL,
    updated_at  DATETIME(3)  NOT NULL,
    INDEX idx_nickname (nickname),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户会话表
CREATE TABLE IF NOT EXISTS user_sessions (
    token      VARCHAR(64)  NOT NULL PRIMARY KEY,
    user_id    VARCHAR(48)  NOT NULL,
    expires_at DATETIME(3)  NOT NULL,
    created_at DATETIME(3)  NOT NULL,
    INDEX idx_user_id (user_id),
    INDEX idx_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 管理员表
CREATE TABLE IF NOT EXISTS admin_users (
    id            INT          NOT NULL AUTO_INCREMENT PRIMARY KEY,
    username      VARCHAR(64)  NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    display_name  VARCHAR(64)  NOT NULL DEFAULT '',
    status        VARCHAR(16)  NOT NULL DEFAULT 'active',
    created_at    DATETIME(3)  NOT NULL,
    updated_at    DATETIME(3)  NOT NULL,
    INDEX idx_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 盲盒系列/活动表（扩展：保底配置）
CREATE TABLE IF NOT EXISTS campaigns (
    id                VARCHAR(48)    NOT NULL PRIMARY KEY,
    name              VARCHAR(128)   NOT NULL,
    slug              VARCHAR(128)   NOT NULL DEFAULT '',
    status            VARCHAR(16)    NOT NULL DEFAULT 'draft',       -- draft / online / offline / ended
    starts_at         DATETIME(3)    NOT NULL,
    ends_at           DATETIME(3)    NOT NULL,
    daily_draw_limit  INT            NOT NULL DEFAULT 3,
    miss_weight       INT            NOT NULL DEFAULT 80,           -- 未中奖权重
    banner_image_url  VARCHAR(512)   NOT NULL DEFAULT '',
    campaign_summary  TEXT,

    -- 盲盒扩展字段
    series_image_url  VARCHAR(512)   NOT NULL DEFAULT '',
    series_item_count INT            NOT NULL DEFAULT 0,
    draw_price        INT            NOT NULL DEFAULT 0,            -- 单抽价格（分）
    ten_draw_price    INT            NOT NULL DEFAULT 0,            -- 十连价格（分）
    pity_enabled      TINYINT(1)     NOT NULL DEFAULT 0,
    soft_pity_n       INT            NOT NULL DEFAULT 60,           -- 软保底开始次数
    pity_factor       DECIMAL(8,4)   NOT NULL DEFAULT 0.0150,      -- 概率递增因子
    hard_pity_n       INT            NOT NULL DEFAULT 90,           -- 硬保底次数
    target_prize_id   VARCHAR(48)    DEFAULT NULL,                  -- 保底目标奖品ID
    max_draw_per_day  INT            NOT NULL DEFAULT 3,            -- 每日抽奖上限

    created_at        DATETIME(3)    NOT NULL,
    updated_at        DATETIME(3)    NOT NULL,
    INDEX idx_status (status),
    INDEX idx_slug (slug),
    INDEX idx_date_range (starts_at, ends_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 盲盒礼品表
CREATE TABLE IF NOT EXISTS prizes (
    id                 VARCHAR(48)  NOT NULL PRIMARY KEY,
    campaign_id        VARCHAR(48)  NOT NULL,
    name               VARCHAR(128) NOT NULL,
    level              VARCHAR(16)  NOT NULL DEFAULT 'common',      -- common / rare / secret / limited
    stock              INT          NOT NULL DEFAULT 0,
    probability_weight INT          NOT NULL DEFAULT 0,
    status             VARCHAR(16)  NOT NULL DEFAULT 'active',

    -- 盲盒扩展字段
    image_url          VARCHAR(512) NOT NULL DEFAULT '',
    sort_order         INT          NOT NULL DEFAULT 0,
    display_prob       VARCHAR(16)  DEFAULT NULL,                   -- 对外公示概率字符串，如 "7.00%"

    created_at         DATETIME(3)  NOT NULL,
    updated_at         DATETIME(3)  NOT NULL,
    INDEX idx_campaign (campaign_id),
    INDEX idx_level (level),
    INDEX idx_sort (campaign_id, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 抽奖记录表
CREATE TABLE IF NOT EXISTS draw_records (
    id           VARCHAR(48)  NOT NULL PRIMARY KEY,
    campaign_id  VARCHAR(48)  NOT NULL,
    user_id      VARCHAR(48)  NOT NULL,
    prize_id     VARCHAR(48)  DEFAULT NULL,
    prize_name   VARCHAR(128) NOT NULL DEFAULT '',
    result       VARCHAR(8)   NOT NULL DEFAULT 'miss',              -- win / miss
    chance_after INT          NOT NULL DEFAULT 0,
    request_id   VARCHAR(64)  NOT NULL DEFAULT '',
    is_ten_pull  TINYINT(1)   NOT NULL DEFAULT 0,
    pity_info    JSON         DEFAULT NULL,                         -- {consecutive_misses, pity_multiplier, is_hard_pity}
    created_at   DATETIME(3)  NOT NULL,
    INDEX idx_user (user_id, created_at DESC),
    INDEX idx_campaign (campaign_id, created_at DESC),
    INDEX idx_result (result)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户每日抽奖额度表
CREATE TABLE IF NOT EXISTS user_campaign_quotas (
    id          BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id     VARCHAR(48)  NOT NULL,
    campaign_id VARCHAR(48)  NOT NULL,
    quota_date  DATE         NOT NULL,
    used_count  INT          NOT NULL DEFAULT 0,
    created_at  DATETIME(3)  NOT NULL,
    updated_at  DATETIME(3)  NOT NULL,
    UNIQUE KEY uk_user_campaign_date (user_id, campaign_id, quota_date),
    INDEX idx_user (user_id),
    INDEX idx_campaign_date (campaign_id, quota_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户库存表
CREATE TABLE IF NOT EXISTS user_inventories (
    id           VARCHAR(48) NOT NULL PRIMARY KEY,
    user_id      VARCHAR(48) NOT NULL,
    prize_id     VARCHAR(48) NOT NULL,
    prize_name   VARCHAR(128) NOT NULL DEFAULT '',
    prize_level  VARCHAR(16) NOT NULL DEFAULT 'common',
    campaign_id  VARCHAR(48) NOT NULL,
    source       VARCHAR(16) NOT NULL DEFAULT 'draw',               -- draw / exchange / redeem
    source_id    VARCHAR(48) DEFAULT NULL,                          -- 来源记录ID
    created_at   DATETIME(3) NOT NULL,
    INDEX idx_user (user_id),
    INDEX idx_user_prize (user_id, prize_id),
    INDEX idx_campaign (campaign_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 奖品发货任务表
CREATE TABLE IF NOT EXISTS prize_fulfillment_tasks (
    id            BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    draw_record_id VARCHAR(48) NOT NULL,
    user_id       VARCHAR(48)  NOT NULL,
    prize_id      VARCHAR(48)  NOT NULL,
    status        VARCHAR(16)  NOT NULL DEFAULT 'pending',          -- pending / processing / fulfilled / rejected
    payload_json  JSON,
    operator_note TEXT,
    created_at    DATETIME(3)  NOT NULL,
    updated_at    DATETIME(3)  NOT NULL,
    fulfilled_at  DATETIME(3)  DEFAULT NULL,
    INDEX idx_user (user_id),
    INDEX idx_status (status),
    INDEX idx_draw_record (draw_record_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 交换市场挂单表
CREATE TABLE IF NOT EXISTS exchange_offers (
    id              VARCHAR(48) NOT NULL PRIMARY KEY,
    user_id         VARCHAR(48) NOT NULL,
    have_prize_id   VARCHAR(48) NOT NULL,
    want_prize_id   VARCHAR(48) NOT NULL,
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',          -- pending / matched / completed / cancelled
    matched_user_id VARCHAR(48) DEFAULT NULL,
    created_at      DATETIME(3) NOT NULL,
    updated_at      DATETIME(3) NOT NULL,
    INDEX idx_user (user_id),
    INDEX idx_status (status),
    INDEX idx_have_prize (have_prize_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户会员表
CREATE TABLE IF NOT EXISTS user_members (
    user_id      VARCHAR(48) NOT NULL PRIMARY KEY,
    level        VARCHAR(16) NOT NULL DEFAULT 'normal',              -- normal / silver / gold / diamond
    points       BIGINT      NOT NULL DEFAULT 0,
    total_draws  BIGINT      NOT NULL DEFAULT 0,
    total_spent  BIGINT      NOT NULL DEFAULT 0,                    -- 累计消费（分）
    created_at   DATETIME(3) NOT NULL,
    updated_at   DATETIME(3) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 积分变动日志表
CREATE TABLE IF NOT EXISTS user_points_logs (
    id          BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id     VARCHAR(48)  NOT NULL,
    points      BIGINT       NOT NULL,                             -- 变动数量
    balance     BIGINT       NOT NULL,                             -- 变动后余额
    reason      VARCHAR(32)  NOT NULL,                             -- draw / exchange / daily / redeem
    remark      VARCHAR(255) DEFAULT NULL,
    created_at  DATETIME(3)  NOT NULL,
    INDEX idx_user (user_id, created_at DESC),
    INDEX idx_reason (reason)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 保底状态表（可选，用于持久化保底计数）
CREATE TABLE IF NOT EXISTS pity_states (
    id                 BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id            VARCHAR(48)  NOT NULL,
    campaign_id        VARCHAR(48)  NOT NULL,
    consecutive_misses INT          NOT NULL DEFAULT 0,
    created_at         DATETIME(3)  NOT NULL,
    updated_at         DATETIME(3)  NOT NULL,
    UNIQUE KEY uk_user_campaign (user_id, campaign_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
