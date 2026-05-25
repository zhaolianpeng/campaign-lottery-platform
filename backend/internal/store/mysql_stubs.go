package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"campaign-lottery-platform/backend/internal/model"
)

// ============================================================
// Stub methods — use memory store for these features
// ============================================================

// ---- 集卡系统 stubs ----

func (store *MySQLStore) DailyCheckIn(userID string, points int64) (*model.CheckInResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	today := now.Format("2006-01-02")

	// Check if already checked in today via points_logs (reason='daily')
	var existingCount int
	err := store.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM user_points_logs WHERE user_id = ? AND reason = 'daily' AND DATE(created_at) = ?", userID, today).Scan(&existingCount)
	if err != nil {
		return nil, err
	}
	if existingCount > 0 {
		return nil, fmt.Errorf("already checked in today")
	}

	// Calculate streak from previous checkin
	streak := 1
	var lastLogTime sql.NullTime
	store.db.QueryRowContext(ctx, "SELECT MAX(created_at) FROM user_points_logs WHERE user_id = ? AND reason = 'daily'", userID).Scan(&lastLogTime)
	if lastLogTime.Valid {
		lastDay := lastLogTime.Time.Truncate(24 * time.Hour)
		todayTime := now.Truncate(24 * time.Hour)
		diff := todayTime.Sub(lastDay)
		if diff == 24*time.Hour {
			// Count previous consecutive checkins for this streak
			store.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM user_points_logs WHERE user_id = ? AND reason = 'daily' AND created_at >= DATE_SUB(?, INTERVAL 7 DAY)", userID, now).Scan(&streak)
			if streak < 1 {
				streak = 1
			}
		}
		// if diff > 24h, streak resets to 1 (already set above)
	}

	// Calculate bonus at streak=7
	totalPoints := points
	isBonus := false
	if streak >= 7 {
		totalPoints += 20
		isBonus = true
		streak = 0
	}

	// Get or create user member, then update points
	var member model.UserMember
	err = store.db.QueryRowContext(ctx, "SELECT user_id, level, points FROM user_members WHERE user_id = ?", userID).Scan(&member.UserID, &member.Level, &member.Points)
	if err == sql.ErrNoRows {
		member = model.UserMember{UserID: userID, Level: model.MemberNormal, Points: 0}
		_, err = store.db.ExecContext(ctx, "INSERT INTO user_members (user_id, level, points, created_at, updated_at) VALUES (?, ?, 0, UTC_TIMESTAMP(), UTC_TIMESTAMP())", userID, model.MemberNormal)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	member.Points += totalPoints
	_, err = store.db.ExecContext(ctx, "UPDATE user_members SET points = ?, updated_at = UTC_TIMESTAMP() WHERE user_id = ?", member.Points, userID)
	if err != nil {
		return nil, err
	}

	// Log the checkin
	_, err = store.db.ExecContext(ctx, "INSERT INTO user_points_logs (user_id, points, balance, reason, remark, created_at) VALUES (?, ?, ?, 'daily', '每日签到', UTC_TIMESTAMP())", userID, totalPoints, member.Points)
	if err != nil {
		return nil, err
	}

	return &model.CheckInResult{
		PointsAwarded: totalPoints,
		StreakDays:    streak,
		IsBonus:       isBonus,
		NewBalance:    member.Points,
	}, nil
}

func (store *MySQLStore) GetCheckInStreak(userID string) (int, error) {
	return 0, fmt.Errorf("mysql check-in streak not implemented, use memory store")
}

func (store *MySQLStore) CheckCollectionCompletion(userID, campaignID string) (*model.CollectionReward, error) {
	return nil, fmt.Errorf("mysql collection completion not implemented, use memory store")
}

func (store *MySQLStore) GrantCollectionReward(userID string, reward *model.CollectionReward) error {
	return fmt.Errorf("mysql collection reward not implemented, use memory store")
}

func (store *MySQLStore) GetLeaderboard(limit int) ([]model.LeaderboardEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `SELECT um.user_id, COALESCE(u.nickname, ''), um.points
		FROM user_members um
		LEFT JOIN users u ON u.id = um.user_id
		ORDER BY um.points DESC
		LIMIT ?`
	rows, err := store.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]model.LeaderboardEntry, 0, limit)
	rank := 1
	for rows.Next() {
		var userID, nickname string
		var points int64
		if err := rows.Scan(&userID, &nickname, &points); err != nil {
			continue
		}
		entries = append(entries, model.LeaderboardEntry{
			Rank:            rank,
			UserID:          userID,
			Nickname:        nickname,
			CollectedCount:  int(points),
			TotalCount:      0,
			ProgressPercent: 0,
			SeriesCompleted: 0,
		})
		rank++
	}
	if entries == nil {
		return []model.LeaderboardEntry{}, nil
	}
	return entries, nil
}

