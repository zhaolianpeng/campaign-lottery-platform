package store

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	mathrand "math/rand/v2"
	"sort"
	"strings"
	"sync"
	"time"

	"campaign-lottery-platform/backend/internal/model"
)

type MemoryStore struct {
	mu             sync.RWMutex
	users          map[string]model.User
	sessions       map[string]model.Session
	adminSessions  map[string]time.Time
	campaigns      map[string]model.Campaign
	prizes         map[string][]model.Prize
	drawRecords    []model.DrawRecord
	userDrawCounts map[string]int
	adminUser      string
	adminPassword  string

	// 盲盒扩展
	inventory       []model.UserInventory      // 用户库存
	exchangeOffers  []model.ExchangeOffer      // 交换市场挂单
	members         map[string]model.UserMember // 用户会员信息
	pointsLog       []model.UserPointsLog      // 积分记录
	nextTaskID      int64                      // 自增任务ID

	// 集卡系统扩展
	checkInDates   map[string]time.Time // userID -> last check-in date
	checkInStreaks map[string]int       // userID -> streak days
	shareCounts    map[string]int       // userID -> today's share count
}

func NewMemoryStore(adminUser string, adminPassword string) *MemoryStore {
	store := &MemoryStore{
		users:          make(map[string]model.User),
		sessions:       make(map[string]model.Session),
		adminSessions:  make(map[string]time.Time),
		campaigns:      make(map[string]model.Campaign),
		prizes:         make(map[string][]model.Prize),
		drawRecords:    make([]model.DrawRecord, 0, 16),
		userDrawCounts: make(map[string]int),
		adminUser:      adminUser,
		adminPassword:  adminPassword,
		inventory:      make([]model.UserInventory, 0, 32),
		exchangeOffers: make([]model.ExchangeOffer, 0, 8),
		members:        make(map[string]model.UserMember),
		pointsLog:      make([]model.UserPointsLog, 0, 16),
		nextTaskID:     1,
		checkInDates:   make(map[string]time.Time),
		checkInStreaks: make(map[string]int),
		shareCounts:    make(map[string]int),
	}

	store.seedDefaultCampaign()
	return store
}

func (s *MemoryStore) Seed() error { return nil }

