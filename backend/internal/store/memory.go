package store

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
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
	inventory      []model.UserInventory
	exchangeOffers []model.ExchangeOffer
	members        map[string]model.UserMember
	nextTaskID     int64
}

func NewMemoryStore(adminUser string, adminPassword string) *MemoryStore {
	s := &MemoryStore{
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
		nextTaskID:     1,
	}
	s.seedDefaultCampaign()
	return s
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

	// 盲盒种子数据
	bb := model.Campaign{
		ID:              "camp_blindbox_001",
		Name:            "梦幻星辰系列盲盒",
		Slug:            "dream-star-series",
		Status:          "online",
		StartsAt:        now.Add(-24 * time.Hour),
		EndsAt:          now.Add(60 * 24 * time.Hour),
		DailyDrawLimit:  10,
		MissWeight:      30,
		BannerImageURL:  "https://static.example.com/blindbox/dream-star/banner.png",
		CampaignSummary: "收集12款星辰主题公仔，集齐全套可兑换隐藏款！每抽必出，软保底30抽递增，60抽硬保底。",
		PityEnabled:     true,
		SoftPityN:       30,
		PityFactor:      0.015,
		HardPityN:       60,
		TargetPrizeID:   "prize_bb_secret",
	}
	s.campaigns[bb.ID] = bb
	s.prizes[bb.ID] = []model.Prize{
		{ID: "prize_bb_01", CampaignID: bb.ID, Name: "射手座·星矢", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_02", CampaignID: bb.ID, Name: "白羊座·穆", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_03", CampaignID: bb.ID, Name: "金牛座·阿鲁", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_04", CampaignID: bb.ID, Name: "双子座·撒加", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_05", CampaignID: bb.ID, Name: "巨蟹座·迪斯", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_06", CampaignID: bb.ID, Name: "狮子座·艾欧", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_07", CampaignID: bb.ID, Name: "处女座·沙加", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_08", CampaignID: bb.ID, Name: "天秤座·童虎", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_09", CampaignID: bb.ID, Name: "天蝎座·米罗", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_10", CampaignID: bb.ID, Name: "射手座·艾俄", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_11", CampaignID: bb.ID, Name: "摩羯座·修罗", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_12", CampaignID: bb.ID, Name: "水瓶座·卡妙", Level: "common", Stock: 2000, ProbabilityWeight: 12, Status: "active"},
		{ID: "prize_bb_rare", CampaignID: bb.ID, Name: "双鱼座·阿布罗狄 (闪光版)", Level: "rare", Stock: 300, ProbabilityWeight: 5, Status: "active"},
		{ID: "prize_bb_secret", CampaignID: bb.ID, Name: "🌟 雅典娜·黄金圣衣 EX", Level: "secret", Stock: 20, ProbabilityWeight: 1, Status: "active"},
	}
}

// ============================================================
// 基础 CRUD
// ============================================================

func (s *MemoryStore) CreateGuestSession(nickname string) (model.User, model.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(nickname) == "" {
		nickname = "Guest" + randomSuffix(4)
	}
	user := model.User{ID: "usr_" + randomSuffix(12), Nickname: nickname, CreatedAt: time.Now().UTC()}
	s.users[user.ID] = user
	session := model.Session{Token: "utk_" + randomSuffix(24), UserID: user.ID, ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour)}
	s.sessions[session.Token] = session
	return user, session, nil
}

func (s *MemoryStore) Campaigns() []model.Campaign {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.Campaign, 0, len(s.campaigns))
	for _, c := range s.campaigns {
		items = append(items, c)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].StartsAt.Before(items[j].StartsAt) })
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
		return s.createMissRecordLocked(userID, campaignID)
	}

	now := time.Now().UTC()
	record := model.DrawRecord{
		ID: "draw_" + randomSuffix(12), CampaignID: campaignID, UserID: userID,
		PrizeID: &prizeID, PrizeName: prizeName, Result: "win", DrawnAt: now,
	}
	s.drawRecords = append([]model.DrawRecord{record}, s.drawRecords...)
	id := prizeID
	s.inventory = append(s.inventory, model.UserInventory{
		ID: "inv_" + randomSuffix(12), UserID: userID, PrizeID: prizeID,
		PrizeName: prizeName, Source: "draw", CreatedAt: now,
		CampaignID: campaignID, PrizeLevel: "",
	})
	_ = id
	return record, nil
}