func (store *MySQLStore) GetCampaignHint(campaignID string) *model.HintMessage {
	return nil
}

func (store *MySQLStore) ShareReward(userID string, points int64) (*model.ShareRewardResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	today := now.Format("2006-01-02")

	// Check daily share count from points_logs (reason='share')
	var dailyCount int
	store.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM user_points_logs WHERE user_id = ? AND reason = 'share' AND DATE(created_at) = ?", userID, today).Scan(&dailyCount)

	if dailyCount >= 10 {
		return nil, fmt.Errorf("今日分享次数已达上限")
	}

	// Get or create user member
	var member model.UserMember
	err := store.db.QueryRowContext(ctx, "SELECT user_id, level, points FROM user_members WHERE user_id = ?", userID).Scan(&member.UserID, &member.Level, &member.Points)
	if err == sql.ErrNoRows {
		member = model.UserMember{UserID: userID, Level: model.MemberNormal, Points: 0}
		_, err = store.db.ExecContext(ctx, "INSERT INTO user_members (user_id, level, points, created_at, updated_at) VALUES (?, ?, 0, UTC_TIMESTAMP(), UTC_TIMESTAMP())", userID, model.MemberNormal)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	member.Points += points
	_, err = store.db.ExecContext(ctx, "UPDATE user_members SET points = ?, updated_at = UTC_TIMESTAMP() WHERE user_id = ?", member.Points, userID)
	if err != nil {
		return nil, err
	}

	// Log share reward
	_, err = store.db.ExecContext(ctx, "INSERT INTO user_points_logs (user_id, points, balance, reason, remark, created_at) VALUES (?, ?, ?, 'share', '分享奖励', UTC_TIMESTAMP())", userID, points, member.Points)
	if err != nil {
		return nil, err
	}

	return &model.ShareRewardResult{
		PointsAwarded: points,
		DailyLeft:     10 - dailyCount - 1,
		NewBalance:    member.Points,
	}, nil
}

func (store *MySQLStore) GetShareDailyCount(userID string) (int, error) {
	return 0, fmt.Errorf("mysql share daily count not implemented, use memory store")
}

func (store *MySQLStore) GetPrizeCount(userID, prizeID string) (int, error) {
	return 0, fmt.Errorf("mysql prize count not implemented, use memory store")
}

func (store *MySQLStore) BlendPrizes(userID string, sourcePrizeID string, campaignID string) (*model.BlendResult, error) {
	return nil, fmt.Errorf("mysql blend prizes not implemented, use memory store")
}

// ---- 积分/会员 stubs (supplementary) ----

func (store *MySQLStore) UpdateUserMember(member *model.UserMember) error {
	return fmt.Errorf("mysql update user member not implemented, use memory store")
}

func (store *MySQLStore) LogPoints(userID string, points int64, balance int64, reason, remark string) error {
	return fmt.Errorf("mysql log points not implemented, use memory store")
}

// ---- 月卡系统 stubs ----

func (store *MySQLStore) GetMonthCard(userID string) (*model.MonthCard, error) {
	return nil, fmt.Errorf("mysql monthcard not implemented, use memory store")
}

func (store *MySQLStore) BuyMonthCard(userID string, cardType model.MonthCardType, pointsCost int64) (*model.MonthCard, error) {
	return nil, fmt.Errorf("mysql monthcard not implemented, use memory store")
}

func (store *MySQLStore) UseFreeDraw(userID string) (int, error) {
	return 0, fmt.Errorf("mysql monthcard not implemented, use memory store")
}

func (store *MySQLStore) GetTodayFreeDrawUsed(userID string) (int, error) {
	return 0, fmt.Errorf("mysql monthcard not implemented, use memory store")
}

// ---- 用户卡（月卡/周卡/季卡）stubs ----

func (store *MySQLStore) GetUserCard(userID string) (*model.UserCard, error) {
	return nil, nil
}

func (store *MySQLStore) BuyCard(userID string, cardType model.CardType) (*model.BuyCardResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, ok := model.CardConfigs[cardType]
	if !ok {
		return nil, fmt.Errorf("unknown card type: %s", cardType)
	}
	cost := int64(cfg.Price)

	// Get or create user member and check points
	var member model.UserMember
	err := store.db.QueryRowContext(ctx, "SELECT user_id, level, points FROM user_members WHERE user_id = ?", userID).Scan(&member.UserID, &member.Level, &member.Points)
	if err == sql.ErrNoRows {
		return nil, ErrInsufficientPoints
	}
	if err != nil {
		return nil, err
	}
	if member.Points < cost {
		return nil, ErrInsufficientPoints
	}

	// Deduct points
	member.Points -= cost
	_, err = store.db.ExecContext(ctx, "UPDATE user_members SET points = ?, updated_at = UTC_TIMESTAMP() WHERE user_id = ?", member.Points, userID)
	if err != nil {
		return nil, err
	}

	// Log the purchase
	_, err = store.db.ExecContext(ctx, "INSERT INTO user_points_logs (user_id, points, balance, reason, remark, created_at) VALUES (?, ?, ?, 'buy_card', ?, UTC_TIMESTAMP())",
		userID, -cost, member.Points, "购买"+cfg.Description)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	expiresAt := now.AddDate(0, 0, cfg.DurationDays)

	return &model.BuyCardResult{
		CardType:  cardType,
		ExpiresAt: expiresAt.Format("2006-01-02"),
		Price:     cfg.Price,
		Points:    member.Points,
	}, nil
}

func (store *MySQLStore) ConsumeFreeDraw(userID string) (bool, error) {
	return false, nil
}

func (store *MySQLStore) GetFreeDrawRemaining(userID string) (int, error) {
	return 0, nil
}

// ---- 战令系统 stubs ----

func (store *MySQLStore) GetActiveSeason() (*model.BattlePassSeason, error) {
	return nil, fmt.Errorf("mysql battlepass not implemented, use memory store")
}

func (store *MySQLStore) GetUserBattlePass(userID string, seasonID int) (*model.BattlePass, error) {
	return nil, fmt.Errorf("mysql battlepass not implemented, use memory store")
}

func (store *MySQLStore) BuyBattlePass(userID string, seasonID int, pointsCost int64) (*model.BattlePass, error) {
	return nil, fmt.Errorf("mysql battlepass not implemented, use memory store")
}

func (store *MySQLStore) AddBattlePassXP(userID string, seasonID int, xp int) (*model.BattlePass, error) {
	return nil, fmt.Errorf("mysql battlepass not implemented, use memory store")
}

func (store *MySQLStore) ClaimBattlePassReward(userID string, seasonID int, level int) (bool, error) {
	return false, fmt.Errorf("mysql battlepass not implemented, use memory store")
}

func (store *MySQLStore) GetBattlePassTasks(seasonID int) ([]model.BattlePassTask, error) {
	return nil, fmt.Errorf("mysql battlepass not implemented, use memory store")
}

func (store *MySQLStore) GetBattlePassTaskProgress(userID string, seasonID int) ([]model.BattlePassTaskProgress, error) {
	return nil, fmt.Errorf("mysql battlepass not implemented, use memory store")
}

func (store *MySQLStore) UpdateTaskProgress(userID string, taskID int) error {
	return fmt.Errorf("mysql battlepass not implemented, use memory store")
}

func (store *MySQLStore) GetBattlePassRewards(seasonID int) ([]model.BattlePassReward, error) {
	return nil, fmt.Errorf("mysql battlepass not implemented, use memory store")
}

// ---- 商店 + 道具 stubs ----

func (store *MySQLStore) GetShopItems() []model.ShopItem {
	return nil
}

func (store *MySQLStore) BuyShopItem(userID string, itemID string, quantity int) (*model.BuyShopItemResult, error) {
	return nil, fmt.Errorf("mysql shop not implemented, use memory store")
}





// ---- 首充礼包 stubs ----



// ---- 社交裂变 stubs ----

func (store *MySQLStore) GenerateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (store *MySQLStore) CreateInviteRecord(inviterID, inviteeID string) (*model.InviteRecord, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetInviteRecords(userID string) ([]model.InviteRecord, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetInviteStats(userID string) (*model.InviteStats, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetOrCreateAssistProgress(inviterID string, assistType model.AssistType) (*model.AssistProgress, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) IsAssistActionRecorded(inviterID, helperID string, assistType model.AssistType) (bool, error) {
	return false, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) RecordAssistAction(inviterID, helperID string, assistType model.AssistType) error {
	return fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) IncrementAssistProgress(inviterID string, assistType model.AssistType) (*model.AssistProgress, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) ClaimAssistReward(inviterID string, assistType model.AssistType) (*model.AssistProgress, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetAssistProgress(inviterID string, assistType model.AssistType) (*model.AssistProgress, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

// ---- 社交裂变：组队开盒 stubs ----

func (store *MySQLStore) CreateTeam(captainID string, input model.CreateTeamRequest) (*model.Team, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) JoinTeam(userID string, teamID string) (*model.TeamMember, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) LeaveTeam(userID, teamID string) error {
	return fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetTeam(teamID string) (*model.Team, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetTeamMembers(teamID string) ([]model.TeamMember, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetUserActiveTeam(userID string) (*model.Team, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) AddTeamDraw(userID, teamID string) (int, error) {
	return 0, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) CompleteTeam(teamID string) (*model.TeamReward, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) ExpireTeam(teamID string) error {
	return fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetExpiredTeams() ([]model.Team, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

// ---- 社交裂变：礼物赠送 stubs ----

func (store *MySQLStore) CreateGift(giverID, receiverID, prizeID, campaignID string) (*model.GiftRecord, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetGift(giftID string) (*model.GiftRecord, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) ReceiveGift(giftID string) (*model.ReceiveGiftResult, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetUserGifts(userID string) ([]model.GiftRecord, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetUserSentGifts(userID string) ([]model.GiftRecord, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) ExpireGift(giftID string) error {
	return fmt.Errorf("mysql social not implemented, use memory store")
}

// ---- 社交裂变：分享卡片 stubs ----

func (store *MySQLStore) CreateShareCard(userID string, cardType string, title, description string, prizeName, prizeLevel, inviteLink string) (*model.ShareCard, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetShareCards(userID string) ([]model.ShareCard, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

// ---- 碎片拼图 stubs ----

func (store *MySQLStore) GetActivePuzzleTemplates() []model.PuzzleTemplate {
	return nil
}

func (store *MySQLStore) GetPuzzleTemplate(templateID string) (*model.PuzzleTemplate, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) GetOrCreatePuzzleProgress(userID, templateID string) (*model.PuzzleProgress, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) AddPuzzlePiece(userID, templateID string, pieceIndex int) (bool, error) {
	return false, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) ComposePuzzle(userID, templateID string) (*model.ComposePuzzleResult, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) GetPuzzleInfo(userID, templateID string) (*model.PuzzleInfo, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) CreatePuzzleTeam(captainID, templateID string) (*model.PuzzleTeam, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) JoinPuzzleTeam(userID, teamID string) (*model.PuzzleTeam, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) GetPuzzleTeam(teamID string) (*model.PuzzleTeam, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) GetUserPuzzleTeams(userID string) ([]model.PuzzleTeam, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) GetUserPuzzleProgresses(userID string) ([]model.PuzzleInfo, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) SharePuzzlePiece(userID, teamID string, pieceIndex int) (bool, error) {
	return false, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

// ---- 抢购/预约 stubs ----

func (store *MySQLStore) GetFlashSales() []model.FlashSale {
	return nil
}

func (store *MySQLStore) GetFlashSale(flashID string) (*model.FlashSale, error) {
	return nil, fmt.Errorf("mysql flash not implemented, use memory store")
}

func (store *MySQLStore) SubscribeFlash(userID, flashID string) error {
	return fmt.Errorf("mysql flash not implemented, use memory store")
}

func (store *MySQLStore) UnsubscribeFlash(userID, flashID string) error {
	return fmt.Errorf("mysql flash not implemented, use memory store")
}

func (store *MySQLStore) IsFlashSubscribed(userID, flashID string) (bool, error) {
	return false, fmt.Errorf("mysql flash not implemented, use memory store")
}

func (store *MySQLStore) PurchaseFlash(userID, flashID string) (*model.FlashPurchaseResult, error) {
	return nil, fmt.Errorf("mysql flash not implemented, use memory store")
}

func (store *MySQLStore) GetUserFlashSubscriptions(userID string) ([]model.FlashSubscription, error) {
	return nil, fmt.Errorf("mysql flash not implemented, use memory store")
}

func (store *MySQLStore) CreateFlashSale(input model.FlashSale) (*model.FlashSale, error) {
	return nil, fmt.Errorf("mysql flash not implemented, use memory store")
}

func (store *MySQLStore) UpdateFlashSaleStatus(flashID, status string) error {
	return fmt.Errorf("mysql flash not implemented, use memory store")
}


// ============================================================
// 道具系统 stubs
// ============================================================

func (store *MySQLStore) GetUserItemQty(userID string, itemType model.ItemType) (int, error) {
	return 0, nil
}

func (store *MySQLStore) AddUserItem(userID string, itemType model.ItemType, qty int) error {
	return nil
}

func (store *MySQLStore) UseUserItem(userID string, itemType model.ItemType, qty int) (bool, error) {
	return false, nil
}

func (store *MySQLStore) GetUserItems(userID string) ([]model.UserItem, error) {
	return nil, nil
}

// ---- 首充礼包 stubs ----

func (store *MySQLStore) GetFirstRechargeStatus(userID string) (*model.UserFirstRecharge, error) {
	return &model.UserFirstRecharge{UserID: userID, Claimed: []string{}}, nil
}

func (store *MySQLStore) ClaimFirstRecharge(userID string, packID string) (*model.ClaimFirstRechargeResult, error) {
	return nil, fmt.Errorf("mysql first recharge not implemented, use memory store")
}