func (s *MemoryStore) seedDefaultCampaign() {
	now := time.Now().UTC()
	campaign := model.Campaign{
		ID:              "camp_launch_001",
		Name:            "夏季开门红抽奖活动",
		Slug:            "summer-launch",
		Status:          "online",
		StartsAt:        now.Add(-24 * time.Hour),
		EndsAt:          now.Add(30 * 24 * time.Hour),
		DailyDrawLimit:  3,
		MissWeight:      86,
		BannerImageURL:  "https://static.example.com/campaign/summer-launch/banner.png",
		CampaignSummary: "新用户登录即可参与，中奖后进入发奖队列，支持后台配置库存和概率。",
	}
	s.campaigns[campaign.ID] = campaign
	s.prizes[campaign.ID] = []model.Prize{
		{ID: "prize_001", CampaignID: campaign.ID, Name: "88元红包", Level: "S", Stock: 8, ProbabilityWeight: 2, Status: "active"},
		{ID: "prize_002", CampaignID: campaign.ID, Name: "20元优惠券", Level: "A", Stock: 60, ProbabilityWeight: 18, Status: "active"},
		{ID: "prize_003", CampaignID: campaign.ID, Name: "品牌周边礼盒", Level: "B", Stock: 20, ProbabilityWeight: 8, Status: "active"},
	}

	// 盲盒系列1: 星空系列
	series := model.Campaign{
		ID:              "series_starry_001",
		Name:            "🌙 星空系列",
		Slug:            "starry-night",
		Status:          "online",
		StartsAt:        now.Add(-24 * time.Hour),
		EndsAt:          now.Add(60 * 24 * time.Hour),
		DailyDrawLimit:  10,
		MissWeight:      72,
		BannerImageURL:  "https://static.example.com/series/starry-night/banner.png",
		CampaignSummary: "收集星光、月色与银河，集齐6款普通+1款隐藏可解锁限定奖励！",
	}
	s.campaigns[series.ID] = series
	s.prizes[series.ID] = []model.Prize{
		// 普通款（权重累加=72）
		{ID: "star_01", CampaignID: series.ID, Name: "繁星点点", Level: "common", Stock: 500, ProbabilityWeight: 15, Status: "active"},
		{ID: "star_02", CampaignID: series.ID, Name: "月光如水", Level: "common", Stock: 500, ProbabilityWeight: 15, Status: "active"},
		{ID: "star_03", CampaignID: series.ID, Name: "银河之泪", Level: "common", Stock: 400, ProbabilityWeight: 14, Status: "active"},
		{ID: "star_04", CampaignID: series.ID, Name: "流星划过", Level: "common", Stock: 400, ProbabilityWeight: 12, Status: "active"},
		{ID: "star_05", CampaignID: series.ID, Name: "极光之舞", Level: "common", Stock: 300, ProbabilityWeight: 10, Status: "active"},
		{ID: "star_06", CampaignID: series.ID, Name: "星云之眼", Level: "common", Stock: 300, ProbabilityWeight: 6, Status: "active"},
		// 稀有款
		{ID: "star_07", CampaignID: series.ID, Name: "星月传说", Level: "rare", Stock: 100, ProbabilityWeight: 15, Status: "active"},
		{ID: "star_08", CampaignID: series.ID, Name: "北极光", Level: "rare", Stock: 80, ProbabilityWeight: 10, Status: "active"},
		// 隐藏款
		{ID: "star_09", CampaignID: series.ID, Name: "宇宙之心 ★", Level: "secret", Stock: 10, ProbabilityWeight: 2, Status: "active"},
		// 限定款
		{ID: "star_10", CampaignID: series.ID, Name: "星辰大海 ★★", Level: "limited", Stock: 3, ProbabilityWeight: 1, Status: "active"},
	}

	// 盲盒系列2: 猫咪系列
	cat := model.Campaign{
		ID:              "series_cat_001",
		Name:            "🐱 猫咪系列",
		Slug:            "cute-cats",
		Status:          "online",
		StartsAt:        now.Add(-12 * time.Hour),
		EndsAt:          now.Add(45 * 24 * time.Hour),
		DailyDrawLimit:  8,
		MissWeight:      68,
		BannerImageURL:  "https://static.example.com/series/cute-cats/banner.png",
		CampaignSummary: "超萌猫咪盲盒！集齐全部6款可以解锁隐藏版「布偶猫王」！",
	}
	s.campaigns[cat.ID] = cat
	s.prizes[cat.ID] = []model.Prize{
		{ID: "cat_01", CampaignID: cat.ID, Name: "英短蓝猫", Level: "common", Stock: 600, ProbabilityWeight: 16, Status: "active"},
		{ID: "cat_02", CampaignID: cat.ID, Name: "橘猫胖胖", Level: "common", Stock: 600, ProbabilityWeight: 16, Status: "active"},
		{ID: "cat_03", CampaignID: cat.ID, Name: "黑猫酷酷", Level: "common", Stock: 500, ProbabilityWeight: 14, Status: "active"},
		{ID: "cat_04", CampaignID: cat.ID, Name: "三花猫", Level: "common", Stock: 500, ProbabilityWeight: 12, Status: "active"},
		{ID: "cat_05", CampaignID: cat.ID, Name: "暹罗猫", Level: "common", Stock: 400, ProbabilityWeight: 10, Status: "active"},
		{ID: "cat_06", CampaignID: cat.ID, Name: "俄罗斯蓝猫", Level: "rare", Stock: 120, ProbabilityWeight: 18, Status: "active"},
		{ID: "cat_07", CampaignID: cat.ID, Name: "布偶猫王 ★", Level: "secret", Stock: 8, ProbabilityWeight: 2, Status: "active"},
	}
}

// ============================================================
// 基础 CRUD（保留原有实现）
// ============================================================

func (s *MemoryStore) CreateGuestSession(nickname string) (model.User, model.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(nickname) == "" {
		nickname = "Guest" + randomSuffix(4)
	}

	user := model.User{
		ID:        "usr_" + randomSuffix(12),
		Nickname:  nickname,
		CreatedAt: time.Now().UTC(),
	}
	s.users[user.ID] = user

	session := model.Session{
		Token:     "utk_" + randomSuffix(24),
		UserID:    user.ID,
		ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
	}
	s.sessions[session.Token] = session
	return user, session, nil
}

func (s *MemoryStore) Campaigns() []model.Campaign {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.Campaign, 0, len(s.campaigns))
	for _, campaign := range s.campaigns {
		items = append(items, campaign)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].StartsAt.Before(items[j].StartsAt)
	})
	return items
}

func (s *MemoryStore) GetCampaign(campaignID string) (model.Campaign, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.campaigns[campaignID]
	if !ok {
		return model.Campaign{}, ErrCampaignNotFound
	}
	return c, nil
}

func (s *MemoryStore) PrizeList(campaignID string) []model.Prize {
	s.mu.RLock()
	defer s.mu.RUnlock()
	prizes := s.prizes[campaignID]
	items := make([]model.Prize, len(prizes))
	copy(items, prizes)
	return items
}

