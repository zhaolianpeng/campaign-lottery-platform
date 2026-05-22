package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"campaign-lottery-platform/backend/internal/model"
)

// ============================================================
// 盲盒扩展的 MySQL 实现
// ============================================================

func (store *MySQLStore) GetCampaign(campaignID string) (model.Campaign, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var c model.Campaign
	err := store.db.QueryRowContext(ctx, `SELECT id, name, slug, status, starts_at, ends_at, daily_draw_limit, miss_weight, banner_image_url, campaign_summary
		FROM campaigns WHERE id = ?`, campaignID).
		Scan(&c.ID, &c.Name, &c.Slug, &c.Status, &c.StartsAt, &c.EndsAt, &c.DailyDrawLimit, &c.MissWeight, &c.BannerImageURL, &c.CampaignSummary)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Campaign{}, ErrCampaignNotFound
	}
	return c, err
}

func (store *MySQLStore) CreateDrawRecord(userID, campaignID, prizeID string, isTenPull bool) (model.DrawRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return model.DrawRecord{}, err
	}
	defer tx.Rollback()

	// Deduct stock
	var prizeName string
	err = tx.QueryRowContext(ctx, `SELECT name FROM prizes WHERE id = ? AND stock > 0 FOR UPDATE`, prizeID).Scan(&prizeName)
	if errors.Is(err, sql.ErrNoRows) {
		return model.DrawRecord{}, fmt.Errorf("prize out of stock")
	}
	if err != nil {
		return model.DrawRecord{}, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE prizes SET stock = stock - 1, updated_at = UTC_TIMESTAMP() WHERE id = ? AND stock > 0`, prizeID)
	if err != nil {
		return model.DrawRecord{}, err
	}

	now := time.Now().UTC()
	recordID := "draw_" + randomSuffix(12)
	record := model.DrawRecord{
		ID: recordID, CampaignID: campaignID, UserID: userID, PrizeName: prizeName,
		Result: "win", DrawnAt: now,
		PrizeID: &prizeID,
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO draw_records (id, campaign_id, user_id, prize_id, prize_name, result, chance_after, request_id, is_ten_pull, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, '', ?, UTC_TIMESTAMP())`, recordID, campaignID, userID, prizeID, prizeName, "win", isTenPull)
	if err != nil {
		return model.DrawRecord{}, err
	}

	// Add to user inventory
	_, err = tx.ExecContext(ctx, `INSERT INTO user_inventories (id, user_id, prize_id, prize_name, prize_level, campaign_id, source, source_id, created_at)
		SELECT ?, ?, ?, name, level, ?, 'draw', ?, UTC_TIMESTAMP() FROM prizes WHERE id = ?`,
		"inv_"+randomSuffix(12), userID, prizeID, campaignID, recordID, prizeID)
	if err != nil {
		return model.DrawRecord{}, err
	}

	// Create fulfillment task
	payload, _ := json.Marshal(map[string]any{"source": "lottery_draw", "draw_record_id": recordID, "prize_id": prizeID})
	_, err = tx.ExecContext(ctx, `INSERT INTO prize_fulfillment_tasks (draw_record_id, user_id, prize_id, status, payload_json, operator_note, created_at, updated_at)
		VALUES (?, ?, ?, 'pending', ?, '', UTC_TIMESTAMP(), UTC_TIMESTAMP())`, recordID, userID, prizeID, string(payload))
	if err != nil {
		return model.DrawRecord{}, err
	}

	if err := tx.Commit(); err != nil {
		return model.DrawRecord{}, err
	}
	store.redis.Del(ctx, store.campaignCacheKey())
	return record, nil
}

func (store *MySQLStore) CreateMissRecord(userID, campaignID string, isTenPull bool) (model.DrawRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	recordID := "draw_" + randomSuffix(12)
	_, err := store.db.ExecContext(ctx, `INSERT INTO draw_records (id, campaign_id, user_id, prize_name, result, chance_after, request_id, is_ten_pull, created_at)
		VALUES (?, ?, ?, '未中奖', 'miss', 0, '', ?, UTC_TIMESTAMP())`, recordID, campaignID, userID, isTenPull)
	if err != nil {
		return model.DrawRecord{}, err
	}
	return model.DrawRecord{
		ID: recordID, CampaignID: campaignID, UserID: userID,
		PrizeName: "未中奖", Result: "miss", DrawnAt: time.Now().UTC(),
	}, nil
}

