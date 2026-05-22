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

	// --- 盲盒专用种子数据 ---
	blindboxID := "camp_blindbox_001"
	_, err = store.db.ExecContext(ctx, `INSERT INTO campaigns (id, name, slug, status, starts_at, ends_at, daily_draw_limit, miss_weight, banner_image_url, campaign_summary,
		pity_enabled, soft_pity_n, pity_factor, hard_pity_n, target_prize_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 30, 0.0150, 60, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())`,
		blindboxID, "梦幻星辰系列盲盒", "dream-star-series", "online",
		now.Add(-24*time.Hour), now.Add(60*24*time.Hour),
		10, 30,
		"https://static.example.com/blindbox/dream-star/banner.png",
		"收集12款星辰主题公仔，集齐全套可兑换隐藏款！每抽必出，软保底30抽递增，60抽硬保底。",
		"prize_bb_secret")
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
	if usedCount >= campaign.DailyDrawLimit {
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

func (store *MySQLStore) AdminLogin(username string, password string) (string, error) {
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
	if err != nil || result == "" {
		return ErrAdminUnauthorized
	}
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

// ============================================================
// 盲盒扩展方法实现
// ============================================================

func (store *MySQLStore) GetCampaign(campaignID string) (model.Campaign, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	items, err := store.fetchCampaigns(ctx, true)
	if err != nil {
		return model.Campaign{}, err
	}
	for _, c := range items {
		if c.ID == campaignID {
			return c, nil
		}
	}
	return model.Campaign{}, ErrCampaignNotFound
}

func (store *MySQLStore) CreateDrawRecord(userID, campaignID, prizeID string, _ bool) (model.DrawRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return model.DrawRecord{}, err
	}
	defer tx.Rollback()

	// 扣库存
	var prizeName string
	err = tx.QueryRowContext(ctx, `SELECT name FROM prizes WHERE id = ? AND stock > 0 AND status = 'active' FOR UPDATE`, prizeID).Scan(&prizeName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.DrawRecord{}, fmt.Errorf("prize out of stock or not found")
		}
		return model.DrawRecord{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE prizes SET stock = stock - 1, updated_at = UTC_TIMESTAMP() WHERE id = ? AND stock > 0`, prizeID); err != nil {
		return model.DrawRecord{}, err
	}

	now := time.Now().UTC()
	record := model.DrawRecord{
		ID:         "draw_" + randomSuffix(12),
		CampaignID: campaignID,
		UserID:     userID,
		PrizeID:    &prizeID,
		PrizeName:  prizeName,
		Result:     "win",
		DrawnAt:    now,
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO draw_records (id, campaign_id, user_id, prize_id, prize_name, result, chance_after, request_id, created_at)
		VALUES (?, ?, ?, ?, ?, 'win', 0, '', ?)`, record.ID, campaignID, userID, prizeID, prizeName, now); err != nil {
		return model.DrawRecord{}, err
	}

	// 加入库存
	if _, err := tx.ExecContext(ctx, `INSERT INTO user_inventories (id, user_id, prize_id, prize_name, prize_level, campaign_id, source, created_at)
		VALUES (?, ?, ?, ?, (SELECT level FROM prizes WHERE id = ?), ?, 'draw', ?)`,
		"inv_"+randomSuffix(12), userID, prizeID, prizeName, prizeID, campaignID, now); err != nil {
		return model.DrawRecord{}, err
	}

	if err := tx.Commit(); err != nil {
		return model.DrawRecord{}, err
	}
	return record, nil
}

func (store *MySQLStore) CreateMissRecord(userID, campaignID string, _ bool) (model.DrawRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	now := time.Now().UTC()
	record := model.DrawRecord{
		ID:         "draw_" + randomSuffix(12),
		CampaignID: campaignID,
		UserID:     userID,
		PrizeName:  "未中奖",
		Result:     "miss",
		DrawnAt:    now,
	}
	_, err := store.db.ExecContext(ctx, `INSERT INTO draw_records (id, campaign_id, user_id, prize_id, prize_name, result, chance_after, request_id, created_at)
		VALUES (?, ?, ?, NULL, '未中奖', 'miss', 0, '', ?)`, record.ID, campaignID, userID, now)
	if err != nil {
		return model.DrawRecord{}, err
	}
	return record, nil
}

func (store *MySQLStore) CheckDrawQuota(userID, campaignID string, dailyLimit int) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	today := time.Now().UTC().Format("2006-01-02")
	var used int
	err := store.db.QueryRowContext(ctx, `SELECT used_count FROM user_campaign_quotas WHERE user_id = ? AND campaign_id = ? AND quota_date = ?`,
		userID, campaignID, today).Scan(&used)
	if errors.Is(err, sql.ErrNoRows) {
		return dailyLimit, nil
	}
	if err != nil {
		return 0, err
	}
	remaining := dailyLimit - used
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

func (store *MySQLStore) DeductDrawQuota(userID, campaignID string, count int) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	today := time.Now().UTC().Format("2006-01-02")

	_, err := store.db.ExecContext(ctx, `INSERT INTO user_campaign_quotas (user_id, campaign_id, quota_date, used_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())
		ON DUPLICATE KEY UPDATE used_count = used_count + ?, updated_at = UTC_TIMESTAMP()`,
		userID, campaignID, today, count, count)
	if err != nil {
		return 0, err
	}

	var used int
	store.db.QueryRowContext(ctx, `SELECT used_count FROM user_campaign_quotas WHERE user_id = ? AND campaign_id = ? AND quota_date = ?`,
		userID, campaignID, today).Scan(&used)

	// 返回剩余次数（需知道 dailyLimit，此处返回增量值，由 service 层计算精确剩余）
	return 99 - used, nil
}