func (s *MemoryStore) UserFromToken(token string) (model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[token]
	if !ok || session.ExpiresAt.Before(time.Now().UTC()) {
		return model.User{}, ErrUnauthorized
	}
	user, ok := s.users[session.UserID]
	if !ok {
		return model.User{}, ErrUnauthorized
	}
	return user, nil
}

// ============================================================
// 盲盒抽奖
// ============================================================

func (s *MemoryStore) CreateDrawRecord(userID, campaignID, prizeID string, _ bool) (model.DrawRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 找到奖品并扣库存
	prizeName := "未知奖品"
	prizes := s.prizes[campaignID]
	found := false
	for i := range prizes {
		if prizes[i].ID == prizeID && prizes[i].Stock > 0 {
			prizes[i].Stock--
			prizeName = prizes[i].Name
			found = true
			break
		}
	}
	s.prizes[campaignID] = prizes
	if !found {
		// 库存不足，记录未中奖
		return s.createMissRecordLocked(userID, campaignID)
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
	s.drawRecords = append([]model.DrawRecord{record}, s.drawRecords...)

	// 添加到用户库存
	id := prizeID
	s.inventory = append(s.inventory, model.UserInventory{
		ID:         "inv_" + randomSuffix(12),
		UserID:     userID,
		PrizeID:    prizeID,
		PrizeName:  prizeName,
		Source:     "draw",
		CreatedAt:  now,
	})

	return record, nil
}

func (s *MemoryStore) createMissRecordLocked(userID, campaignID string) (model.DrawRecord, error) {
	now := time.Now().UTC()
	record := model.DrawRecord{
		ID:         "draw_" + randomSuffix(12),
		CampaignID: campaignID,
		UserID:     userID,
		PrizeName:  "未中奖",
		Result:     "miss",
		DrawnAt:    now,
	}
	s.drawRecords = append([]model.DrawRecord{record}, s.drawRecords...)
	return record, nil
}

func (s *MemoryStore) CreateMissRecord(userID, campaignID string, _ bool) (model.DrawRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createMissRecordLocked(userID, campaignID)
}

func (s *MemoryStore) CheckDrawQuota(userID, campaignID string, dailyLimit int) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	used := s.userDrawCounts[userID+":"+campaignID]
	remaining := dailyLimit - used
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

func (s *MemoryStore) DeductDrawQuota(userID, campaignID string, count int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userID + ":" + campaignID
	s.userDrawCounts[key] += count
	// Campaign 的 DailyDrawLimit 需要从外部传入，这里只能返回增量后的值
	return 99 - s.userDrawCounts[key], nil
}

// ============================================================
// 原 Draw 兼容（由 service 接管逻辑后，这里保留桩件）
// ============================================================

func (s *MemoryStore) Draw(token string, campaignID string) (model.DrawResult, error) {
	return model.DrawResult{}, errors.New("use BlindBoxDraw via Service instead")
}

func (s *MemoryStore) UserDrawRecords(token string) ([]model.DrawRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[token]
	if !ok || session.ExpiresAt.Before(time.Now().UTC()) {
		return nil, ErrUnauthorized
	}

	items := make([]model.DrawRecord, 0, 8)
	for _, record := range s.drawRecords {
		if record.UserID == session.UserID {
			items = append(items, record)
		}
	}
	return items, nil
}

// ============================================================
// 库存 / 收集进度
// ============================================================

func (s *MemoryStore) GetUserInventory(userID string) ([]model.UserInventory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.UserInventory, 0, 8)
	for _, inv := range s.inventory {
		if inv.UserID == userID {
			items = append(items, inv)
		}
	}
	return items, nil
}

func (s *MemoryStore) GetSeriesProgress(userID, campaignID, campaignName string) (*model.SeriesProgress, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prizes := s.prizes[campaignID]
	totalItems := len(prizes)
	collectedMap := make(map[string]int)

	for _, inv := range s.inventory {
		if inv.UserID == userID && inv.CampaignID == campaignID {
			collectedMap[inv.PrizeID]++
		}
	}

	collected := make([]model.CollectedPrize, 0, len(collectedMap))
	missing := make([]model.PrizeSummary, 0)
	duplicates := 0

	for _, p := range prizes {
		if count, ok := collectedMap[p.ID]; ok {
			collected = append(collected, model.CollectedPrize{Prize: p, Count: count})
			if count > 1 {
				duplicates += count - 1
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
		CampaignID:      campaignID,
		CampaignName:    campaignName,
		TotalItems:      totalItems,
		CollectedItems:  len(collected),
		ProgressPercent: pct,
		Duplicates:      duplicates,
		CollectedPrizes: collected,
		MissingPrizes:   missing,
	}, nil
}

func (s *MemoryStore) UserHasPrize(userID, prizeID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, inv := range s.inventory {
		if inv.UserID == userID && inv.PrizeID == prizeID {
			return true, nil
		}
	}
	return false, nil
}

// ============================================================
// 交换市场
// ============================================================

func (s *MemoryStore) ExchangeOffers() []model.ExchangeOffer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.ExchangeOffer, len(s.exchangeOffers))
	copy(items, s.exchangeOffers)
	return items
}