func (store *MySQLStore) CheckDrawQuota(userID, campaignID string, dailyLimit int) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	quotaDate := time.Now().UTC().Format("2006-01-02")
	var used int
	err := store.db.QueryRowContext(ctx, `SELECT used_count FROM user_campaign_quotas
		WHERE user_id = ? AND campaign_id = ? AND quota_date = ?`, userID, campaignID, quotaDate).Scan(&used)
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

	quotaDate := time.Now().UTC().Format("2006-01-02")
	_, err := store.db.ExecContext(ctx, `INSERT INTO user_campaign_quotas (user_id, campaign_id, quota_date, used_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())
		ON DUPLICATE KEY UPDATE used_count = used_count + ?, updated_at = UTC_TIMESTAMP()`,
		userID, campaignID, quotaDate, count, count)
	if err != nil {
		return 0, err
	}

	var used int
	err = store.db.QueryRowContext(ctx, `SELECT used_count FROM user_campaign_quotas
		WHERE user_id = ? AND campaign_id = ? AND quota_date = ?`, userID, campaignID, quotaDate).Scan(&used)
	if err != nil {
		return 0, err
	}
	// We don't know dailyLimit here, return used count (caller knows limit)
	return used, nil
}

func (store *MySQLStore) GetUserInventory(userID string) ([]model.UserInventory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := store.db.QueryContext(ctx, `SELECT id, user_id, prize_id, prize_name, COALESCE(prize_level, ''), campaign_id, source, created_at
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

	// Get total items in series
	var totalItems int
	err := store.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM prizes WHERE campaign_id = ? AND status = 'active'`, campaignID).Scan(&totalItems)
	if err != nil {
		return nil, err
	}

	// Get user's collected items per prize in this campaign
	rows, err := store.db.QueryContext(ctx, `SELECT inv.prize_id, p.name, p.level, COUNT(1)
		FROM user_inventories inv
		JOIN prizes p ON p.id = inv.prize_id
		WHERE inv.user_id = ? AND inv.campaign_id = ?
		GROUP BY inv.prize_id, p.name, p.level`, userID, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type collectedInfo struct {
		Prize model.Prize
		Count int
	}
	collectedMap := make(map[string]*collectedInfo)
	for rows.Next() {
		var ci collectedInfo
		if err := rows.Scan(&ci.Prize.ID, &ci.Prize.Name, &ci.Prize.Level, &ci.Count); err != nil {
			return nil, err
		}
		collectedMap[ci.Prize.ID] = &ci
	}

	// Get all prizes for this campaign
	allRows, err := store.db.QueryContext(ctx, `SELECT id, campaign_id, name, level, stock, probability_weight, status
		FROM prizes WHERE campaign_id = ? ORDER BY level ASC, id ASC`, campaignID)
	if err != nil {
		return nil, err
	}
	defer allRows.Close()

	collected := make([]model.CollectedPrize, 0, len(collectedMap))
	missing := make([]model.PrizeSummary, 0)
	duplicates := 0

	for allRows.Next() {
		var p model.Prize
		if err := allRows.Scan(&p.ID, &p.CampaignID, &p.Name, &p.Level, &p.Stock, &p.ProbabilityWeight, &p.Status); err != nil {
			return nil, err
		}
		if ci, ok := collectedMap[p.ID]; ok {
			collected = append(collected, model.CollectedPrize{Prize: p, Count: ci.Count})
			if ci.Count > 1 {
				duplicates += ci.Count - 1
			}
		} else {
			missing = append(missing, model.PrizeSummary{
				PrizeID: p.ID, PrizeName: p.Name, PrizeLevel: p.Level,
			})
		}
	}

	pct := 0.0
	if totalItems > 0 {
		pct = float64(len(collected)) / float64(totalItems) * 100
	}
	return &model.SeriesProgress{
		CampaignID: campaignID, CampaignName: campaignName,
		TotalItems: totalItems, CollectedItems: len(collected),
		ProgressPercent: pct, Duplicates: duplicates,
		CollectedPrizes: collected, MissingPrizes: missing,
	}, nil
}

func (store *MySQLStore) UserHasPrize(userID, prizeID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var count int
	err := store.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM user_inventories WHERE user_id = ? AND prize_id = ?`, userID, prizeID).Scan(&count)
	return count > 0, err
}

// ============================================================
// 交换市场
// ============================================================

func (store *MySQLStore) ExchangeOffers() []model.ExchangeOffer {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := store.db.QueryContext(ctx, `SELECT eo.id, eo.user_id, u.nickname,
		eo.have_prize_id, COALESCE(p1.name, ''), eo.want_prize_id, COALESCE(p2.name, ''),
		eo.status, eo.created_at
		FROM exchange_offers eo
		LEFT JOIN users u ON u.id = eo.user_id
		LEFT JOIN prizes p1 ON p1.id = eo.have_prize_id
		LEFT JOIN prizes p2 ON p2.id = eo.want_prize_id
		WHERE eo.status = 'pending'
		ORDER BY eo.created_at DESC`)
	if err != nil {
		return []model.ExchangeOffer{}
	}
	defer rows.Close()

	items := make([]model.ExchangeOffer, 0, 8)
	for rows.Next() {
		var o model.ExchangeOffer
		if err := rows.Scan(&o.ID, &o.UserID, &o.UserNickname,
			&o.HavePrizeID, &o.HavePrizeName, &o.WantPrizeID, &o.WantPrizeName,
			&o.Status, &o.CreatedAt); err != nil {
			continue
		}
		items = append(items, o)
	}
	return items
}

func (store *MySQLStore) CreateExchangeOffer(userID string, input model.ExchangeOfferMutation) (model.ExchangeOffer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id := "ex_offer_" + randomSuffix(10)
	_, err := store.db.ExecContext(ctx, `INSERT INTO exchange_offers (id, user_id, have_prize_id, want_prize_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'pending', UTC_TIMESTAMP(), UTC_TIMESTAMP())`, id, userID, input.HavePrizeID, input.WantPrizeID)
	if err != nil {
		return model.ExchangeOffer{}, err
	}
	return model.ExchangeOffer{
		ID: id, UserID: userID, HavePrizeID: input.HavePrizeID,
		WantPrizeID: input.WantPrizeID, Status: "pending", CreatedAt: time.Now().UTC(),
	}, nil
}

func (store *MySQLStore) CancelExchangeOffer(userID, offerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := store.db.ExecContext(ctx, `UPDATE exchange_offers SET status = 'cancelled', updated_at = UTC_TIMESTAMP()
		WHERE id = ? AND user_id = ? AND status = 'pending'`, offerID, userID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return errors.New("offer not found or not yours")
	}
	return nil
}

func (store *MySQLStore) AcceptExchangeOffer(userID, offerID string) (model.ExchangeOffer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := store.db.ExecContext(ctx, `UPDATE exchange_offers SET status = 'matched', matched_user_id = ?, updated_at = UTC_TIMESTAMP()
		WHERE id = ? AND status = 'pending' AND user_id != ?`, userID, offerID, userID)
	if err != nil {
		return model.ExchangeOffer{}, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return model.ExchangeOffer{}, errors.New("offer not found or already matched")
	}

	var o model.ExchangeOffer
	err = store.db.QueryRowContext(ctx, `SELECT id, user_id, have_prize_id, want_prize_id, status, created_at
		FROM exchange_offers WHERE id = ?`, offerID).
		Scan(&o.ID, &o.UserID, &o.HavePrizeID, &o.WantPrizeID, &o.Status, &o.CreatedAt)
	return o, err
}

// ============================================================
// 积分/会员
// ============================================================

func (store *MySQLStore) GetUserMember(userID string) (*model.UserMember, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var m model.UserMember
	err := store.db.QueryRowContext(ctx, `SELECT user_id, level, points, total_draws, total_spent, created_at, updated_at
		FROM user_members WHERE user_id = ?`, userID).Scan(&m.UserID, &m.Level, &m.Points, &m.TotalDraws, &m.TotalSpent, &m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return &model.UserMember{UserID: userID, Level: model.MemberNormal}, nil
	}
	return &m, err
}

func (store *MySQLStore) GetPointsLog(userID string) ([]model.UserPointsLog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := store.db.QueryContext(ctx, `SELECT id, user_id, points, balance, reason, COALESCE(remark, ''), created_at
		FROM user_points_logs WHERE user_id = ? ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.UserPointsLog, 0, 16)
	for rows.Next() {
		var log model.UserPointsLog
		if err := rows.Scan(&log.ID, &log.UserID, &log.Points, &log.Balance, &log.Reason, &log.Remark, &log.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, log)
	}
	return items, nil
}