func (store *MySQLStore) GetUserInventory(userID string) ([]model.UserInventory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := store.db.QueryContext(ctx, `SELECT id, user_id, prize_id, prize_name, prize_level, campaign_id, source, created_at
		FROM user_inventories WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.UserInventory, 0, 16)
	for rows.Next() {
		var item model.UserInventory
		if err := rows.Scan(&item.ID, &item.UserID, &item.PrizeID, &item.PrizeName, &item.PrizeLevel, &item.CampaignID, &item.Source, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (store *MySQLStore) GetSeriesProgress(userID, campaignID, campaignName string) (*model.SeriesProgress, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 获取系列总款式数
	var totalItems int
	err := store.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM prizes WHERE campaign_id = ? AND status = 'active'`, campaignID).Scan(&totalItems)
	if err != nil {
		return nil, err
	}

	// 获取用户已收集的款式
	rows, err := store.db.QueryContext(ctx, `SELECT prize_id, prize_name, prize_level, COUNT(1)
		FROM user_inventories WHERE user_id = ? AND campaign_id = ?
		GROUP BY prize_id, prize_name, prize_level`, userID, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	collected := make(map[string]int) // prizeID -> count
	prizeNames := make(map[string]string)
	prizeLevels := make(map[string]string)
	for rows.Next() {
		var prizeID, prizeName, prizeLevel string
		var count int
		if err := rows.Scan(&prizeID, &prizeName, &prizeLevel, &count); err != nil {
			return nil, err
		}
		collected[prizeID] = count
		prizeNames[prizeID] = prizeName
		prizeLevels[prizeID] = prizeLevel
	}

	// 构建已收集列表和缺失列表
	collectedItems := make([]model.CollectedPrize, 0, len(collected))
	for prizeID, count := range collected {
		collectedItems = append(collectedItems, model.CollectedPrize{
			Prize: model.Prize{ID: prizeID, Name: prizeNames[prizeID], Level: prizeLevels[prizeID]},
			Count: count,
		})
	}

	// 计算重复款数量
	duplicates := 0
	for _, count := range collected {
		if count > 1 {
			duplicates += count - 1
		}
	}

	progress := &model.SeriesProgress{
		CampaignID:      campaignID,
		CampaignName:    campaignName,
		TotalItems:      totalItems,
		CollectedItems:  len(collectedItems),
		Duplicates:      duplicates,
		CollectedPrizes: collectedItems,
	}
	if totalItems > 0 {
		progress.ProgressPercent = float64(len(collectedItems)) / float64(totalItems) * 100
	}

	// 缺失款式
	prizes, err := store.fetchPrizes(ctx, campaignID)
	if err == nil {
		for _, p := range prizes {
			if _, ok := collected[p.ID]; !ok {
				progress.MissingPrizes = append(progress.MissingPrizes, model.PrizeSummary{
					PrizeID: p.ID, PrizeName: p.Name, PrizeLevel: p.Level, Stock: p.Stock,
				})
			}
		}
	}

	return progress, nil
}

func (store *MySQLStore) UserHasPrize(userID, prizeID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var count int
	err := store.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM user_inventories WHERE user_id = ? AND prize_id = ?`, userID, prizeID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (store *MySQLStore) ExchangeOffers() []model.ExchangeOffer {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := store.db.QueryContext(ctx, `SELECT id, user_id, user_nickname, have_prize_id, have_prize_name, want_prize_id, want_prize_name, status, created_at
		FROM exchange_offers WHERE status = 'pending' ORDER BY created_at DESC LIMIT 50`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]model.ExchangeOffer, 0, 8)
	for rows.Next() {
		var item model.ExchangeOffer
		if err := rows.Scan(&item.ID, &item.UserID, &item.UserNickname, &item.HavePrizeID, &item.HavePrizeName,
			&item.WantPrizeID, &item.WantPrizeName, &item.Status, &item.CreatedAt); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (store *MySQLStore) CreateExchangeOffer(userID string, input model.ExchangeOfferMutation) (model.ExchangeOffer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 获取奖品名称
	var haveName, wantName string
	if err := store.db.QueryRowContext(ctx, `SELECT name FROM prizes WHERE id = ?`, input.HavePrizeID).Scan(&haveName); err != nil {
		return model.ExchangeOffer{}, fmt.Errorf("have_prize not found: %w", err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT name FROM prizes WHERE id = ?`, input.WantPrizeID).Scan(&wantName); err != nil {
		return model.ExchangeOffer{}, fmt.Errorf("want_prize not found: %w", err)
	}

	var nickname string
	store.db.QueryRowContext(ctx, `SELECT nickname FROM users WHERE id = ?`, userID).Scan(&nickname)

	offer := model.ExchangeOffer{
		ID:            "exch_" + randomSuffix(12),
		UserID:        userID,
		UserNickname:  nickname,
		HavePrizeID:   input.HavePrizeID,
		HavePrizeName: haveName,
		WantPrizeID:   input.WantPrizeID,
		WantPrizeName: wantName,
		Status:        "pending",
		CreatedAt:     time.Now().UTC(),
	}
	_, err := store.db.ExecContext(ctx, `INSERT INTO exchange_offers (id, user_id, user_nickname, have_prize_id, have_prize_name, want_prize_id, want_prize_name, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?)`,
		offer.ID, userID, nickname, input.HavePrizeID, haveName, input.WantPrizeID, wantName, offer.CreatedAt, offer.CreatedAt)
	if err != nil {
		return model.ExchangeOffer{}, err
	}
	return offer, nil
}