func (s *MemoryStore) CreateExchangeOffer(userID string, input model.ExchangeOfferMutation) (model.ExchangeOffer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	offer := model.ExchangeOffer{
		ID:           "ex_offer_" + randomSuffix(10),
		UserID:       userID,
		HavePrizeID:  input.HavePrizeID,
		WantPrizeID:  input.WantPrizeID,
		Status:       "pending",
		CreatedAt:    time.Now().UTC(),
	}
	if u, ok := s.users[userID]; ok {
		offer.UserNickname = u.Nickname
	}
	// 填充名称
	for _, inv := range s.inventory {
		if inv.PrizeID == input.HavePrizeID {
			offer.HavePrizeName = inv.PrizeName
		}
	}
	for _, prizes := range s.prizes {
		for _, p := range prizes {
			if p.ID == input.WantPrizeID {
				offer.WantPrizeName = p.Name
			}
		}
	}

	s.exchangeOffers = append([]model.ExchangeOffer{offer}, s.exchangeOffers...)
	return offer, nil
}

func (s *MemoryStore) CancelExchangeOffer(userID, offerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.exchangeOffers {
		if s.exchangeOffers[i].ID == offerID {
			if s.exchangeOffers[i].UserID != userID {
				return errors.New("not your offer")
			}
			s.exchangeOffers[i].Status = "cancelled"
			return nil
		}
	}
	return errors.New("offer not found")
}

func (s *MemoryStore) AcceptExchangeOffer(userID, offerID string) (model.ExchangeOffer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.exchangeOffers {
		if s.exchangeOffers[i].ID == offerID {
			if s.exchangeOffers[i].UserID == userID {
				return model.ExchangeOffer{}, errors.New("cannot accept your own offer")
			}
			if s.exchangeOffers[i].Status != "pending" {
				return model.ExchangeOffer{}, errors.New("offer is no longer pending")
			}
			s.exchangeOffers[i].Status = "matched"
			return s.exchangeOffers[i], nil
		}
	}
	return model.ExchangeOffer{}, errors.New("offer not found")
}

// ============================================================
// 积分/会员
// ============================================================

func (s *MemoryStore) GetUserMember(userID string) (*model.UserMember, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if m, ok := s.members[userID]; ok {
		return &m, nil
	}
	return &model.UserMember{UserID: userID, Level: model.MemberNormal}, nil
}

func (s *MemoryStore) GetPointsLog(userID string) ([]model.UserPointsLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.UserPointsLog, 0, 8)
	for _, log := range s.pointsLog {
		if log.UserID == userID {
			items = append(items, log)
		}
	}
	return items, nil
}

func (s *MemoryStore) UpdateUserMember(member *model.UserMember) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.members[member.UserID] = *member
	return nil
}

func (s *MemoryStore) RedeemPrize(userID string, input model.RedeemRequest) (*model.RedeemResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 找奖品
	var prizeName, prizeLevel string
	var found bool
	for _, prizes := range s.prizes {
		for _, p := range prizes {
			if p.ID == input.PrizeID && p.Status == "active" {
				prizeName = p.Name
				prizeLevel = p.Level
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("prize not found")
	}

	pointsCost := map[string]int64{
		"common": 100, "rare": 500, "secret": 2000, "limited": 5000,
	}[prizeLevel]
	if pointsCost == 0 {
		pointsCost = 100
	}

	member, ok := s.members[userID]
	if !ok {
		member = model.UserMember{UserID: userID, Level: model.MemberNormal, Points: 0}
	}
	if member.Points < pointsCost {
		return nil, fmt.Errorf("insufficient points: have %d, need %d", member.Points, pointsCost)
	}

	member.Points -= pointsCost
	s.members[userID] = member

	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID:        int64(len(s.pointsLog) + 1),
		UserID:    userID,
		Points:    -pointsCost,
		Balance:   member.Points,
		Reason:    "redeem",
		Remark:    "兑换: " + prizeName,
		CreatedAt: time.Now().UTC(),
	})

	// 加库存
	prizeCampaignID := ""
	for cid, prizes := range s.prizes {
		for _, p := range prizes {
			if p.ID == input.PrizeID {
				prizeCampaignID = cid
				break
			}
		}
		if prizeCampaignID != "" {
			break
		}
	}
	now := time.Now().UTC()
	s.inventory = append(s.inventory, model.UserInventory{
		ID:         "inv_" + randomSuffix(12),
		UserID:     userID,
		PrizeID:    input.PrizeID,
		PrizeName:  prizeName,
		PrizeLevel: prizeLevel,
		CampaignID: prizeCampaignID,
		Source:     "redeem",
		CreatedAt:  now,
	})

	return &model.RedeemResult{
		RecordID:   "rdm_" + randomSuffix(12),
		PrizeID:    input.PrizeID,
		PrizeName:  prizeName,
		PointsCost: pointsCost,
		Remaining:  member.Points,
	}, nil
}