func (store *MySQLStore) RedeemPrize(userID string, input model.RedeemRequest) (*model.RedeemResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Get member points
	var points int64
	err = tx.QueryRowContext(ctx, `SELECT points FROM user_members WHERE user_id = ? FOR UPDATE`, userID).Scan(&points)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("member not found")
	}
	if err != nil {
		return nil, err
	}

	// Get prize info and cost
	var prizeName string
	var cost int64
	err = tx.QueryRowContext(ctx, `SELECT name, 100 FROM prizes WHERE id = ? AND status = 'active' AND stock > 0`, input.PrizeID).Scan(&prizeName, &cost)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("prize not available")
	}
	if err != nil {
		return nil, err
	}

	if points < cost {
		return nil, errors.New("insufficient points")
	}

	// Deduct points
	newBalance := points - cost
	_, err = tx.ExecContext(ctx, `UPDATE user_members SET points = ?, updated_at = UTC_TIMESTAMP() WHERE user_id = ?`, newBalance, userID)
	if err != nil {
		return nil, err
	}

	// Log
	_, err = tx.ExecContext(ctx, `INSERT INTO user_points_logs (user_id, points, balance, reason, remark, created_at)
		VALUES (?, ?, ?, 'redeem', ?, UTC_TIMESTAMP())`, userID, -cost, newBalance, "兑换: "+prizeName)
	if err != nil {
		return nil, err
	}

	// Add inventory
	invID := "inv_" + randomSuffix(12)
	_, err = tx.ExecContext(ctx, `INSERT INTO user_inventories (id, user_id, prize_id, prize_name, source, created_at)
		VALUES (?, ?, ?, ?, 'redeem', UTC_TIMESTAMP())`, invID, userID, input.PrizeID, prizeName)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &model.RedeemResult{
		RecordID: invID, PrizeID: input.PrizeID, PrizeName: prizeName,
		PointsCost: cost, Remaining: newBalance,
	}, nil
}

