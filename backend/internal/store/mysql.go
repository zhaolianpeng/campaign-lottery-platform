package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"campaign-lottery-platform/backend/internal/config"
	"campaign-lottery-platform/backend/internal/model"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type MySQLStore struct {
	db            *sql.DB
	redis         *redis.Client
	redisPrefix   string
	adminUser     string
	adminPassword string
}

func NewMySQLStore(cfg config.Config) (*MySQLStore, error) {
	db, err := sql.Open(cfg.DBDriver, cfg.DBDSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &MySQLStore{
		db:            db,
		redis:         redisClient,
		redisPrefix:   cfg.RedisPrefix,
		adminUser:     cfg.AdminUser,
		adminPassword: cfg.AdminPassword,
	}, nil
}

func (store *MySQLStore) Seed() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var count int
	if err := store.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM admin_users").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte(store.adminPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		_, err = store.db.ExecContext(ctx, "INSERT INTO admin_users (username, password_hash, display_name, status, created_at, updated_at) VALUES (?, ?, ?, 'active', UTC_TIMESTAMP(), UTC_TIMESTAMP())", store.adminUser, string(hash), "系统管理员")
		if err != nil {
			return err
		}
	}

	if err := store.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM campaigns").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	now := time.Now().UTC()
	campaignID := "camp_launch_001"
	_, err := store.db.ExecContext(ctx, `INSERT INTO campaigns (id, name, slug, status, starts_at, ends_at, daily_draw_limit, miss_weight, banner_image_url, campaign_summary, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())`,
		campaignID, "夏季开门红抽奖活动", "summer-launch", "online", now.Add(-24*time.Hour), now.Add(30*24*time.Hour), 3, 86, "https://static.example.com/campaign/summer-launch/banner.png", "新用户登录即可参与，中奖后进入发奖队列，支持后台配置库存和概率。")
	if err != nil {
		return err
	}

	seedPrizes := []model.Prize{
		{ID: "prize_001", CampaignID: campaignID, Name: "88元红包", Level: "S", Stock: 8, ProbabilityWeight: 2, Status: "active"},
		{ID: "prize_002", CampaignID: campaignID, Name: "20元优惠券", Level: "A", Stock: 60, ProbabilityWeight: 18, Status: "active"},
		{ID: "prize_003", CampaignID: campaignID, Name: "品牌周边礼盒", Level: "B", Stock: 20, ProbabilityWeight: 8, Status: "active"},
	}
	for _, prize := range seedPrizes {
		_, err = store.db.ExecContext(ctx, `INSERT INTO prizes (id, campaign_id, name, level, stock, probability_weight, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())`, prize.ID, prize.CampaignID, prize.Name, prize.Level, prize.Stock, prize.ProbabilityWeight, prize.Status)
		if err != nil {
			return err
		}
	}

	// 盲盒专用种子数据
	blindboxID := "camp_blindbox_001"
	bbPityConfig := model.PityConfig{
		Enabled:     true,
		SoftPityN:   30,
		PityFactor:  0.015,
		HardPityN:   60,
		TargetPrize: "prize_bb_secret",
		UPPoolEnabled: true,
		UPPrizeID:     "prize_bb_secret",
		UPMultiplier:  5,
		UPLevel:       "secret",
		UPStartAt:     now.Add(-24 * time.Hour),
		UPEndAt:       now.Add(14 * 24 * time.Hour),
	}
	bbPityBytes, _ := json.Marshal(bbPityConfig)
	_, err = store.db.ExecContext(ctx, `INSERT INTO campaigns (id, name, slug, status, starts_at, ends_at, daily_draw_limit, miss_weight, banner_image_url, campaign_summary, pity_config, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())`,
		blindboxID, "梦幻星辰系列盲盒", "dream-star-series", "online",
		now.Add(-24*time.Hour), now.Add(60*24*time.Hour),
		10, 30,
		"https://static.example.com/blindbox/dream-star/banner.png",
		"收集12款星辰主题公仔，集齐全套可兑换隐藏款！每抽必出，软保底30抽递增，60抽硬保底。🌟 限时UP：雅典娜·黄金圣衣 EX 概率5倍提升！",
		string(bbPityBytes))
	if err != nil {
		return err
	}

	bbPrizes := []model.Prize{
		{ID: "prize_bb_01", CampaignID: blindboxID, Name: "射手座·星矢", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_02", CampaignID: blindboxID, Name: "白羊座·穆", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_03", CampaignID: blindboxID, Name: "金牛座·阿鲁", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_04", CampaignID: blindboxID, Name: "双子座·撒加", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_05", CampaignID: blindboxID, Name: "巨蟹座·迪斯", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_06", CampaignID: blindboxID, Name: "狮子座·艾欧", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_07", CampaignID: blindboxID, Name: "处女座·沙加", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_08", CampaignID: blindboxID, Name: "天秤座·童虎", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_09", CampaignID: blindboxID, Name: "天蝎座·米罗", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_10", CampaignID: blindboxID, Name: "射手座·艾俄", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_11", CampaignID: blindboxID, Name: "摩羯座·修罗", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_12", CampaignID: blindboxID, Name: "水瓶座·卡妙", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_rare", CampaignID: blindboxID, Name: "双鱼座·阿布罗狄 (闪光版)", Level: "rare", Stock: 300, ProbabilityWeight: 5, Status: "active"},
		{ID: "prize_bb_secret", CampaignID: blindboxID, Name: "🌟 雅典娜·黄金圣衣 EX", Level: "secret", Stock: 20, ProbabilityWeight: 1, Status: "active"},
	}
	for _, prize := range bbPrizes {
		_, err = store.db.ExecContext(ctx, `INSERT INTO prizes (id, campaign_id, name, level, stock, probability_weight, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())`, prize.ID, prize.CampaignID, prize.Name, prize.Level, prize.Stock, prize.ProbabilityWeight, prize.Status)
		if err != nil {
			return err
		}
	}

	return nil
}