// ============================================================
// 集卡系统扩展
// ============================================================

// DailyCheckIn 每日签到
func (s *MemoryStore) DailyCheckIn(userID string, points int64) (*model.CheckInResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	member, ok := s.members[userID]
	if !ok {
		member = model.UserMember{UserID: userID, Level: model.MemberNormal, Points: 0}
	}

	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)

	// 计算连续签到天数
	lastDate, hasLast := s.checkInDates[userID]
	streak := s.checkInStreaks[userID]
	if hasLast {
		lastDay := lastDate.Truncate(24 * time.Hour)
		diff := today.Sub(lastDay)
		if diff == 24*time.Hour {
			// 连续签到
			streak++
		} else if diff > 24*time.Hour {
			// 中断，重置
			streak = 1
		}
		// diff == 0 表示今天已签过到，保持 streak 不变
	} else {
		streak = 1
	}

	// 更新签到日期
	s.checkInDates[userID] = now
	s.checkInStreaks[userID] = streak

	// 加积分
	totalPoints := points
	isBonus := false
	if streak == 7 {
		totalPoints += 20
		isBonus = true
		// 重置连续天数，重新计数
		s.checkInStreaks[userID] = 0
	}

	member.Points += totalPoints
	s.members[userID] = member

	// 记录积分日志
	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID:        int64(len(s.pointsLog) + 1),
		UserID:    userID,
		Points:    totalPoints,
		Balance:   member.Points,
		Reason:    "daily",
		Remark:    "每日签到",
		CreatedAt: now,
	})

	return &model.CheckInResult{
		PointsAwarded: totalPoints,
		StreakDays:    streak,
		IsBonus:       isBonus,
		NewBalance:    member.Points,
	}, nil
}

// GetCheckInStreak 获取连续签到天数
func (s *MemoryStore) GetCheckInStreak(userID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.checkInStreaks[userID], nil
}

// CheckCollectionCompletion 检查用户是否集齐系列所有普通+稀有款式
func (s *MemoryStore) CheckCollectionCompletion(userID, campaignID string) (*model.CollectionReward, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prizes := s.prizes[campaignID]
	if len(prizes) == 0 {
		return nil, nil
	}

	// 筛选需要集齐的款式（排除 secret 和 limited）
	required := make([]model.Prize, 0, len(prizes))
	campaignName := ""
	for _, p := range prizes {
		if p.Level == "secret" || p.Level == "limited" {
			continue
		}
		if p.Status != "active" {
			continue
		}
		required = append(required, p)
		campaignName = p.CampaignID // fallback name source
	}

	if len(required) == 0 {
		return nil, nil
	}

	// 查找 campaign name
	if c, ok := s.campaigns[campaignID]; ok {
		campaignName = c.Name
	}

	// 构建用户已拥有的奖品集合
	owned := make(map[string]bool)
	for _, inv := range s.inventory {
		if inv.UserID == userID && inv.CampaignID == campaignID {
			owned[inv.PrizeID] = true
		}
	}

	// 检查是否拥有所有必需的款式
	for _, p := range required {
		if !owned[p.ID] {
			return nil, nil
		}
	}

	// 集齐了
	return &model.CollectionReward{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		RewardType:   "title",
		RewardName:   "收集大师",
		Description:  "恭喜集齐",
	}, nil
}

// GrantCollectionReward 发放集齐奖励
func (s *MemoryStore) GrantCollectionReward(userID string, reward *model.CollectionReward) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	// 添加库存记录
	s.inventory = append(s.inventory, model.UserInventory{
		ID:         "inv_" + randomSuffix(12),
		UserID:     userID,
		PrizeID:    reward.CampaignID,
		PrizeName:  reward.RewardName,
		CampaignID: reward.CampaignID,
		Source:     "collection_reward",
		CreatedAt:  now,
	})

	// 加积分
	member, ok := s.members[userID]
	if !ok {
		member = model.UserMember{UserID: userID, Level: model.MemberNormal, Points: 0}
	}
	member.Points += 500
	s.members[userID] = member

	// 记录积分日志
	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID:        int64(len(s.pointsLog) + 1),
		UserID:    userID,
		Points:    500,
		Balance:   member.Points,
		Reason:    "collection_reward",
		Remark:    "集齐系列奖励: " + reward.RewardName,
		CreatedAt: now,
	})

	return nil
}