// ============================================================
// 数据统计
// ============================================================

func (store *MySQLStore) GetDrawStatistics(token, campaignID string) (*model.DrawStatistics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	if err := store.ensureAdmin(ctx, token); err != nil {
		return nil, err
	}

	where := ""
	args := []any{}
	if campaignID != "" {
		where = " WHERE campaign_id = ?"
		args = append(args, campaignID)
	}

	var totalDraws, totalWins int64
	if err := store.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM draw_records"+where, args...).Scan(&totalDraws); err != nil {
		return nil, err
	}
	if err := store.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM draw_records"+where+" AND result='win'", args...).Scan(&totalWins); err != nil {
		return nil, err
	}

	winRate := 0.0
	if totalDraws > 0 {
		winRate = float64(totalWins) / float64(totalDraws) * 100
	}

	// Prize breakdown
	query := `SELECT p.id, p.name, p.level, COUNT(dr.id)
		FROM prizes p
		LEFT JOIN draw_records dr ON dr.prize_id = p.id AND dr.result = 'win'`
	if campaignID != "" {
		query += " WHERE p.campaign_id = ?"
	}
	query += " GROUP BY p.id, p.name, p.level ORDER BY COUNT(dr.id) DESC"

	var prizeRows *sql.Rows
	var err error
	if campaignID != "" {
		prizeRows, err = store.db.QueryContext(ctx, query, campaignID)
	} else {
		prizeRows, err = store.db.QueryContext(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	defer prizeRows.Close()

	breakdown := make([]model.PrizeStatItem, 0, 8)
	for prizeRows.Next() {
		var item model.PrizeStatItem
		if err := prizeRows.Scan(&item.PrizeID, &item.PrizeName, &item.Level, &item.Count); err != nil {
			continue
		}
		if totalWins > 0 {
			item.Percent = float64(item.Count) / float64(totalWins) * 100
		}
		breakdown = append(breakdown, item)
	}

	var totalUsers int64
	store.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM users").Scan(&totalUsers)

	return &model.DrawStatistics{
		TotalDraws: totalDraws, TotalUsers: totalUsers, TotalWins: totalWins,
		WinRate: winRate, PrizeBreakdown: breakdown,
	}, nil
}