func (s *MemoryStore) createMissRecordLocked(userID, campaignID string) (model.DrawRecord, error) {
	now := time.Now().UTC()
	record := model.DrawRecord{
		ID: "draw_" + randomSuffix(12), CampaignID: campaignID, UserID: userID,
		PrizeName: "未中奖", Result: "miss", DrawnAt: now,
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
	return s.userDrawCounts[key], nil
}

// ============================================================
// 原 Draw 兼容
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
			missing = append(missing, model.PrizeSummary{PrizeID: p.ID, PrizeName: p.Name, PrizeLevel: p.Level})
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
		ID: "ex_offer_" + randomSuffix(10), UserID: userID,
		HavePrizeID: input.HavePrizeID, WantPrizeID: input.WantPrizeID,
		Status: "pending", CreatedAt: time.Now().UTC(),
	}
	if u, ok := s.users[userID]; ok {
		offer.UserNickname = u.Nickname
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
				return model.ExchangeOffer{}, errors.New("offer no longer pending")
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
	return []model.UserPointsLog{}, nil
}

func (s *MemoryStore) RedeemPrize(userID string, input model.RedeemRequest) (*model.RedeemResult, error) {
	return nil, errors.New("not implemented in memory store")
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
			prizeSummaries = append(prizeSummaries, model.PrizeSummary{PrizeID: prize.ID, PrizeName: prize.Name, PrizeLevel: prize.Level, Stock: prize.Stock})
		}
	}
	campaigns := make([]model.Campaign, 0, len(s.campaigns))
	for _, c := range s.campaigns {
		campaigns = append(campaigns, c)
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
		TotalUsers: len(s.users), TotalDraws: len(s.drawRecords), TotalWins: totalWins,
		Campaigns: campaigns, PrizeSummaries: prizeSummaries,
		RecentDraws: recentDraws, UserDrawBalance: balance,
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
		ID: "camp_" + randomSuffix(10), Name: input.Name, Slug: input.Slug,
		Status: input.Status, StartsAt: input.StartsAt, EndsAt: input.EndsAt,
		DailyDrawLimit: input.DailyDrawLimit, MissWeight: input.MissWeight,
		BannerImageURL: input.BannerImageURL, CampaignSummary: input.CampaignSummary,
		PityEnabled: input.PityEnabled, SoftPityN: input.SoftPityN,
		PityFactor: input.PityFactor, HardPityN: input.HardPityN,
		TargetPrizeID: input.TargetPrizeID,
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
		ID: campaignID, Name: input.Name, Slug: input.Slug,
		Status: input.Status, StartsAt: input.StartsAt, EndsAt: input.EndsAt,
		DailyDrawLimit: input.DailyDrawLimit, MissWeight: input.MissWeight,
		BannerImageURL: input.BannerImageURL, CampaignSummary: input.CampaignSummary,
		PityEnabled: input.PityEnabled, SoftPityN: input.SoftPityN,
		PityFactor: input.PityFactor, HardPityN: input.HardPityN,
		TargetPrizeID: input.TargetPrizeID,
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
		ID: "prize_" + randomSuffix(10), CampaignID: campaignID,
		Name: input.Name, Level: input.Level, Stock: input.Stock,
		ProbabilityWeight: input.ProbabilityWeight, Status: input.Status,
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
	for cid, prizes := range s.prizes {
		for i := range prizes {
			if prizes[i].ID == prizeID {
				prizes[i].Name = input.Name
				prizes[i].Level = input.Level
				prizes[i].Stock = input.Stock
				prizes[i].ProbabilityWeight = input.ProbabilityWeight
				prizes[i].Status = input.Status
				s.prizes[cid] = prizes
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
	for cid, prizes := range s.prizes {
		filtered := prizes[:0]
		for _, prize := range prizes {
			if prize.ID != prizeID {
				filtered = append(filtered, prize)
			}
		}
		s.prizes[cid] = filtered
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
		if (campaignID == "" || r.CampaignID == campaignID) && r.Result == "win" && r.PrizeID != nil {
			prizeCounts[*r.PrizeID]++
			totalWins++
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
				breakdown = append(breakdown, model.PrizeStatItem{PrizeID: p.ID, PrizeName: p.Name, Level: p.Level, Count: count, Percent: percent})
			}
		}
	}
	winRate := 0.0
	if totalDraws > 0 {
		winRate = float64(totalWins) / float64(totalDraws) * 100
	}
	return &model.DrawStatistics{TotalDraws: totalDraws, TotalUsers: totalUsers, TotalWins: totalWins, WinRate: winRate, PrizeBreakdown: breakdown}, nil
}

func (s *MemoryStore) ensureAdmin(token string) error {
	expiresAt, ok := s.adminSessions[token]
	if !ok || expiresAt.Before(time.Now().UTC()) {
		return ErrAdminUnauthorized
	}
	return nil
}

func randomSuffix(size int) string {
	buf := make([]byte, size)
	_, err := rand.Read(buf)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return strings.ToLower(hex.EncodeToString(buf))[:size]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