// GetLeaderboard 获取收集排行榜
func (s *MemoryStore) GetLeaderboard(limit int) ([]model.LeaderboardEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.users) == 0 {
		return []model.LeaderboardEntry{}, nil
	}

	// 收集每个系列的总款式数（排除 secret 和 limited）
	seriesTotal := make(map[string]int)
	for cid, prizes := range s.prizes {
		count := 0
		for _, p := range prizes {
			if p.Level != "secret" && p.Level != "limited" && p.Status == "active" {
				count++
			}
		}
		if count > 0 {
			seriesTotal[cid] = count
		}
	}

	// 统计每个用户在各系列的收集进度
	type userProgress struct {
		collectedCount int
		seriesCount    int // 已集齐的系列数
		seriesTotal    int // 总系列款式数（最大）
	}
	userMap := make(map[string]*userProgress)

	for _, inv := range s.inventory {
		if inv.CampaignID == "" {
			continue
		}
		// 只统计非 secret/limited 的奖品
		total, ok := seriesTotal[inv.CampaignID]
		if !ok {
			continue
		}
		if _, exists := userMap[inv.UserID]; !exists {
			userMap[inv.UserID] = &userProgress{}
		}
		userMap[inv.UserID].seriesTotal = total
	}

	// 按用户+系列去重统计 collectedCount
	type userCampaignKey struct {
		userID     string
		campaignID string
	}
	seen := make(map[userCampaignKey]map[string]bool)

	for _, inv := range s.inventory {
		if _, ok := seriesTotal[inv.CampaignID]; !ok {
			continue
		}
		key := userCampaignKey{userID: inv.UserID, campaignID: inv.CampaignID}
		if seen[key] == nil {
			seen[key] = make(map[string]bool)
		}
		seen[key][inv.PrizeID] = true
	}

	for key, prizes := range seen {
		p := userMap[key.userID]
		if p == nil {
			continue
		}
		p.collectedCount += len(prizes)
		if len(prizes) >= seriesTotal[key.campaignID] {
			p.seriesCount++
		}
	}

	// 构建排行榜条目
	entries := make([]model.LeaderboardEntry, 0, len(userMap))
	for uid, progress := range userMap {
		nickname := ""
		if u, ok := s.users[uid]; ok {
			nickname = u.Nickname
		}
		totalCount := progress.seriesTotal
		pct := 0.0
		if totalCount > 0 {
			pct = float64(progress.collectedCount) / float64(totalCount) * 100
		}
		entries = append(entries, model.LeaderboardEntry{
			UserID:          uid,
			Nickname:        nickname,
			CollectedCount:  progress.collectedCount,
			TotalCount:      totalCount,
			ProgressPercent: pct,
			SeriesCompleted: progress.seriesCount,
		})
	}

	// 按 collectedCount 降序排序
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].CollectedCount != entries[j].CollectedCount {
			return entries[i].CollectedCount > entries[j].CollectedCount
		}
		return entries[i].UserID < entries[j].UserID
	})

	// 取前 limit 名并填充排名
	if limit > len(entries) {
		limit = len(entries)
	}
	result := entries[:limit]
	for i := range result {
		result[i].Rank = i + 1
	}

	return result, nil
}

// GetCampaignHint 获取系列摇盒提示文案
func (s *MemoryStore) GetCampaignHint(campaignID string) *model.HintMessage {
	hints := []model.HintMessage{
		{Type: "hot", Content: "据大数据分析，本系列当前隐藏款热度较高"},
		{Type: "social", Content: "已有 xx 位用户抽到此系列的隐藏款"},
		{Type: "luck", Content: "刚刚有用户十连抽中了稀有款！"},
	}
	idx := mathrand.N(len(hints))
	return &hints[idx]
}

// ShareReward 分享奖励
func (s *MemoryStore) ShareReward(userID string, points int64) (*model.ShareRewardResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查当日分享次数
	count := s.shareCounts[userID]
	if count >= 10 {
		return nil, fmt.Errorf("今日分享次数已达上限")
	}

	// 增加分享次数
	s.shareCounts[userID] = count + 1

	// 加积分
	member, ok := s.members[userID]
	if !ok {
		member = model.UserMember{UserID: userID, Level: model.MemberNormal, Points: 0}
	}
	member.Points += points
	s.members[userID] = member

	// 记录积分日志
	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID:        int64(len(s.pointsLog) + 1),
		UserID:    userID,
		Points:    points,
		Balance:   member.Points,
		Reason:    "share",
		Remark:    "分享奖励",
		CreatedAt: time.Now().UTC(),
	})

	return &model.ShareRewardResult{
		PointsAwarded: points,
		DailyLeft:     10 - s.shareCounts[userID],
		NewBalance:    member.Points,
	}, nil
}