func (store *MySQLStore) CancelExchangeOffer(userID, offerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := store.db.ExecContext(ctx, `UPDATE exchange_offers SET status = 'cancelled', updated_at = UTC_TIMESTAMP() WHERE id = ? AND user_id = ? AND status = 'pending'`, offerID, userID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("offer not found or already matched")
	}
	return nil
}

func (store *MySQLStore) AcceptExchangeOffer(userID, offerID string) (model.ExchangeOffer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return model.ExchangeOffer{}, err
	}
	defer tx.Rollback()

	// 读取挂单信息（加锁）
	var offer model.ExchangeOffer
	err = tx.QueryRowContext(ctx, `SELECT id, user_id, user_nickname, have_prize_id, have_prize_name, want_prize_id, want_prize_name, status, created_at
		FROM exchange_offers WHERE id = ? AND status = 'pending' FOR UPDATE`, offerID).Scan(
		&offer.ID, &offer.UserID, &offer.UserNickname, &offer.HavePrizeID, &offer.HavePrizeName,
		&offer.WantPrizeID, &offer.WantPrizeName, &offer.Status, &offer.CreatedAt)
	if err != nil {
		return model.ExchangeOffer{}, fmt.Errorf("offer not available")
	}

	// 不能接受自己的挂单
	if offer.UserID == userID {
		return model.ExchangeOffer{}, fmt.Errorf("cannot accept your own offer")
	}

	// 确保双方都有对方想要的库存
	var accepterHasWant int
	tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM user_inventories WHERE user_id = ? AND prize_id = ?`, userID, offer.WantPrizeID).Scan(&accepterHasWant)
	if accepterHasWant == 0 {
		return model.ExchangeOffer{}, fmt.Errorf("you don't own the requested prize")
	}

	var offererHasHave int
	tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM user_inventories WHERE user_id = ? AND prize_id = ?`, offer.UserID, offer.HavePrizeID).Scan(&offererHasHave)
	if offererHasHave == 0 {
		return model.ExchangeOffer{}, fmt.Errorf("offerer's prize is no longer available")
	}

	// 删除双方库存中对应的条目（各一条）
	_, _ = tx.ExecContext(ctx, `DELETE FROM user_inventories WHERE user_id = ? AND prize_id = ? LIMIT 1`, userID, offer.WantPrizeID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM user_inventories WHERE user_id = ? AND prize_id = ? LIMIT 1`, offer.UserID, offer.HavePrizeID)

	// 插入交换后的库存
	now := time.Now().UTC()
	tx.ExecContext(ctx, `INSERT INTO user_inventories (id, user_id, prize_id, prize_name, prize_level, campaign_id, source, created_at)
		VALUES (?, ?, ?, ?, (SELECT level FROM prizes WHERE id = ?), (SELECT campaign_id FROM prizes WHERE id = ?), 'exchange', ?)`,
		"inv_"+randomSuffix(12), userID, offer.HavePrizeID, offer.HavePrizeName, offer.HavePrizeID, offer.HavePrizeID, now)
	tx.ExecContext(ctx, `INSERT INTO user_inventories (id, user_id, prize_id, prize_name, prize_level, campaign_id, source, created_at)
		VALUES (?, ?, ?, ?, (SELECT level FROM prizes WHERE id = ?), (SELECT campaign_id FROM prizes WHERE id = ?), 'exchange', ?)`,
		"inv_"+randomSuffix(12), offer.UserID, offer.WantPrizeID, offer.WantPrizeName, offer.WantPrizeID, offer.WantPrizeID, now)

	// 更新挂单状态
	tx.ExecContext(ctx, `UPDATE exchange_offers SET status = 'completed', updated_at = ? WHERE id = ?`, now, offerID)

	if err := tx.Commit(); err != nil {
		return model.ExchangeOffer{}, err
	}
	offer.Status = "completed"
	return offer, nil
}

func (store *MySQLStore) GetUserMember(userID string) (*model.UserMember, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var member model.UserMember
	err := store.db.QueryRowContext(ctx, `SELECT user_id, level, points, total_draws, total_spent, created_at, updated_at
		FROM user_members WHERE user_id = ?`, userID).Scan(
		&member.UserID, &member.Level, &member.Points, &member.TotalDraws, &member.TotalSpent, &member.CreatedAt, &member.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return &model.UserMember{
			UserID: userID,
			Level:  model.MemberNormal,
			Points: 0,
		}, nil
	}
	return &member, err
}

func (store *MySQLStore) UpdateUserMember(member *model.UserMember) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := store.db.ExecContext(ctx, `INSERT INTO user_members (user_id, level, points, total_draws, total_spent, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())
		ON DUPLICATE KEY UPDATE level = ?, points = ?, total_draws = ?, total_spent = ?, updated_at = UTC_TIMESTAMP()`,
		member.UserID, member.Level, member.Points, member.TotalDraws, member.TotalSpent,
		member.Level, member.Points, member.TotalDraws, member.TotalSpent)
	return err
}

func (store *MySQLStore) GetPointsLog(userID string) ([]model.UserPointsLog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := store.db.QueryContext(ctx, `SELECT id, user_id, points, balance, reason, remark, created_at
		FROM user_points_logs WHERE user_id = ? ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.UserPointsLog, 0, 16)
	for rows.Next() {
		var item model.UserPointsLog
		if err := rows.Scan(&item.ID, &item.UserID, &item.Points, &item.Balance, &item.Reason, &item.Remark, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (store *MySQLStore) LogPoints(userID string, points int64, balance int64, reason, remark string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := store.db.ExecContext(ctx, `INSERT INTO user_points_logs (user_id, points, balance, reason, remark, created_at)
		VALUES (?, ?, ?, ?, ?, UTC_TIMESTAMP())`, userID, points, balance, reason, remark)
	return err
}

func (store *MySQLStore) RedeemPrize(userID string, input model.RedeemRequest) (*model.RedeemResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 查询奖品信息和价格（用积分价格简化：普通=100,稀有=500,隐藏=2000）
	var prizeName, prizeLevel string
	if err := tx.QueryRowContext(ctx, `SELECT name, level FROM prizes WHERE id = ? AND status = 'active'`, input.PrizeID).Scan(&prizeName, &prizeLevel); err != nil {
		return nil, fmt.Errorf("prize not found")
	}
	pointsCost := map[string]int64{
		"common":  100, "rare": 500, "secret": 2000, "limited": 5000,
	}[prizeLevel]
	if pointsCost == 0 {
		pointsCost = 100
	}

	// 查余额
	var balance int64
	_ = tx.QueryRowContext(ctx, `SELECT points FROM user_members WHERE user_id = ? FOR UPDATE`, userID).Scan(&balance)
	if balance < pointsCost {
		return nil, fmt.Errorf("insufficient points: have %d, need %d", balance, pointsCost)
	}

	// 扣积分
	if _, err := tx.ExecContext(ctx, `UPDATE user_members SET points = points - ? WHERE user_id = ?`, pointsCost, userID); err != nil {
		return nil, err
	}

	// 记日志
	tx.ExecContext(ctx, `INSERT INTO user_points_logs (user_id, points, balance, reason, remark, created_at)
		VALUES (?, ?, ?, 'redeem', ?, UTC_TIMESTAMP())`, userID, -pointsCost, balance-pointsCost, "兑换: "+prizeName)

	// 加入库存
	now := time.Now().UTC()
	tx.ExecContext(ctx, `INSERT INTO user_inventories (id, user_id, prize_id, prize_name, prize_level, campaign_id, source, created_at)
		VALUES (?, ?, ?, ?, ?, (SELECT campaign_id FROM prizes WHERE id = ?), 'redeem', ?)`,
		"inv_"+randomSuffix(12), userID, input.PrizeID, prizeName, prizeLevel, input.PrizeID, now)

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &model.RedeemResult{
		RecordID:   "rdm_" + randomSuffix(12),
		PrizeID:    input.PrizeID,
		PrizeName:  prizeName,
		PointsCost: pointsCost,
		Remaining:  balance - pointsCost,
	}, nil
}

func (store *MySQLStore) GetDrawStatistics(token, campaignID string) (*model.DrawStatistics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.ensureAdmin(ctx, token); err != nil {
		return nil, err
	}

	var totalDraws, totalUsers, totalWins int64
	where := ""
	args := []any{}
	if campaignID != "" {
		where = " WHERE campaign_id = ?"
		args = append(args, campaignID)
	}

	store.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM draw_records`+where, args...).Scan(&totalDraws)
	store.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT user_id) FROM draw_records`+where, args...).Scan(&totalUsers)
	store.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM draw_records WHERE result = 'win'`+where, args...).Scan(&totalWins)

	winRate := 0.0
	if totalDraws > 0 {
		winRate = float64(totalWins) / float64(totalDraws) * 100
	}

	// 奖品分布
	prizeRows, err := store.db.QueryContext(ctx, `SELECT prize_id, prize_name, level, COUNT(1) as cnt
		FROM draw_records JOIN prizes ON draw_records.prize_id = prizes.id
		WHERE draw_records.result = 'win'`+where+` GROUP BY prize_id, prize_name, level ORDER BY cnt DESC`, args...)
	var prizeBreakdown []model.PrizeStatItem
	if err == nil {
		defer prizeRows.Close()
		for prizeRows.Next() {
			var item model.PrizeStatItem
			if prizeRows.Scan(&item.PrizeID, &item.PrizeName, &item.Level, &item.Count) == nil {
				if totalWins > 0 {
					item.Percent = float64(item.Count) / float64(totalWins) * 100
				}
				prizeBreakdown = append(prizeBreakdown, item)
			}
		}
	}

	return &model.DrawStatistics{
		TotalDraws:     totalDraws,
		TotalUsers:     totalUsers,
		TotalWins:      totalWins,
		WinRate:        winRate,
		PrizeBreakdown: prizeBreakdown,
	}, nil
}

// ============================================================
// 集卡系统扩展方法实现
// ============================================================

// DailyCheckIn 每日签到 - 简化实现：直接给 user_members 加积分
func (store *MySQLStore) DailyCheckIn(userID string, points int64) (*model.CheckInResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 先查 user_members，不存在则创建
	var currentPoints int64
	err := store.db.QueryRowContext(ctx, `SELECT points FROM user_members WHERE user_id = ?`, userID).Scan(&currentPoints)
	if errors.Is(err, sql.ErrNoRows) {
		now := time.Now().UTC()
		_, err = store.db.ExecContext(ctx, `INSERT INTO user_members (user_id, level, points, total_draws, total_spent, created_at, updated_at)
			VALUES (?, 'normal', 0, 0, 0, ?, ?)`, userID, now, now)
		if err != nil {
			return nil, err
		}
		currentPoints = 0
	} else if err != nil {
		return nil, err
	}

	newBalance := currentPoints + points

	// 加积分
	_, err = store.db.ExecContext(ctx, `UPDATE user_members SET points = points + ?, updated_at = UTC_TIMESTAMP() WHERE user_id = ?`, points, userID)
	if err != nil {
		return nil, err
	}

	// 记录积分日志
	_, err = store.db.ExecContext(ctx, `INSERT INTO user_points_logs (user_id, points, balance, reason, remark, created_at)
		VALUES (?, ?, ?, 'daily', '每日签到', UTC_TIMESTAMP())`, userID, points, newBalance)
	if err != nil {
		return nil, err
	}

	return &model.CheckInResult{
		PointsAwarded: points,
		StreakDays:    1,
		IsBonus:       false,
		NewBalance:    newBalance,
	}, nil
}

// GetCheckInStreak 获取连续签到天数 - 简化返回0
func (store *MySQLStore) GetCheckInStreak(userID string) (int, error) {
	return 0, nil
}

// CheckCollectionCompletion 检查用户是否集齐系列所有款式
func (store *MySQLStore) CheckCollectionCompletion(userID, campaignID string) (*model.CollectionReward, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 查询所有 active 的奖品，排除 secret/limited 级别
	rows, err := store.db.QueryContext(ctx, `SELECT id, name FROM prizes WHERE campaign_id = ? AND status = 'active' AND level NOT IN ('secret', 'limited')`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type prizeInfo struct {
		ID   string
		Name string
	}
	var allPrizes []prizeInfo
	for rows.Next() {
		var p prizeInfo
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		allPrizes = append(allPrizes, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(allPrizes) == 0 {
		return nil, nil
	}

	// 检查用户是否拥有所有这些款式
	for _, p := range allPrizes {
		var count int
		err := store.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM user_inventories WHERE user_id = ? AND prize_id = ?`, userID, p.ID).Scan(&count)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, nil // 缺少该款式
		}
	}

	// 集齐了
	var campaignName string
	store.db.QueryRowContext(ctx, `SELECT name FROM campaigns WHERE id = ?`, campaignID).Scan(&campaignName)

	return &model.CollectionReward{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		RewardType:   "title",
		RewardName:   "收集大师",
		Description:  "恭喜你集齐了「" + campaignName + "」系列所有款式！",
	}, nil
}