func (store *MySQLStore) CreateGuestSession(nickname string) (model.User, model.Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if strings.TrimSpace(nickname) == "" {
		nickname = "Guest" + randomSuffix(4)
	}
	user := model.User{ID: "usr_" + randomSuffix(12), Nickname: nickname, CreatedAt: time.Now().UTC()}
	session := model.Session{Token: "utk_" + randomSuffix(24), UserID: user.ID, ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour)}

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return model.User{}, model.Session{}, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "INSERT INTO users (id, nickname, created_at, updated_at) VALUES (?, ?, ?, ?)", user.ID, user.Nickname, user.CreatedAt, user.CreatedAt)
	if err != nil {
		return model.User{}, model.Session{}, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO user_sessions (token, user_id, expires_at, created_at) VALUES (?, ?, ?, UTC_TIMESTAMP())", session.Token, session.UserID, session.ExpiresAt)
	if err != nil {
		return model.User{}, model.Session{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.User{}, model.Session{}, err
	}
	if err := store.cacheUserSession(ctx, session); err != nil {
		return model.User{}, model.Session{}, err
	}
	return user, session, nil
}

func (store *MySQLStore) Campaigns() []model.Campaign {
	items, err := store.fetchCampaigns(context.Background(), false)
	if err != nil {
		return []model.Campaign{}
	}
	return items
}

func (store *MySQLStore) PrizeList(campaignID string) []model.Prize {
	items, err := store.fetchPrizes(context.Background(), campaignID)
	if err != nil {
		return []model.Prize{}
	}
	return items
}

func (store *MySQLStore) UserFromToken(token string) (model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := store.readUserSession(ctx, token)
	if err != nil {
		return model.User{}, err
	}

	var user model.User
	err = store.db.QueryRowContext(ctx, "SELECT id, nickname, created_at FROM users WHERE id = ?", session.UserID).Scan(&user.ID, &user.Nickname, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, ErrUnauthorized
	}
	if err != nil {
		return model.User{}, err
	}
	return user, nil
}

func (store *MySQLStore) Draw(token string, campaignID string) (model.DrawResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	session, err := store.readUserSession(ctx, token)
	if err != nil {
		return model.DrawResult{}, err
	}

	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return model.DrawResult{}, err
	}
	defer tx.Rollback()

	var campaign model.Campaign
	err = tx.QueryRowContext(ctx, `SELECT id, name, slug, status, starts_at, ends_at, daily_draw_limit, miss_weight, banner_image_url, campaign_summary
		FROM campaigns WHERE id = ? FOR UPDATE`, campaignID).
		Scan(&campaign.ID, &campaign.Name, &campaign.Slug, &campaign.Status, &campaign.StartsAt, &campaign.EndsAt, &campaign.DailyDrawLimit, &campaign.MissWeight, &campaign.BannerImageURL, &campaign.CampaignSummary)
	if errors.Is(err, sql.ErrNoRows) {
		return model.DrawResult{}, ErrCampaignNotFound
	}
	if err != nil {
		return model.DrawResult{}, err
	}

	now := time.Now().UTC()
	if now.Before(campaign.StartsAt) || now.After(campaign.EndsAt) || campaign.Status != "online" {
		return model.DrawResult{}, ErrCampaignInactive
	}

	quotaDate := now.Format("2006-01-02")
	_, err = tx.ExecContext(ctx, `INSERT INTO user_campaign_quotas (user_id, campaign_id, quota_date, used_count, created_at, updated_at)
		VALUES (?, ?, ?, 0, UTC_TIMESTAMP(), UTC_TIMESTAMP())
		ON DUPLICATE KEY UPDATE updated_at = VALUES(updated_at)`, session.UserID, campaignID, quotaDate)
	if err != nil {
		return model.DrawResult{}, err
	}

	var usedCount int
	err = tx.QueryRowContext(ctx, `SELECT used_count FROM user_campaign_quotas WHERE user_id = ? AND campaign_id = ? AND quota_date = ? FOR UPDATE`, session.UserID, campaignID, quotaDate).Scan(&usedCount)
	if err != nil {
		return model.DrawResult{}, err
	}
	if campaign.DailyDrawLimit > 0 && usedCount >= campaign.DailyDrawLimit {
		return model.DrawResult{}, ErrNoDrawChances
	}

	rows, err := tx.QueryContext(ctx, `SELECT id, campaign_id, name, level, stock, probability_weight, status FROM prizes WHERE campaign_id = ? ORDER BY level ASC, id ASC FOR UPDATE`, campaignID)
	if err != nil {
		return model.DrawResult{}, err
	}
	defer rows.Close()
	prizes := make([]model.Prize, 0, 8)
	for rows.Next() {
		var prize model.Prize
		if err := rows.Scan(&prize.ID, &prize.CampaignID, &prize.Name, &prize.Level, &prize.Stock, &prize.ProbabilityWeight, &prize.Status); err != nil {
			return model.DrawResult{}, err
		}
		prizes = append(prizes, prize)
	}

	totalWeight := campaign.MissWeight
	for _, prize := range prizes {
		if prize.Status == "active" && prize.Stock > 0 && prize.ProbabilityWeight > 0 {
			totalWeight += prize.ProbabilityWeight
		}
	}
	roll := rand.IntN(totalWeight)
	acc := 0
	result := "miss"
	prizeName := "未中奖"
	var prizeID *string
	for _, prize := range prizes {
		if prize.Status != "active" || prize.Stock <= 0 || prize.ProbabilityWeight <= 0 {
			continue
		}
		acc += prize.ProbabilityWeight
		if roll < acc {
			_, err = tx.ExecContext(ctx, `UPDATE prizes SET stock = stock - 1, updated_at = UTC_TIMESTAMP() WHERE id = ? AND stock > 0`, prize.ID)
			if err != nil {
				return model.DrawResult{}, err
			}
			pickedID := prize.ID
			prizeID = &pickedID
			prizeName = prize.Name
			result = "win"
			break
		}
	}

	newUsed := usedCount + 1
	_, err = tx.ExecContext(ctx, `UPDATE user_campaign_quotas SET used_count = ?, updated_at = UTC_TIMESTAMP() WHERE user_id = ? AND campaign_id = ? AND quota_date = ?`, newUsed, session.UserID, campaignID, quotaDate)
	if err != nil {
		return model.DrawResult{}, err
	}

	record := model.DrawRecord{ID: "draw_" + randomSuffix(12), CampaignID: campaignID, UserID: session.UserID, PrizeID: prizeID, PrizeName: prizeName, Result: result, DrawnAt: now, ChanceAfter: campaign.DailyDrawLimit - newUsed}
	_, err = tx.ExecContext(ctx, `INSERT INTO draw_records (id, campaign_id, user_id, prize_id, prize_name, result, chance_after, request_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, record.ID, record.CampaignID, record.UserID, record.PrizeID, record.PrizeName, record.Result, record.ChanceAfter, "", record.DrawnAt)
	if err != nil {
		return model.DrawResult{}, err
	}

	if prizeID != nil {
		payload, _ := json.Marshal(map[string]any{"source": "lottery_draw", "draw_record_id": record.ID, "prize_id": *prizeID})
		_, err = tx.ExecContext(ctx, `INSERT INTO prize_fulfillment_tasks (draw_record_id, user_id, prize_id, status, payload_json, operator_note, created_at, updated_at)
			VALUES (?, ?, ?, 'pending', ?, '', UTC_TIMESTAMP(), UTC_TIMESTAMP())`, record.ID, record.UserID, *prizeID, string(payload))
		if err != nil {
			return model.DrawResult{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return model.DrawResult{}, err
	}
	store.redis.Del(ctx, store.campaignCacheKey())
	return model.DrawResult{Record: record, RemainingChances: record.ChanceAfter}, nil
}

func (store *MySQLStore) UserDrawRecords(token string) ([]model.DrawRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := store.readUserSession(ctx, token)
	if err != nil {
		return nil, err
	}
	return store.fetchDrawRecordsByUser(ctx, session.UserID)
}
func (store *MySQLStore) AdminLogin(username, password string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var passwordHash string
	err := store.db.QueryRowContext(ctx, `SELECT password_hash FROM admin_users WHERE username = ? AND status = 'active'`, username).Scan(&passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrBadAdminAuth
	}
	if err != nil {
		return "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return "", ErrBadAdminAuth
	}
	token := "atk_" + randomSuffix(24)

	// Also persist to MySQL so admin tokens survive Redis restart
	now := time.Now().UTC()
	expiresAt := now.Add(12 * time.Hour)
	_, mySQLErr := store.db.ExecContext(ctx,
		`INSERT INTO user_sessions (token, user_id, expires_at, created_at)
		 VALUES (?, 'admin', ?, UTC_TIMESTAMP())
		 ON DUPLICATE KEY UPDATE expires_at = VALUES(expires_at)`,
		token, expiresAt)
	if mySQLErr != nil {
		// Log but don't fail - Redis is the primary store
		fmt.Printf("WARN: failed to persist admin session to MySQL: %v\n", mySQLErr)
	}

	if err := store.redis.Set(ctx, store.adminSessionKey(token), username, 12*time.Hour).Err(); err != nil {
		return "", err
	}
	return token, nil
}

func (store *MySQLStore) AdminOverview(token string) (model.AdminOverview, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.ensureAdmin(ctx, token); err != nil {
		return model.AdminOverview{}, err
	}
	var totalUsers, totalDraws, totalWins int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&totalUsers); err != nil {
		return model.AdminOverview{}, err
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM draw_records`).Scan(&totalDraws); err != nil {
		return model.AdminOverview{}, err
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM draw_records WHERE result = 'win'`).Scan(&totalWins); err != nil {
		return model.AdminOverview{}, err
	}
	campaigns, err := store.fetchCampaigns(ctx, true)
	if err != nil {
		return model.AdminOverview{}, err
	}
	prizeSummaries, err := store.fetchPrizeSummaries(ctx)
	if err != nil {
		return model.AdminOverview{}, err
	}
	recentDraws, err := store.fetchRecentDrawRecords(ctx, 10)
	if err != nil {
		return model.AdminOverview{}, err
	}
	balance, err := store.fetchQuotaBalance(ctx)
	if err != nil {
		return model.AdminOverview{}, err
	}
	return model.AdminOverview{TotalUsers: totalUsers, TotalDraws: totalDraws, TotalWins: totalWins, Campaigns: campaigns, PrizeSummaries: prizeSummaries, RecentDraws: recentDraws, UserDrawBalance: balance}, nil
}

func (store *MySQLStore) AdminDrawRecords(token string) ([]model.DrawRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.ensureAdmin(ctx, token); err != nil {
		return nil, err
	}
	return store.fetchRecentDrawRecords(ctx, 200)
}

func (store *MySQLStore) AdminCampaigns(token string) ([]model.Campaign, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.ensureAdmin(ctx, token); err != nil {
		return nil, err
	}
	return store.fetchCampaigns(ctx, true)
}

func (store *MySQLStore) CreateCampaign(token string, input model.CampaignMutation) (model.Campaign, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.ensureAdmin(ctx, token); err != nil {
		return model.Campaign{}, err
	}
	campaign := model.Campaign{ID: "camp_" + randomSuffix(10), Name: input.Name, Slug: input.Slug, Status: input.Status, StartsAt: input.StartsAt, EndsAt: input.EndsAt, DailyDrawLimit: input.DailyDrawLimit, MissWeight: input.MissWeight, BannerImageURL: input.BannerImageURL, CampaignSummary: input.CampaignSummary, PityConfig: input.PityConfig}
	pityConfigBytes, _ := json.Marshal(input.PityConfig)
	pityConfigStr := string(pityConfigBytes)
	_, err := store.db.ExecContext(ctx, `INSERT INTO campaigns (id, name, slug, status, starts_at, ends_at, daily_draw_limit, miss_weight, banner_image_url, campaign_summary, pity_config, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())`, campaign.ID, campaign.Name, campaign.Slug, campaign.Status, campaign.StartsAt, campaign.EndsAt, campaign.DailyDrawLimit, campaign.MissWeight, campaign.BannerImageURL, campaign.CampaignSummary, pityConfigStr)
	if err != nil {
		return model.Campaign{}, err
	}
	store.redis.Del(ctx, store.campaignCacheKey())
	return campaign, nil
}

func (store *MySQLStore) UpdateCampaign(token string, campaignID string, input model.CampaignMutation) (model.Campaign, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.ensureAdmin(ctx, token); err != nil {
		return model.Campaign{}, err
	}
	pityConfigBytes, _ := json.Marshal(input.PityConfig)
	pityConfigStr := string(pityConfigBytes)
	result, err := store.db.ExecContext(ctx, `UPDATE campaigns SET name = ?, slug = ?, status = ?, starts_at = ?, ends_at = ?, daily_draw_limit = ?, miss_weight = ?, banner_image_url = ?, campaign_summary = ?, pity_config = ?, updated_at = UTC_TIMESTAMP() WHERE id = ?`,
		input.Name, input.Slug, input.Status, input.StartsAt, input.EndsAt, input.DailyDrawLimit, input.MissWeight, input.BannerImageURL, input.CampaignSummary, pityConfigStr, campaignID)
	if err != nil {
		return model.Campaign{}, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return model.Campaign{}, ErrCampaignNotFound
	}
	store.redis.Del(ctx, store.campaignCacheKey())
	return model.Campaign{ID: campaignID, Name: input.Name, Slug: input.Slug, Status: input.Status, StartsAt: input.StartsAt, EndsAt: input.EndsAt, DailyDrawLimit: input.DailyDrawLimit, MissWeight: input.MissWeight, BannerImageURL: input.BannerImageURL, CampaignSummary: input.CampaignSummary}, nil
}

func (store *MySQLStore) DeleteCampaign(token string, campaignID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.ensureAdmin(ctx, token); err != nil {
		return err
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM prizes WHERE campaign_id = ?`, campaignID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM campaigns WHERE id = ?`, campaignID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	store.redis.Del(ctx, store.campaignCacheKey())
	return nil
}

func (store *MySQLStore) AdminPrizes(token string, campaignID string) ([]model.Prize, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.ensureAdmin(ctx, token); err != nil {
		return nil, err
	}
	return store.fetchPrizes(ctx, campaignID)
}

func (store *MySQLStore) CreatePrize(token string, campaignID string, input model.PrizeMutation) (model.Prize, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.ensureAdmin(ctx, token); err != nil {
		return model.Prize{}, err
	}
	prize := model.Prize{ID: "prize_" + randomSuffix(10), CampaignID: campaignID, Name: input.Name, Level: input.Level, Stock: input.Stock, ProbabilityWeight: input.ProbabilityWeight, Status: input.Status}
	_, err := store.db.ExecContext(ctx, `INSERT INTO prizes (id, campaign_id, name, level, stock, probability_weight, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())`, prize.ID, prize.CampaignID, prize.Name, prize.Level, prize.Stock, prize.ProbabilityWeight, prize.Status)
	if err != nil {
		return model.Prize{}, err
	}
	store.redis.Del(ctx, store.campaignCacheKey())
	return prize, nil
}

func (store *MySQLStore) UpdatePrize(token string, prizeID string, input model.PrizeMutation) (model.Prize, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.ensureAdmin(ctx, token); err != nil {
		return model.Prize{}, err
	}
	var campaignID string
	err := store.db.QueryRowContext(ctx, `SELECT campaign_id FROM prizes WHERE id = ?`, prizeID).Scan(&campaignID)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Prize{}, ErrCampaignNotFound
	}
	if err != nil {
		return model.Prize{}, err
	}
	_, err = store.db.ExecContext(ctx, `UPDATE prizes SET name = ?, level = ?, stock = ?, probability_weight = ?, status = ?, updated_at = UTC_TIMESTAMP() WHERE id = ?`, input.Name, input.Level, input.Stock, input.ProbabilityWeight, input.Status, prizeID)
	if err != nil {
		return model.Prize{}, err
	}
	store.redis.Del(ctx, store.campaignCacheKey())
	return model.Prize{ID: prizeID, CampaignID: campaignID, Name: input.Name, Level: input.Level, Stock: input.Stock, ProbabilityWeight: input.ProbabilityWeight, Status: input.Status}, nil
}

func (store *MySQLStore) DeletePrize(token string, prizeID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.ensureAdmin(ctx, token); err != nil {
		return err
	}
	_, err := store.db.ExecContext(ctx, `DELETE FROM prizes WHERE id = ?`, prizeID)
	if err != nil {
		return err
	}
	store.redis.Del(ctx, store.campaignCacheKey())
	return nil
}

func (store *MySQLStore) FulfillmentTasks(token string) ([]model.FulfillmentTask, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.ensureAdmin(ctx, token); err != nil {
		return nil, err
	}
	rows, err := store.db.QueryContext(ctx, `SELECT id, draw_record_id, user_id, prize_id, status, COALESCE(CAST(payload_json AS CHAR), ''), operator_note, created_at, updated_at, fulfilled_at FROM prize_fulfillment_tasks ORDER BY id DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.FulfillmentTask, 0, 16)
	for rows.Next() {
		var item model.FulfillmentTask
		if err := rows.Scan(&item.ID, &item.DrawRecordID, &item.UserID, &item.PrizeID, &item.Status, &item.PayloadJSON, &item.OperatorNote, &item.CreatedAt, &item.UpdatedAt, &item.FulfilledAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (store *MySQLStore) UpdateFulfillmentTask(token string, taskID int64, input model.FulfillmentTaskMutation) (model.FulfillmentTask, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.ensureAdmin(ctx, token); err != nil {
		return model.FulfillmentTask{}, err
	}
	fulfilledAt := sql.NullTime{}
	if input.Status == "fulfilled" {
		fulfilledAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
	}
	_, err := store.db.ExecContext(ctx, `UPDATE prize_fulfillment_tasks SET status = ?, operator_note = ?, fulfilled_at = ?, updated_at = UTC_TIMESTAMP() WHERE id = ?`, input.Status, input.OperatorNote, fulfilledAt, taskID)
	if err != nil {
		return model.FulfillmentTask{}, err
	}
	rows, err := store.FulfillmentTasks(token)
	if err != nil {
		return model.FulfillmentTask{}, err
	}
	for _, item := range rows {
		if item.ID == taskID {
			return item, nil
		}
	}
	return model.FulfillmentTask{}, fmt.Errorf("fulfillment task not found")
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

func (store *MySQLStore) fetchCampaigns(ctx context.Context, includeOffline bool) ([]model.Campaign, error) {
	if !includeOffline {
		if cached, err := store.redis.Get(ctx, store.campaignCacheKey()).Result(); err == nil && cached != "" {
			var items []model.Campaign
			if json.Unmarshal([]byte(cached), &items) == nil {
				return items, nil
			}
		}
	}

	query := `SELECT id, name, slug, status, starts_at, ends_at, daily_draw_limit, miss_weight, banner_image_url, campaign_summary, pity_config FROM campaigns`
	if !includeOffline {
		query += ` WHERE status = 'online' ORDER BY starts_at DESC`
	} else {
		query += ` ORDER BY created_at DESC`
	}
	rows, err := store.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.Campaign, 0, 8)
	for rows.Next() {
		var item model.Campaign
		var pityConfigJSON string
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Status, &item.StartsAt, &item.EndsAt, &item.DailyDrawLimit, &item.MissWeight, &item.BannerImageURL, &item.CampaignSummary, &pityConfigJSON); err != nil {
			return nil, err
		}
		if pityConfigJSON != "" {
			json.Unmarshal([]byte(pityConfigJSON), &item.PityConfig)
		}
		items = append(items, item)
	}
	if !includeOffline {
		if payload, err := json.Marshal(items); err == nil {
			store.redis.Set(ctx, store.campaignCacheKey(), payload, 5*time.Minute)
		}
	}
	return items, nil
}

func (store *MySQLStore) fetchPrizes(ctx context.Context, campaignID string) ([]model.Prize, error) {
	rows, err := store.db.QueryContext(ctx, `SELECT id, campaign_id, name, level, stock, probability_weight, status FROM prizes WHERE campaign_id = ? ORDER BY level ASC, id ASC`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.Prize, 0, 8)
	for rows.Next() {
		var item model.Prize
		if err := rows.Scan(&item.ID, &item.CampaignID, &item.Name, &item.Level, &item.Stock, &item.ProbabilityWeight, &item.Status); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (store *MySQLStore) fetchDrawRecordsByUser(ctx context.Context, userID string) ([]model.DrawRecord, error) {
	rows, err := store.db.QueryContext(ctx, `SELECT id, campaign_id, user_id, prize_id, prize_name, result, created_at, chance_after FROM draw_records WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDrawRecords(rows)
}

func (store *MySQLStore) fetchRecentDrawRecords(ctx context.Context, limit int) ([]model.DrawRecord, error) {
	rows, err := store.db.QueryContext(ctx, `SELECT id, campaign_id, user_id, prize_id, prize_name, result, created_at, chance_after FROM draw_records ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDrawRecords(rows)
}

func (store *MySQLStore) fetchPrizeSummaries(ctx context.Context) ([]model.PrizeSummary, error) {
	rows, err := store.db.QueryContext(ctx, `SELECT id, name, level, stock FROM prizes ORDER BY campaign_id ASC, level ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.PrizeSummary, 0, 8)
	for rows.Next() {
		var item model.PrizeSummary
		if err := rows.Scan(&item.PrizeID, &item.PrizeName, &item.PrizeLevel, &item.Stock); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (store *MySQLStore) fetchQuotaBalance(ctx context.Context) (map[string]int, error) {
	rows, err := store.db.QueryContext(ctx, `SELECT user_id, campaign_id, used_count FROM user_campaign_quotas ORDER BY updated_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make(map[string]int)
	for rows.Next() {
		var userID, campaignID string
		var used int
		if err := rows.Scan(&userID, &campaignID, &used); err != nil {
			return nil, err
		}
		items[userID+":"+campaignID] = used
	}
	return items, nil
}

func (store *MySQLStore) readUserSession(ctx context.Context, token string) (model.Session, error) {
	if token == "" {
		return model.Session{}, ErrUnauthorized
	}
	if cached, err := store.redis.Get(ctx, store.userSessionKey(token)).Result(); err == nil {
		var session model.Session
		if json.Unmarshal([]byte(cached), &session) == nil && session.ExpiresAt.After(time.Now().UTC()) {
			return session, nil
		}
	}

	var session model.Session
	err := store.db.QueryRowContext(ctx, `SELECT token, user_id, expires_at FROM user_sessions WHERE token = ?`, token).Scan(&session.Token, &session.UserID, &session.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) || session.ExpiresAt.Before(time.Now().UTC()) {
		return model.Session{}, ErrUnauthorized
	}
	if err != nil {
		return model.Session{}, err
	}
	if err := store.cacheUserSession(ctx, session); err != nil {
		return model.Session{}, err
	}
	return session, nil
}

func (store *MySQLStore) cacheUserSession(ctx context.Context, session model.Session) error {
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return store.redis.Set(ctx, store.userSessionKey(session.Token), payload, time.Until(session.ExpiresAt)).Err()
}

func (store *MySQLStore) ensureAdmin(ctx context.Context, token string) error {
	if token == "" {
		return ErrAdminUnauthorized
	}
	result, err := store.redis.Get(ctx, store.adminSessionKey(token)).Result()
	if err == nil && result != "" {
		return nil
	}

	// Fallback: check MySQL if Redis missed (e.g. after restart)
	var userID string
	var expiresAt time.Time
	sqlErr := store.db.QueryRowContext(ctx,
		`SELECT user_id, expires_at FROM user_sessions WHERE token = ? AND user_id = 'admin'`,
		token).Scan(&userID, &expiresAt)
	if sqlErr != nil || userID != "admin" || time.Now().UTC().After(expiresAt) {
		return ErrAdminUnauthorized
	}

	// Re-cache into Redis for next time
	store.redis.Set(ctx, store.adminSessionKey(token), userID, time.Until(expiresAt))
	return nil
}

func (store *MySQLStore) campaignCacheKey() string {
	return store.redisPrefix + "campaigns"
}

func (store *MySQLStore) adminSessionKey(token string) string {
	return store.redisPrefix + "admin:session:" + token
}

func (store *MySQLStore) userSessionKey(token string) string {
	return store.redisPrefix + "user:session:" + token
}

func scanDrawRecords(rows *sql.Rows) ([]model.DrawRecord, error) {
	items := make([]model.DrawRecord, 0, 16)
	for rows.Next() {
		var item model.DrawRecord
		var prizeID sql.NullString
		if err := rows.Scan(&item.ID, &item.CampaignID, &item.UserID, &prizeID, &item.PrizeName, &item.Result, &item.DrawnAt, &item.ChanceAfter); err != nil {
			return nil, err
		}
		if prizeID.Valid {
			item.PrizeID = &prizeID.String
		}
		items = append(items, item)
	}
	return items, nil
}

func parseTaskID(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}

// 🆕 ---- v2.0 活动系统 MySQL stubs ----
func (store *MySQLStore) GetActiveActivities() []model.Activity { return nil }
func (store *MySQLStore) GetAllActivities() []model.Activity { return nil }
func (store *MySQLStore) GetActivity(activityID string) (*model.Activity, error) {
	return nil, fmt.Errorf("mysql activity not implemented, use memory store")
}
func (store *MySQLStore) CreateActivity(input model.ActivityCreateRequest) (*model.Activity, error) {
	return nil, fmt.Errorf("mysql activity not implemented, use memory store")
}
func (store *MySQLStore) UpdateActivity(activityID string, input model.ActivityUpdateRequest) (*model.Activity, error) {
	return nil, fmt.Errorf("mysql activity not implemented, use memory store")
}
func (store *MySQLStore) DeleteActivity(activityID string) error {
	return fmt.Errorf("mysql activity not implemented, use memory store")
}
func (store *MySQLStore) GetActivityRewards(activityID string) ([]model.ActivityReward, error) {
	return nil, fmt.Errorf("mysql activity not implemented, use memory store")
}
func (store *MySQLStore) JoinActivity(userID, activityID string) (*model.ActivityParticipation, error) {
	return nil, fmt.Errorf("mysql activity not implemented, use memory store")
}
func (store *MySQLStore) GetUserActivityParticipation(userID, activityID string) (*model.ActivityParticipation, error) {
	return nil, fmt.Errorf("mysql activity not implemented, use memory store")
}
func (store *MySQLStore) GetUserActivityParticipations(userID string) ([]model.ActivityParticipation, error) {
	return nil, fmt.Errorf("mysql activity not implemented, use memory store")
}
func (store *MySQLStore) ClaimActivityReward(userID, activityID, rewardID string) (*model.ActivityReward, error) {
	return nil, fmt.Errorf("mysql activity not implemented, use memory store")
}