// GetShareDailyCount 获取今日已分享次数
func (s *MemoryStore) GetShareDailyCount(userID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.shareCounts[userID], nil
}

// GetPrizeCount 获取用户某系列某款式的数量
func (s *MemoryStore) GetPrizeCount(userID, prizeID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, inv := range s.inventory {
		if inv.UserID == userID && inv.PrizeID == prizeID {
			count++
		}
	}
	return count, nil
}

// ============================================================
// 管理员
// ============================================================

func (s *MemoryStore) AdminLogin(username string, password string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if username != s.adminUser || password != s.adminPassword {
		return "", ErrBadAdminAuth
	}
	token := "atk_" + randomSuffix(24)
	s.adminSessions[token] = time.Now().UTC().Add(12 * time.Hour)
	return token, nil
}

func (s *MemoryStore) AdminOverview(token string) (model.AdminOverview, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.ensureAdmin(token); err != nil {
		return model.AdminOverview{}, err
	}

	prizeSummaries := make([]model.PrizeSummary, 0, 8)
	for _, prizes := range s.prizes {
		for _, prize := range prizes {
			prizeSummaries = append(prizeSummaries, model.PrizeSummary{
				PrizeID: prize.ID, PrizeName: prize.Name, PrizeLevel: prize.Level, Stock: prize.Stock,
			})
		}
	}

	campaigns := make([]model.Campaign, 0, len(s.campaigns))
	for _, campaign := range s.campaigns {
		campaigns = append(campaigns, campaign)
	}

	recentDraws := make([]model.DrawRecord, 0, minInt(10, len(s.drawRecords)))
	for i, record := range s.drawRecords {
		if i >= 10 {
			break
		}
		recentDraws = append(recentDraws, record)
	}

	totalWins := 0
	for _, record := range s.drawRecords {
		if record.Result == "win" {
			totalWins++
		}
	}

	balance := make(map[string]int, len(s.userDrawCounts))
	for key, used := range s.userDrawCounts {
		balance[key] = used
	}

	return model.AdminOverview{
		TotalUsers:      len(s.users),
		TotalDraws:      len(s.drawRecords),
		TotalWins:       totalWins,
		Campaigns:       campaigns,
		PrizeSummaries:  prizeSummaries,
		RecentDraws:     recentDraws,
		UserDrawBalance: balance,
	}, nil
}

func (s *MemoryStore) AdminDrawRecords(token string) ([]model.DrawRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureAdmin(token); err != nil {
		return nil, err
	}
	items := make([]model.DrawRecord, len(s.drawRecords))
	copy(items, s.drawRecords)
	return items, nil
}

func (s *MemoryStore) AdminCampaigns(token string) ([]model.Campaign, error) {
	if err := s.ensureAdmin(token); err != nil {
		return nil, err
	}
	return s.Campaigns(), nil
}

func (s *MemoryStore) CreateCampaign(token string, input model.CampaignMutation) (model.Campaign, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureAdmin(token); err != nil {
		return model.Campaign{}, err
	}
	campaign := model.Campaign{
		ID:              "camp_" + randomSuffix(10),
		Name:            input.Name,
		Slug:            input.Slug,
		Status:          input.Status,
		StartsAt:        input.StartsAt,
		EndsAt:          input.EndsAt,
		DailyDrawLimit:  input.DailyDrawLimit,
		MissWeight:      input.MissWeight,
		BannerImageURL:  input.BannerImageURL,
		CampaignSummary: input.CampaignSummary,
	}
	s.campaigns[campaign.ID] = campaign
	return campaign, nil
}

func (s *MemoryStore) UpdateCampaign(token string, campaignID string, input model.CampaignMutation) (model.Campaign, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureAdmin(token); err != nil {
		return model.Campaign{}, err
	}
	_, ok := s.campaigns[campaignID]
	if !ok {
		return model.Campaign{}, ErrCampaignNotFound
	}
	campaign := model.Campaign{
		ID: campaignID, Name: input.Name, Slug: input.Slug, Status: input.Status,
		StartsAt: input.StartsAt, EndsAt: input.EndsAt, DailyDrawLimit: input.DailyDrawLimit,
		MissWeight: input.MissWeight, BannerImageURL: input.BannerImageURL, CampaignSummary: input.CampaignSummary,
	}
	s.campaigns[campaignID] = campaign
	return campaign, nil
}