// GrantCollectionReward 发放集齐奖励
func (store *MySQLStore) GrantCollectionReward(userID string, reward *model.CollectionReward) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 插入一条 inventory 记录，source="collection_reward"
	_, err = tx.ExecContext(ctx, `INSERT INTO user_inventories (id, user_id, prize_id, prize_name, prize_level, campaign_id, source, created_at)
		VALUES (?, ?, ?, ?, 'collection', ?, 'collection_reward', UTC_TIMESTAMP())`,
		"inv_"+randomSuffix(12), userID, "reward_"+reward.RewardName, reward.RewardName, reward.CampaignID)
	if err != nil {
		return err
	}

	// 加500积分
	_, err = tx.ExecContext(ctx, `UPDATE user_members SET points = points + 500, updated_at = UTC_TIMESTAMP() WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	// 查询当前余额
	var balance int64
	tx.QueryRowContext(ctx, `SELECT points FROM user_members WHERE user_id = ?`, userID).Scan(&balance)

	// 记录积分日志
	_, err = tx.ExecContext(ctx, `INSERT INTO user_points_logs (user_id, points, balance, reason, remark, created_at)
		VALUES (?, ?, ?, 'collection', ?, UTC_TIMESTAMP())`, userID, int64(500), balance, "集卡奖励: "+reward.RewardName)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetLeaderboard 获取收集排行榜
func (store *MySQLStore) GetLeaderboard(limit int) ([]model.LeaderboardEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := store.db.QueryContext(ctx, `SELECT user_id, COUNT(DISTINCT prize_id) as collected
		FROM user_inventories GROUP BY user_id ORDER BY collected DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]model.LeaderboardEntry, 0)
	rank := 0
	for rows.Next() {
		rank++
		var entry model.LeaderboardEntry
		if err := rows.Scan(&entry.UserID, &entry.CollectedCount); err != nil {
			return nil, err
		}
		entry.Rank = rank
		// 获取昵称
		store.db.QueryRowContext(ctx, `SELECT nickname FROM users WHERE id = ?`, entry.UserID).Scan(&entry.Nickname)
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

// GetCampaignHint 随机返回一条摇盒提示文案
func (store *MySQLStore) GetCampaignHint(campaignID string) *model.HintMessage {
	hints := []model.HintMessage{
		{Type: "hot", Content: "这个系列的隐藏款好像特别轻，摇晃时有细微的零件松动声。"},
		{Type: "social", Content: "听说今天很多玩家都在抽这个系列，热门款式快要被抽光了！"},
		{Type: "luck", Content: "今天的幸运色是红色，试试在整点时刻抽取？"},
	}
	return &hints[rand.IntN(len(hints))]
}

// ShareReward 分享奖励
func (store *MySQLStore) ShareReward(userID string, points int64) (*model.ShareRewardResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 查询当前余额
	var balance int64
	err = tx.QueryRowContext(ctx, `SELECT points FROM user_members WHERE user_id = ? FOR UPDATE`, userID).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		now := time.Now().UTC()
		_, err = tx.ExecContext(ctx, `INSERT INTO user_members (user_id, level, points, total_draws, total_spent, created_at, updated_at)
			VALUES (?, 'normal', 0, 0, 0, ?, ?)`, userID, now, now)
		if err != nil {
			return nil, err
		}
		balance = 0
	} else if err != nil {
		return nil, err
	}

	newBalance := balance + points

	// 加积分
	_, err = tx.ExecContext(ctx, `UPDATE user_members SET points = points + ?, updated_at = UTC_TIMESTAMP() WHERE user_id = ?`, points, userID)
	if err != nil {
		return nil, err
	}

	// 记录积分日志
	_, err = tx.ExecContext(ctx, `INSERT INTO user_points_logs (user_id, points, balance, reason, remark, created_at)
		VALUES (?, ?, ?, 'share', '分享奖励', UTC_TIMESTAMP())`, userID, points, newBalance)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &model.ShareRewardResult{
		PointsAwarded: points,
		DailyLeft:     9,
		NewBalance:    newBalance,
	}, nil
}

// GetShareDailyCount 获取今日已分享次数 - 简化返回0
func (store *MySQLStore) GetShareDailyCount(userID string) (int, error) {
	return 0, nil
}

// GetPrizeCount 获取用户某系列某款式的数量
func (store *MySQLStore) GetPrizeCount(userID, prizeID string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var count int
	err := store.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM user_inventories WHERE user_id = ? AND prize_id = ?`, userID, prizeID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (store *MySQLStore) BlendPrizes(userID string, sourcePrizeID string, campaignID string) (*model.BlendResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. 查找源款式信息
	var sourcePrize model.Prize
	var sourceCampaignID string
	err := store.db.QueryRowContext(ctx,
		`SELECT id, campaign_id, name, level, stock, probability_weight, status FROM prizes WHERE id = ? AND status = 'active'`,
		sourcePrizeID,
	).Scan(&sourcePrize.ID, &sourceCampaignID, &sourcePrize.Name, &sourcePrize.Level,
		&sourcePrize.Stock, &sourcePrize.ProbabilityWeight, &sourcePrize.Status)
	if err != nil {
		return nil, fmt.Errorf("source prize not found: %w", err)
	}
	sourcePrize.CampaignID = sourceCampaignID

	// 2. 查找合成配方
	recipe, ok := model.BlendRecipes[sourcePrize.Level]
	if !ok {
		return nil, fmt.Errorf("no blend recipe for level: %s", sourcePrize.Level)
	}

	// 3. 检查用户拥有数量
	var haveCount int
	err = store.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM user_inventories WHERE user_id = ? AND prize_id = ?`,
		userID, sourcePrizeID,
	).Scan(&haveCount)
	if err != nil {
		return nil, err
	}
	if haveCount < recipe.NeedCount {
		return nil, fmt.Errorf("need %d of %s, have %d", recipe.NeedCount, sourcePrize.Name, haveCount)
	}

	// 4. 查找目标级别的一个可用奖品（同系列同级随机选一个）
	var resultPrize model.Prize
	var resultCampaignID string
	err = store.db.QueryRowContext(ctx,
		`SELECT id, campaign_id, name, level, stock, probability_weight, status FROM prizes
		 WHERE campaign_id = ? AND level = ? AND status = 'active' AND stock > 0
		 AND id != ? ORDER BY RAND() LIMIT 1`,
		campaignID, recipe.ResultLevel, sourcePrizeID,
	).Scan(&resultPrize.ID, &resultCampaignID, &resultPrize.Name, &resultPrize.Level,
		&resultPrize.Stock, &resultPrize.ProbabilityWeight, &resultPrize.Status)
	if err != nil {
		return nil, fmt.Errorf("no available prize of level %s to blend into: %w", recipe.ResultLevel, err)
	}

	// 5. 事务执行合成
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 5a. 删除 N 条源款式库存记录（按创建时间升序，消耗最早的）
	resultDelete, err := tx.ExecContext(ctx,
		`DELETE FROM user_inventories
		 WHERE user_id = ? AND prize_id = ?
		 ORDER BY created_at ASC
		 LIMIT ?`,
		userID, sourcePrizeID, recipe.NeedCount,
	)
	if err != nil {
		return nil, err
	}
	deleted, _ := resultDelete.RowsAffected()
	if int(deleted) < recipe.NeedCount {
		return nil, fmt.Errorf("concurrent blend detected: expected %d rows, deleted %d", recipe.NeedCount, deleted)
	}

	// 5b. 扣减库存
	_, err = tx.ExecContext(ctx,
		`UPDATE prizes SET stock = stock - 1 WHERE id = ? AND stock > 0`,
		resultPrize.ID,
	)
	if err != nil {
		return nil, err
	}

	// 5c. 添加结果到用户库存
	now := time.Now().UTC()
	invID := "inv_" + randomSuffix(16)
	_, err = tx.ExecContext(ctx,
		`INSERT INTO user_inventories (id, user_id, prize_id, campaign_id, source, created_at)
		 VALUES (?, ?, ?, ?, 'blend', ?)`,
		invID, userID, resultPrize.ID, campaignID, now,
	)
	if err != nil {
		return nil, err
	}

	// 5d. 记录合成日志
	_, err = tx.ExecContext(ctx,
		`INSERT INTO lottery_logs (id, user_id, campaign_id, prize_id, result, chance_after, drawn_at)
		 VALUES (?, ?, ?, ?, 'blend', 0, ?)`,
		"blend_"+randomSuffix(16), userID, campaignID, resultPrize.ID, now,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &model.BlendResult{
		SourcePrizeID:   sourcePrizeID,
		SourcePrizeName: sourcePrize.Name,
		SourceLevel:     sourcePrize.Level,
		ResultPrizeID:   resultPrize.ID,
		ResultPrizeName: resultPrize.Name,
		ResultLevel:     resultPrize.Level,
		RemainingSrc:    haveCount - recipe.NeedCount,
	}, nil
}