func (s *MemoryStore) DeleteCampaign(token string, campaignID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureAdmin(token); err != nil {
		return err
	}
	delete(s.campaigns, campaignID)
	delete(s.prizes, campaignID)
	return nil
}

func (s *MemoryStore) AdminPrizes(token string, campaignID string) ([]model.Prize, error) {
	if err := s.ensureAdmin(token); err != nil {
		return nil, err
	}
	return s.PrizeList(campaignID), nil
}

func (s *MemoryStore) CreatePrize(token string, campaignID string, input model.PrizeMutation) (model.Prize, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureAdmin(token); err != nil {
		return model.Prize{}, err
	}
	prize := model.Prize{
		ID: "prize_" + randomSuffix(10), CampaignID: campaignID, Name: input.Name,
		Level: input.Level, Stock: input.Stock, ProbabilityWeight: input.ProbabilityWeight, Status: input.Status,
	}
	s.prizes[campaignID] = append(s.prizes[campaignID], prize)
	return prize, nil
}

func (s *MemoryStore) UpdatePrize(token string, prizeID string, input model.PrizeMutation) (model.Prize, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureAdmin(token); err != nil {
		return model.Prize{}, err
	}
	for campaignID, prizes := range s.prizes {
		for i := range prizes {
			if prizes[i].ID == prizeID {
				prizes[i].Name = input.Name
				prizes[i].Level = input.Level
				prizes[i].Stock = input.Stock
				prizes[i].ProbabilityWeight = input.ProbabilityWeight
				prizes[i].Status = input.Status
				s.prizes[campaignID] = prizes
				return prizes[i], nil
			}
		}
	}
	return model.Prize{}, ErrCampaignNotFound
}

func (s *MemoryStore) DeletePrize(token string, prizeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureAdmin(token); err != nil {
		return err
	}
	for campaignID, prizes := range s.prizes {
		filtered := prizes[:0]
		for _, prize := range prizes {
			if prize.ID != prizeID {
				filtered = append(filtered, prize)
			}
		}
		s.prizes[campaignID] = filtered
	}
	return nil
}

func (s *MemoryStore) FulfillmentTasks(token string) ([]model.FulfillmentTask, error) {
	if err := s.ensureAdmin(token); err != nil {
		return nil, err
	}
	return []model.FulfillmentTask{}, nil
}

func (s *MemoryStore) UpdateFulfillmentTask(token string, taskID int64, input model.FulfillmentTaskMutation) (model.FulfillmentTask, error) {
	if err := s.ensureAdmin(token); err != nil {
		return model.FulfillmentTask{}, err
	}
	return model.FulfillmentTask{ID: taskID, Status: input.Status, OperatorNote: input.OperatorNote}, nil
}

func (s *MemoryStore) GetDrawStatistics(token, campaignID string) (*model.DrawStatistics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureAdmin(token); err != nil {
		return nil, err
	}

	totalDraws := int64(len(s.drawRecords))
	totalUsers := int64(len(s.users))
	totalWins := int64(0)
	prizeCounts := make(map[string]int64)

	for _, r := range s.drawRecords {
		if r.CampaignID == campaignID || campaignID == "" {
			if r.Result == "win" && r.PrizeID != nil {
				prizeCounts[*r.PrizeID]++
				totalWins++
			}
		}
	}

	breakdown := make([]model.PrizeStatItem, 0, len(prizeCounts))
	for _, prizes := range s.prizes {
		for _, p := range prizes {
			if count, ok := prizeCounts[p.ID]; ok {
				percent := 0.0
				if totalWins > 0 {
					percent = float64(count) / float64(totalWins) * 100
				}
				breakdown = append(breakdown, model.PrizeStatItem{
					PrizeID: p.ID, PrizeName: p.Name, Level: p.Level, Count: count, Percent: percent,
				})
			}
		}
	}

	winRate := 0.0
	if totalDraws > 0 {
		winRate = float64(totalWins) / float64(totalDraws) * 100
	}

	return &model.DrawStatistics{
		TotalDraws:     totalDraws,
		TotalUsers:     totalUsers,
		TotalWins:      totalWins,
		WinRate:        winRate,
		PrizeBreakdown: breakdown,
	}, nil
}

// ============================================================
// 内部辅助
// ============================================================

func (s *MemoryStore) ensureAdmin(token string) error {
	expiresAt, ok := s.adminSessions[token]
	if !ok || expiresAt.Before(time.Now().UTC()) {
		return ErrAdminUnauthorized
	}
	return nil
}

func randomSuffix(size int) string {
	buffer := make([]byte, size)
	_, err := rand.Read(buffer)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return strings.ToLower(hex.EncodeToString(buffer))[:size]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
