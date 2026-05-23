package store

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	mathrand "math/rand/v2"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"campaign-lottery-platform/backend/internal/model"
	"campaign-lottery-platform/backend/internal/store"
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

	// 🆕 月卡系统
	monthCards   map[string]*model.MonthCard
	freeDrawUsed map[string]int

	// 月卡/付费卡系统
	userCards map[string]model.UserCard

	// 🆕 战令系统
	battlePasses          map[string]model.BattlePass           // key: userID:seasonID
	battlePassSeasons     map[int]model.BattlePassSeason
	battlePassTasks       []model.BattlePassTask
	battlePassTaskProgress map[string]model.BattlePassTaskProgress // key: userID:taskID
	battlePassRewards     []model.BattlePassReward
	nextSeasonID          int
	nextTaskIDCounter     int

	// 🆕 商店 + 道具 + 首充
	shopItems            []model.ShopItem
	userItems            map[string]model.UserItem // key: userID:itemType
	firstRechargeRecords map[string]model.UserFirstRecharge // key: userID

	// 🆕 v1.5 社交裂变
	inviteRecords       []model.InviteRecord
	assistProgress      map[string]model.AssistProgress // key: inviterID:assistType
	assistActions       []model.AssistAction
	teamIDSeq           int
	teams               map[string]model.Team
	teamMembers         map[string][]model.TeamMember // key: teamID
	giftRecords         []model.GiftRecord
	giftIDSeq           int
	shareCards          map[string][]model.ShareCard // key: userID
	inviteIDSeq         int
	assistIDSeq         int
	shareCardIDSeq      int
	teamRewards         map[string]model.TeamReward  // key: teamID
	giftPrizeInfo       map[string]string            // giftID -> prizeName:prizeLevel

	// 🆕 v1.6 碎片拼图
	puzzleTemplates    []model.PuzzleTemplate
	puzzleProgresses   map[string]*model.PuzzleProgress // key: userID:templateID
	puzzleTeams        map[string]*model.PuzzleTeam     // key: teamID
	puzzleTeamIDSeq    int

	// 🆕 v1.6 预约抢购
	flashSales         []model.FlashSale
	flashSubscriptions []model.FlashSubscription
	flashIDSeq         int

	// 🆕 活动系统
	activities              []model.Activity
	activityParticipations  []model.ActivityParticipation
	activityRewards         []model.ActivityReward
	activityIDSeq           int
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
		monthCards:     make(map[string]*model.MonthCard),
		freeDrawUsed:   make(map[string]int),
		userCards:      make(map[string]model.UserCard),
		battlePassSeasons:      make(map[int]model.BattlePassSeason),
		battlePasses:           make(map[string]*model.BattlePass),
		battlePassTasks:        make([]model.BattlePassTask, 0),
		battlePassTaskProgress: make(map[string]model.BattlePassTaskProgress),
		battlePassRewards:      make([]model.BattlePassReward, 0),
		shopItems:              make([]model.ShopItem, 0),
		userItems:              make(map[string]model.UserItem),
		firstRechargeRecords:   make(map[string]model.UserFirstRecharge),
		inviteRecords:          make([]model.InviteRecord, 0),
		assistProgress:         make(map[string]model.AssistProgress),
		assistActions:          make([]model.AssistAction, 0),
		teams:                  make(map[string]model.Team),
		teamMembers:            make(map[string][]model.TeamMember),
		giftRecords:            make([]model.GiftRecord, 0),
		shareCards:             make(map[string][]model.ShareCard),
		teamRewards:            make(map[string]model.TeamReward),
		giftPrizeInfo:          make(map[string]string),
		puzzleTemplates:        make([]model.PuzzleTemplate, 0),
		puzzleProgresses:       make(map[string]*model.PuzzleProgress),
		puzzleTeams:            make(map[string]*model.PuzzleTeam),
		puzzleTeamIDSeq:        0,
		flashSales:             make([]model.FlashSale, 0),
		flashSubscriptions:     make([]model.FlashSubscription, 0),
		flashIDSeq:             0,
		activities:             make([]model.Activity, 0),
		activityParticipations: make([]model.ActivityParticipation, 0),
		activityRewards:        make([]model.ActivityReward, 0),
		activityIDSeq:          0,
	}

	store.seedDefaultCampaign()
	store.seedBattlePass()
	store.seedShop()
	store.seedPuzzle()
	store.seedActivities()
	return store
}

func (s *MemoryStore) Seed() error { return nil }

// 🆕 seedBattlePass 初始化默认战令赛季、任务和奖励
func (s *MemoryStore) seedBattlePass() {
	now := time.Now().UTC()
	seasonID := 1
	s.battlePassSeasons[seasonID] = model.BattlePassSeason{
		ID: seasonID, Name: "S1 星辰之约",
		MaxLevel: 50, XPPerLevel: 100,
		StartAt: now.Add(-7 * 24 * time.Hour),
		EndAt:   now.Add(21 * 24 * time.Hour),
		Status:  "active",
	}
	// 每日任务
	s.battlePassTasks = append(s.battlePassTasks,
		model.BattlePassTask{ID: 1, SeasonID: seasonID, Type: "daily", Name: "每日签到", Description: "完成每日签到", XPReward: 20, TargetCount: 1},
		model.BattlePassTask{ID: 2, SeasonID: seasonID, Type: "daily", Name: "单抽一次", Description: "进行一次单抽", XPReward: 30, TargetCount: 1},
		model.BattlePassTask{ID: 3, SeasonID: seasonID, Type: "daily", Name: "分享开盒", Description: "分享开盒结果", XPReward: 10, TargetCount: 1},
		model.BattlePassTask{ID: 4, SeasonID: seasonID, Type: "weekly", Name: "十连抽一次", Description: "进行一次十连抽", XPReward: 50, TargetCount: 1},
		model.BattlePassTask{ID: 5, SeasonID: seasonID, Type: "weekly", Name: "合成一次", Description: "进行合成操作", XPReward: 40, TargetCount: 1},
		model.BattlePassTask{ID: 6, SeasonID: seasonID, Type: "weekly", Name: "发起交换", Description: "发起一次交换", XPReward: 30, TargetCount: 1},
		model.BattlePassTask{ID: 7, SeasonID: seasonID, Type: "season", Name: "集齐一个系列", Description: "完整集齐任意一个系列", XPReward: 200, TargetCount: 1},
		model.BattlePassTask{ID: 8, SeasonID: seasonID, Type: "season", Name: "合成隐藏款", Description: "合成获得隐藏款", XPReward: 300, TargetCount: 1},
	)
	// 免费版奖励
	for lvl := 5; lvl <= 50; lvl += 5 {
		s.battlePassRewards = append(s.battlePassRewards,
			model.BattlePassReward{Level: lvl, PassType: "free", RewardType: "points", RewardName: "积分", RewardQty: lvl * 10},
		)
	}
	// 付费版额外奖励
	for lvl := 2; lvl <= 50; lvl += 2 {
		s.battlePassRewards = append(s.battlePassRewards,
			model.BattlePassReward{Level: lvl, PassType: "paid", RewardType: "draw_ticket", RewardName: "单抽券", RewardQty: 1},
		)
	}
	s.battlePassRewards = append(s.battlePassRewards,
		model.BattlePassReward{Level: 50, PassType: "paid", RewardType: "prize", RewardName: "S1限定头像框", RewardQty: 1, RewardID: "bp_reward_001"},
	)
}

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

	// 新用户赠送初始积分
	now := time.Now().UTC()
	s.members[user.ID] = model.UserMember{
		UserID:    user.ID,
		Level:     model.MemberNormal,
		Points:    100,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID:        int64(len(s.pointsLog) + 1),
		UserID:    user.ID,
		Points:    100,
		Balance:   100,
		Reason:    "welcome",
		Remark:    "新用户注册赠送",
		CreatedAt: now,
	})

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

func (s *MemoryStore) LogPoints(userID string, points int64, balance int64, reason, remark string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID:        int64(len(s.pointsLog) + 1),
		UserID:    userID,
		Points:    points,
		Balance:   balance,
		Reason:    reason,
		Remark:    remark,
		CreatedAt: time.Now().UTC(),
	})
	return nil
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

// ============================================================
// 月卡/付费卡系统 MemoryStore 实现
// ============================================================

func (s *MemoryStore) GetUserCard(userID string) (*model.UserCard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	card, ok := s.userCards[userID]
	if !ok || card.ExpiresAt.Before(time.Now().UTC()) {
		return nil, nil
	}
	// 每天重置免费次数
	today := time.Now().UTC().Format("2006-01-02")
	if card.FreeDate != today {
		card.DailyFreeUsed = 0
		card.FreeDate = today
		s.userCards[userID] = card // 存回去
	}
	return &card, nil
}

func (s *MemoryStore) BuyCard(userID string, cardType model.CardType) (*model.BuyCardResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, ok := model.CardConfigs[cardType]
	if !ok {
		return nil, fmt.Errorf("unknown card type: %s", cardType)
	}

	member, ok := s.members[userID]
	if !ok {
		member = model.UserMember{UserID: userID, Level: model.MemberNormal, Points: 0}
	}
	cost := int64(cfg.Price)
	if member.Points < cost {
		return nil, store.ErrInsufficientPoints
	}

	member.Points -= cost
	s.members[userID] = member

	now := time.Now().UTC()
	card := model.UserCard{
		UserID:    userID,
		CardType:  cardType,
		StartedAt: now,
		ExpiresAt: now.AddDate(0, 0, cfg.DurationDays),
		FreeDate:  now.Format("2006-01-02"),
	}
	s.userCards[userID] = card

	// 记录积分
	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID: int64(len(s.pointsLog) + 1), UserID: userID, Points: -cost,
		Balance: member.Points, Reason: "buy_card", Remark: "购买" + cfg.Description, CreatedAt: now,
	})

	return &model.BuyCardResult{
		CardType: cardType, ExpiresAt: card.ExpiresAt.Format("2006-01-02"),
		Price: cfg.Price, Points: member.Points,
	}, nil
}

func (s *MemoryStore) ConsumeFreeDraw(userID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	card, ok := s.userCards[userID]
	if !ok || card.ExpiresAt.Before(time.Now().UTC()) {
		return false, nil
	}
	today := time.Now().UTC().Format("2006-01-02")
	if card.FreeDate != today {
		card.DailyFreeUsed = 0
		card.FreeDate = today
	}
	cfg := model.CardConfigs[card.CardType]
	if card.DailyFreeUsed >= cfg.FreeDrawsDaily {
		return false, nil
	}
	card.DailyFreeUsed++
	s.userCards[userID] = card
	return true, nil
}

func (s *MemoryStore) GetFreeDrawRemaining(userID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	card, ok := s.userCards[userID]
	if !ok || card.ExpiresAt.Before(time.Now().UTC()) {
		return 0, nil
	}
	cfg := model.CardConfigs[card.CardType]
	today := time.Now().UTC().Format("2006-01-02")
	used := card.DailyFreeUsed
	if card.FreeDate != today {
		used = 0
	}
	remaining := cfg.FreeDrawsDaily - used
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
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

func (s *MemoryStore) BlendPrizes(userID string, sourcePrizeID string, campaignID string) (*model.BlendResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 查找源款式信息
	var sourcePrize model.Prize
	var found bool
	for _, prizes := range s.prizes {
		for _, p := range prizes {
			if p.ID == sourcePrizeID {
				sourcePrize = p
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

	recipe, ok := model.BlendRecipes[sourcePrize.Level]
	if !ok {
		return nil, fmt.Errorf("no recipe for level: %s", sourcePrize.Level)
	}

	// 检查用户拥有数量
	count := 0
	for _, inv := range s.inventory {
		if inv.UserID == userID && inv.PrizeID == sourcePrizeID {
			count++
		}
	}
	if count < recipe.NeedCount {
		return nil, fmt.Errorf("need %d of %s, have %d", recipe.NeedCount, sourcePrize.Name, count)
	}

	// 找到目标级别的奖品（同级随机选一个 active 且 stock > 0 的）
	var resultPrize model.Prize
	found = false
	for _, prizes := range s.prizes {
		for _, p := range prizes {
			if p.Level == recipe.ResultLevel && p.Status == "active" && p.Stock > 0 {
				resultPrize = p
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("no available prize of level %s to blend into", recipe.ResultLevel)
	}

	now := time.Now().UTC()

	// 删除 N 条库存记录
	deleted := 0
	newInventory := make([]model.UserInventory, 0, len(s.inventory))
	for _, inv := range s.inventory {
		if inv.UserID == userID && inv.PrizeID == sourcePrizeID && deleted < recipe.NeedCount {
			deleted++
			continue
		}
		newInventory = append(newInventory, inv)
	}
	s.inventory = newInventory

	// 扣库存
	for i := range s.prizes[campaignID] {
		if s.prizes[campaignID][i].ID == resultPrize.ID && s.prizes[campaignID][i].Stock > 0 {
			s.prizes[campaignID][i].Stock--
			break
		}
	}

	// 添加结果到用户库存
	s.inventory = append(s.inventory, model.UserInventory{
		ID:         "inv_" + randomSuffix(12),
		UserID:     userID,
		PrizeID:    resultPrize.ID,
		PrizeName:  resultPrize.Name,
		PrizeLevel: resultPrize.Level,
		CampaignID: campaignID,
		Source:     "blend",
		CreatedAt:  now,
	})

	return &model.BlendResult{
		SourcePrizeID:   sourcePrizeID,
		SourcePrizeName: sourcePrize.Name,
		SourceLevel:     sourcePrize.Level,
		ResultPrizeID:   resultPrize.ID,
		ResultPrizeName: resultPrize.Name,
		ResultLevel:     resultPrize.Level,
		RemainingSrc:    count - recipe.NeedCount,
	}, nil
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
		PityConfig:      input.PityConfig,
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
		PityConfig: input.PityConfig,
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

// ─────────────────────────────────────────────────────────────
// 🆕 月卡系统 MemoryStore 实现
// ─────────────────────────────────────────────────────────────

func (s *MemoryStore) GetMonthCard(userID string) (*model.MonthCard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if card, ok := s.monthCards[userID]; ok && card.ExpiresAt.After(time.Now().UTC()) {
		c := *card
		return &c, nil
	}
	return nil, nil
}

func (s *MemoryStore) BuyMonthCard(userID string, cardType model.MonthCardType, pointsCost int64) (*model.MonthCard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	var freeDraws int
	var discount float64
	var price int64
	var duration time.Duration
	switch cardType {
	case model.MonthCardWeekly:
		freeDraws, discount, price, duration = 1, 0.9, 990, 7*24*time.Hour
	case model.MonthCardMonthly:
		freeDraws, discount, price, duration = 2, 0.8, 2800, 30*24*time.Hour
	case model.MonthCardSeason:
		freeDraws, discount, price, duration = 3, 0.75, 6800, 90*24*time.Hour
	default:
		return nil, fmt.Errorf("invalid card type: %s", cardType)
	}
	s.monthCards[userID] = &model.MonthCard{
		ID: "mcard_" + randomSuffix(12), UserID: userID, CardType: cardType,
		Price: price, FreeDraws: freeDraws, DrawDiscount: discount,
		StartedAt: now, ExpiresAt: now.Add(duration), CreatedAt: now,
	}
	c := *s.monthCards[userID]
	return &c, nil
}

func (s *MemoryStore) UseFreeDraw(userID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := "free:" + userID + ":" + time.Now().UTC().Format("2006-01-02")
	s.freeDrawUsed[key]++
	return s.freeDrawUsed[key], nil
}

func (s *MemoryStore) GetTodayFreeDrawUsed(userID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := "free:" + userID + ":" + time.Now().UTC().Format("2006-01-02")
	return s.freeDrawUsed[key], nil
}

// ─────────────────────────────────────────────────────────────
// 🆕 战令系统 MemoryStore 实现
// ─────────────────────────────────────────────────────────────

func (s *MemoryStore) GetActiveSeason() (*model.BattlePassSeason, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, season := range s.battlePassSeasons {
		if season.Status == "active" {
			s := season
			return &s, nil
		}
	}
	return nil, store.ErrNoActiveSeason
}

func (s *MemoryStore) GetUserBattlePass(userID string, seasonID int) (*model.BattlePass, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := userID + ":" + strconv.Itoa(seasonID)
	if bp, ok := s.battlePasses[key]; ok {
		b := *bp
		return &b, nil
	}
	// 创建免费版战令
	now := time.Now().UTC()
	season, err := s.GetActiveSeason()
	if err != nil {
		return nil, err
	}
	bp := &model.BattlePass{
		UserID: userID, SeasonID: seasonID, PassType: "free",
		Level: 1, XP: 0, TotalXP: 0, ClaimedLevels: []int{}, UpdatedAt: now,
	}
	s.battlePasses[key] = bp
	b := *bp
	return &b, nil
}

func (s *MemoryStore) BuyBattlePass(userID string, seasonID int, pointsCost int64) (*model.BattlePass, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userID + ":" + strconv.Itoa(seasonID)
	bp, ok := s.battlePasses[key]
	if !ok {
		return nil, store.ErrCampaignNotFound
	}
	if bp.PassType == "paid" {
		return nil, store.ErrAlreadyPurchased
	}
	now := time.Now().UTC()
	bp.PassType = "paid"
	bp.BoughtAt = now
	bp.UpdatedAt = now
	b := *bp
	return &b, nil
}

func (s *MemoryStore) AddBattlePassXP(userID string, seasonID int, xp int) (*model.BattlePass, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userID + ":" + strconv.Itoa(seasonID)
	bp, ok := s.battlePasses[key]
	if !ok {
		return nil, store.ErrCampaignNotFound
	}
	season, err := s.GetActiveSeason()
	if err != nil {
		return nil, err
	}
	bp.XP += xp
	bp.TotalXP += xp
	for bp.Level < season.MaxLevel && bp.XP >= season.XPPerLevel {
		bp.XP -= season.XPPerLevel
		bp.Level++
	}
	bp.UpdatedAt = time.Now().UTC()
	b := *bp
	return &b, nil
}

func (s *MemoryStore) ClaimBattlePassReward(userID string, seasonID int, level int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userID + ":" + strconv.Itoa(seasonID)
	bp, ok := s.battlePasses[key]
	if !ok {
		return false, store.ErrCampaignNotFound
	}
	if bp.Level < level {
		return false, store.ErrNotEligible
	}
	for _, l := range bp.ClaimedLevels {
		if l == level {
			return false, nil // already claimed
		}
	}
	bp.ClaimedLevels = append(bp.ClaimedLevels, level)
	return true, nil
}

func (s *MemoryStore) GetBattlePassTasks(seasonID int) ([]model.BattlePassTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tasks := make([]model.BattlePassTask, 0)
	for _, t := range s.battlePassTasks {
		if t.SeasonID == seasonID {
			tasks = append(tasks, t)
		}
	}
	return tasks, nil
}

func (s *MemoryStore) GetBattlePassTaskProgress(userID string, seasonID int) ([]model.BattlePassTaskProgress, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	progs := make([]model.BattlePassTaskProgress, 0)
	for _, p := range s.battlePassTaskProgress {
		if p.UserID == userID {
			progs = append(progs, p)
		}
	}
	return progs, nil
}

func (s *MemoryStore) UpdateTaskProgress(userID string, taskID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userID + ":" + strconv.Itoa(taskID)
	prog, ok := s.battlePassTaskProgress[key]
	if !ok {
		s.battlePassTaskProgress[key] = model.BattlePassTaskProgress{
			UserID: userID, TaskID: taskID, Progress: 1, Completed: false,
		}
		return nil
	}
	prog.Progress++
	return nil
}

func (s *MemoryStore) GetBattlePassRewards(seasonID int) ([]model.BattlePassReward, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rewards := make([]model.BattlePassReward, 0)
	for _, r := range s.battlePassRewards {
		if r.Level <= seasonID/1000*1000+50 { // simplified filter
			rewards = append(rewards, r)
		}
	}
	return rewards, nil
}

// 🆕 seedShop 初始化商店商品和首充礼包
func (s *MemoryStore) seedShop() {
	now := time.Now().UTC()
	s.shopItems = []model.ShopItem{
		// 每日特价
		{ID: "daily_special", Name: "每日特价", Description: "每日限购1次，随机获得好礼", PricePoints: 100, PriceCash: 0, ItemType: model.ItemFreeDraw, ItemQty: 1, Stock: -1, DailyLimit: 1, Category: "daily", IsActive: true, SortOrder: 1},
		// 道具商店
		{ID: "hint_card_1", Name: "提示卡", Description: "排除当前池中1个不想要的款式", PricePoints: 10, ItemType: model.ItemHintCard, ItemQty: 1, Stock: -1, Category: "item", IsActive: true, SortOrder: 10},
		{ID: "hint_card_10", Name: "提示卡×10", Description: "十张提示卡打包优惠", PricePoints: 90, ItemType: model.ItemHintCard, ItemQty: 10, Stock: -1, Category: "item", IsActive: true, SortOrder: 11},
		{ID: "see_through_1", Name: "透卡", Description: "预览当前1抽将出什么款（仅普通款）", PricePoints: 20, ItemType: model.ItemSeeThrough, ItemQty: 1, Stock: -1, Category: "item", IsActive: true, SortOrder: 20},
		{ID: "see_through_5", Name: "透卡×5", Description: "五张透卡打包优惠", PricePoints: 90, ItemType: model.ItemSeeThrough, ItemQty: 5, Stock: -1, Category: "item", IsActive: true, SortOrder: 21},
		{ID: "pity_inherit_1", Name: "保底继承券", Description: "保底计数可在同类型不同池间转移一次", PricePoints: 50, ItemType: model.ItemPityInherit, ItemQty: 1, Stock: -1, Category: "item", IsActive: true, SortOrder: 30},
		{ID: "specify_voucher_1", Name: "指定款券", Description: "必得指定普通款（不可指定稀有+）", PricePoints: 200, ItemType: model.ItemSpecifyVoucher, ItemQty: 1, Stock: -1, Category: "item", IsActive: true, SortOrder: 40},
		{ID: "ten_draw_ticket_1", Name: "十连券", Description: "直接进行十连抽，无需消耗积分", PricePoints: 950, ItemType: model.ItemTenDrawTicket, ItemQty: 1, Stock: -1, Category: "item", IsActive: true, SortOrder: 50},
		// 周礼包
		{ID: "weekly_pack", Name: "周礼包", Description: "含积分+十连券+限定时装", PricePoints: 1800, ItemType: model.ItemTenDrawTicket, ItemQty: 2, Stock: 100, DailyLimit: 1, Category: "weekly", IsActive: true, ExpiresAt: ptrTime(now.Add(7 * 24 * time.Hour)), SortOrder: 60},
		// 节日礼包
		{ID: "festival_pack_1", Name: "节日礼包·小", Description: "积分+十连券+限定头像框", PricePoints: 6800, ItemType: model.ItemTenDrawTicket, ItemQty: 5, Stock: 500, DailyLimit: 1, Category: "festival", IsActive: true, ExpiresAt: ptrTime(now.Add(14 * 24 * time.Hour)), SortOrder: 70},
		{ID: "festival_pack_2", Name: "节日礼包·大", Description: "大量积分+十连券+稀有款自选", PricePoints: 19800, ItemType: model.ItemTenDrawTicket, ItemQty: 15, Stock: 100, DailyLimit: 1, Category: "festival", IsActive: true, ExpiresAt: ptrTime(now.Add(14 * 24 * time.Hour)), SortOrder: 71},
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

// 🆕 seedPuzzle 初始化拼图模板
func (s *MemoryStore) seedPuzzle() {
	now := time.Now().UTC()
	s.puzzleTemplates = []model.PuzzleTemplate{
		{
			ID:          "weekly_puzzle",
			Name:        "每周拼图",
			CampaignID:  "camp_launch_001",
			TotalPieces: 6,
			PieceNames:  []string{"碎片一", "碎片二", "碎片三", "碎片四", "碎片五", "碎片六"},
			RewardType:  "prize",
			RewardID:    "prize_001",
			RewardQty:   1,
			RewardName:  "88元红包",
			PeriodType:  "weekly",
			IsActive:    true,
			CreatedAt:   now,
		},
		{
			ID:          "monthly_puzzle",
			Name:        "月度大拼图",
			CampaignID:  "camp_launch_001",
			TotalPieces: 24,
			PieceNames:  []string{"碎1", "碎2", "碎3", "碎4", "碎5", "碎6", "碎7", "碎8", "碎9", "碎10", "碎11", "碎12", "碎13", "碎14", "碎15", "碎16", "碎17", "碎18", "碎19", "碎20", "碎21", "碎22", "碎23", "碎24"},
			RewardType:  "prize",
			RewardID:    "prize_003",
			RewardQty:   1,
			RewardName:  "品牌周边礼盒",
			PeriodType:  "monthly",
			IsActive:    true,
			CreatedAt:   now,
		},
	}
}

// ============================================================
// 🆕 商店 + 道具 + 首充 MemoryStore 实现
// ============================================================

func (s *MemoryStore) GetShopItems() []model.ShopItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.ShopItem, 0, len(s.shopItems))
	for _, item := range s.shopItems {
		if !item.IsActive {
			continue
		}
		if item.ExpiresAt != nil && time.Now().UTC().After(*item.ExpiresAt) {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MemoryStore) BuyShopItem(userID string, itemID string, quantity int) (*model.BuyShopItemResult, error) {
	if quantity <= 0 {
		quantity = 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// 查找商品
	var item *model.ShopItem
	for i := range s.shopItems {
		if s.shopItems[i].ID == itemID {
			item = &s.shopItems[i]
			break
		}
	}
	if item == nil || !item.IsActive {
		return nil, store.ErrCampaignNotFound
	}

	// 检查过期
	if item.ExpiresAt != nil && time.Now().UTC().After(*item.ExpiresAt) {
		return nil, fmt.Errorf("item expired")
	}

	// 检查库存
	if item.Stock >= 0 && item.Stock < quantity {
		return nil, fmt.Errorf("insufficient stock: only %d left", item.Stock)
	}

	// 检查用户积分
	member, ok := s.members[userID]
	if !ok {
		return nil, store.ErrUnauthorized
	}
	totalCost := item.PricePoints * int64(quantity)
	if member.Points < totalCost {
		return nil, store.ErrInsufficientPoints
	}

	// 扣积分
	member.Points -= totalCost
	member.TotalSpent += totalCost
	s.members[userID] = member

	// 扣库存
	if item.Stock >= 0 {
		item.Stock -= quantity
	}

	// 添加道具
	s.addUserItemLocked(userID, item.ItemType, item.ItemQty*quantity)

	// 记录日志（简化）
	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID: s.nextTaskID, UserID: userID, Points: -totalCost,
		Balance: member.Points, Reason: "shop", Remark: fmt.Sprintf("购买 %s x%d", item.Name, quantity),
		CreatedAt: time.Now().UTC(),
	})
	s.nextTaskID++

	newQty, _ := s.getUserItemQtyLocked(userID, item.ItemType)

	return &model.BuyShopItemResult{
		ItemType:   item.ItemType,
		ItemName:   item.Name,
		Quantity:   quantity,
		PointsCost: totalCost,
		NewPoints:  member.Points,
		NewQty:     newQty,
	}, nil
}

func (s *MemoryStore) addUserItemLocked(userID string, itemType model.ItemType, qty int) {
	key := userID + ":" + string(itemType)
	existing, ok := s.userItems[key]
	if ok {
		existing.Quantity += qty
		s.userItems[key] = existing
	} else {
		s.userItems[key] = model.UserItem{UserID: userID, ItemType: itemType, Quantity: qty}
	}
}

func (s *MemoryStore) getUserItemQtyLocked(userID string, itemType model.ItemType) (int, bool) {
	key := userID + ":" + string(itemType)
	existing, ok := s.userItems[key]
	if !ok {
		return 0, false
	}
	return existing.Quantity, true
}

func (s *MemoryStore) GetUserItemQty(userID string, itemType model.ItemType) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	qty, _ := s.getUserItemQtyLocked(userID, itemType)
	return qty, nil
}

func (s *MemoryStore) AddUserItem(userID string, itemType model.ItemType, qty int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addUserItemLocked(userID, itemType, qty)
	return nil
}

func (s *MemoryStore) UseUserItem(userID string, itemType model.ItemType, qty int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userID + ":" + string(itemType)
	existing, ok := s.userItems[key]
	if !ok || existing.Quantity < qty {
		return false, nil
	}
	existing.Quantity -= qty
	if existing.Quantity <= 0 {
		delete(s.userItems, key)
	} else {
		s.userItems[key] = existing
	}
	return true, nil
}

func (s *MemoryStore) GetUserItems(userID string) ([]model.UserItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.UserItem, 0)
	for _, v := range s.userItems {
		if v.UserID == userID && v.Quantity > 0 {
			items = append(items, v)
		}
	}
	return items, nil
}

// 🆕 首充礼包

func (s *MemoryStore) GetFirstRechargeStatus(userID string) (*model.UserFirstRecharge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.firstRechargeRecords[userID]
	if !ok {
		return &model.UserFirstRecharge{UserID: userID, Claimed: []string{}}, nil
	}
	cp := rec
	return &cp, nil
}

func (s *MemoryStore) ClaimFirstRecharge(userID string, packID string) (*model.ClaimFirstRechargeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查礼包是否有效
	pack, ok := model.FirstRechargePacks[packID]
	if !ok {
		return nil, fmt.Errorf("invalid first recharge pack: %s", packID)
	}

	// 检查是否已经领取过
	rec, exists := s.firstRechargeRecords[userID]
	if exists {
		for _, claimed := range rec.Claimed {
			if claimed == packID {
				return nil, fmt.Errorf("already claimed pack: %s", packID)
			}
		}
	} else {
		rec = model.UserFirstRecharge{UserID: userID, Claimed: []string{}}
	}

	// 检查积分是否足够
	member, ok := s.members[userID]
	if !ok {
		return nil, store.ErrUnauthorized
	}
	if member.Points < pack.PricePoints {
		return nil, store.ErrInsufficientPoints
	}

	// 扣积分
	member.Points -= pack.PricePoints
	member.TotalSpent += pack.PricePoints
	s.members[userID] = member

	// 发放礼包内容
	for _, item := range pack.Items {
		switch item.Type {
		case "points":
			// 已包含在礼包积分中，直接加到余额
			member.Points += int64(item.Qty)
			s.members[userID] = member
		case "hint_card":
			s.addUserItemLocked(userID, model.ItemHintCard, item.Qty)
		case "see_through":
			s.addUserItemLocked(userID, model.ItemSeeThrough, item.Qty)
		case "ten_draw_ticket":
			s.addUserItemLocked(userID, model.ItemTenDrawTicket, item.Qty)
		case "free_draw":
			s.addUserItemLocked(userID, model.ItemFreeDraw, item.Qty)
		case "prize":
			// 随机找一个该系列的稀有款发放（简化：直接给积分等价物）
			// 实际应该发款式到用户库存，这里简化处理
			pointsExtra := int64(item.Qty * 200) // 稀有款约200积分
			member.Points += pointsExtra
			s.members[userID] = member
		case "month_card":
			// 赠送月卡（一个月的monthly卡）
			card := &model.MonthCard{
				ID: userID + ":monthly:gift", UserID: userID,
				CardType: model.MonthCardMonthly, Price: 0,
				FreeDraws: 2, DrawDiscount: 0.8,
				StartedAt: time.Now().UTC(),
				ExpiresAt: time.Now().UTC().Add(30 * 24 * time.Hour),
				CreatedAt: time.Now().UTC(),
			}
			s.monthCards[userID] = card
		}
	}

	// 记录领取
	member.Points += pack.PricePoints // 退回礼包价值作为积分（首充双倍等效）
	s.members[userID] = member

	rec.Claimed = append(rec.Claimed, packID)
	s.firstRechargeRecords[userID] = rec

	// 记录日志
	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID: s.nextTaskID, UserID: userID, Points: 0,
		Balance: member.Points, Reason: "first_recharge", Remark: fmt.Sprintf("领取%s", pack.Name),
		CreatedAt: time.Now().UTC(),
	})
	s.nextTaskID++

	return &model.ClaimFirstRechargeResult{
		PackID: packID, PackName: pack.Name,
		Items: pack.Items, NewPoints: member.Points,
	}, nil
}

// GenerateID generates a random hex ID using crypto/rand (8 bytes → 16 hex chars)
func (s *MemoryStore) GenerateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateInviteRecord creates a new invite record
func (s *MemoryStore) CreateInviteRecord(inviterID, inviteeID string) *model.InviteRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.inviteIDSeq++
	record := model.InviteRecord{
		ID:        "inv_" + strconv.Itoa(s.inviteIDSeq),
		InviterID: inviterID,
		InviteeID: inviteeID,
		CreatedAt: time.Now().UTC(),
	}
	s.inviteRecords = append(s.inviteRecords, record)
	return &record
}

// GetInviteRecords returns all invite records where InviterID == userID
func (s *MemoryStore) GetInviteRecords(userID string) []model.InviteRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []model.InviteRecord
	for _, r := range s.inviteRecords {
		if r.InviterID == userID {
			result = append(result, r)
		}
	}
	return result
}

// GetInviteStats returns total invite count and total assist count for a user
func (s *MemoryStore) GetInviteStats(userID string) (invites int, assists int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, r := range s.inviteRecords {
		if r.InviterID == userID {
			invites++
		}
	}
	for _, a := range s.assistActions {
		if a.InviterID == userID {
			assists++
		}
	}
	return
}

// GetOrCreateAssistProgress returns existing assist progress or creates a new one.
// TargetCount: free_draw=3, pity_reduce=5, craft_boost=2. ExpiresAt = now+24h.
func (s *MemoryStore) GetOrCreateAssistProgress(inviterID string, assistType model.AssistType) *model.AssistProgress {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s", inviterID, assistType)
	if p, ok := s.assistProgress[key]; ok {
		return &p
	}

	var targetCount int
	switch assistType {
	case model.AssistFreeDraw:
		targetCount = 3
	case model.AssistPityReduce:
		targetCount = 5
	case model.AssistCraftBoost:
		targetCount = 2
	default:
		targetCount = 3
	}

	created := time.Now().UTC()
	progress := model.AssistProgress{
		InviterID:   inviterID,
		AssistType:  assistType,
		TargetCount: targetCount,
		Current:     0,
		Claimed:     false,
		ExpiresAt:   created.Add(24 * time.Hour),
		CreatedAt:   created,
	}
	s.assistProgress[key] = progress
	return &progress
}

// IsAssistActionRecorded checks if the same helper already assisted today for the given type
func (s *MemoryStore) IsAssistActionRecorded(inviterID, helperID string, assistType model.AssistType) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	for _, a := range s.assistActions {
		if a.InviterID == inviterID && a.HelperID == helperID && a.AssistType == assistType {
			if a.CreatedAt.Year() == now.Year() && a.CreatedAt.YearDay() == now.YearDay() {
				return true
			}
		}
	}
	return false
}

// RecordAssistAction records a new assist action with a generated ID and current timestamp
func (s *MemoryStore) RecordAssistAction(inviterID, helperID string, assistType model.AssistType) *model.AssistAction {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.assistIDSeq++
	action := model.AssistAction{
		ID:         "ast_" + strconv.Itoa(s.assistIDSeq),
		InviterID:  inviterID,
		HelperID:   helperID,
		AssistType: assistType,
		CreatedAt:  time.Now().UTC(),
	}
	s.assistActions = append(s.assistActions, action)
	return &action
}

// ============================================================
// 队伍社交 (Team Social)
// ============================================================

// CreateTeam creates a new team with the given captain and request.
func (s *MemoryStore) CreateTeam(captainID string, input model.CreateTeamRequest) (*model.Team, error) {
	if input.MaxMembers < 2 || input.MaxMembers > 5 {
		return nil, errors.New("max_members must be between 2 and 5")
	}
	if input.GoalDraws <= 0 {
		return nil, errors.New("goal_draws must be greater than 0")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check user doesn't already have an active team
	activeTeam := s.getUserActiveTeamLocked(captainID)
	if activeTeam != nil {
		return nil, errors.New("user already has an active team")
	}

	now := time.Now().UTC()
	team := model.Team{
		ID:           s.GenerateID(),
		CaptainID:    captainID,
		Name:         input.Name,
		MaxMembers:   input.MaxMembers,
		GoalDraws:    input.GoalDraws,
		CurrentDraws: 0,
		StartsAt:     now,
		ExpiresAt:    now.Add(48 * time.Hour),
		Status:       "recruiting",
		CreatedAt:    now,
	}

	s.teams[team.ID] = team

	// Add captain as first team member
	member := model.TeamMember{
		TeamID:   team.ID,
		UserID:   captainID,
		Draws:    0,
		JoinedAt: now,
	}
	s.teamMembers[team.ID] = []model.TeamMember{member}

	return &team, nil
}

// JoinTeam adds a user to an existing team.
func (s *MemoryStore) JoinTeam(userID, teamID string) (*model.TeamMember, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.teams[teamID]
	if !ok {
		return nil, errors.New("team not found")
	}
	if team.Status != "recruiting" {
		return nil, errors.New("team is not recruiting")
	}

	members := s.teamMembers[teamID]
	if len(members) >= team.MaxMembers {
		return nil, errors.New("team is full")
	}

	// Check user not already in team
	for _, m := range members {
		if m.UserID == userID {
			return nil, errors.New("user is already in this team")
		}
	}

	now := time.Now().UTC()
	member := model.TeamMember{
		TeamID:   teamID,
		UserID:   userID,
		Draws:    0,
		JoinedAt: now,
	}
	s.teamMembers[teamID] = append(members, member)

	// If full, set status to active
	if len(s.teamMembers[teamID]) >= team.MaxMembers {
		team.Status = "active"
		s.teams[teamID] = team
	}

	return &member, nil
}

// LeaveTeam removes a user from a team. If the captain leaves, the next oldest
// member becomes captain. If no members remain, the team is disbanded.
func (s *MemoryStore) LeaveTeam(userID, teamID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.teams[teamID]
	if !ok {
		return errors.New("team not found")
	}

	members := s.teamMembers[teamID]
	found := false
	var idx int
	for i, m := range members {
		if m.UserID == userID {
			found = true
			idx = i
			break
		}
	}
	if !found {
		return errors.New("user is not a member of this team")
	}

	// Remove the member
	members = append(members[:idx], members[idx+1:]...)

	if len(members) == 0 {
		// No members left, disband the team
		delete(s.teams, teamID)
		delete(s.teamMembers, teamID)
		return nil
	}

	s.teamMembers[teamID] = members

	// If the leaving user was the captain, transfer captaincy to the next oldest member
	if team.CaptainID == userID {
		// First member in the slice is the oldest (they joined earliest)
		team.CaptainID = members[0].UserID
		s.teams[teamID] = team
	}

	return nil
}

// GetTeam returns a team by ID.
func (s *MemoryStore) GetTeam(teamID string) (*model.Team, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	team, ok := s.teams[teamID]
	if !ok {
		return nil, errors.New("team not found")
	}
	return &team, nil
}

// GetTeamMembers returns all members of a team.
func (s *MemoryStore) GetTeamMembers(teamID string) ([]model.TeamMember, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	members, ok := s.teamMembers[teamID]
	if !ok {
		return nil, errors.New("team not found")
	}
	return members, nil
}

// GetUserActiveTeam finds an active team (recruiting or active) for a user.
func (s *MemoryStore) GetUserActiveTeam(userID string) (*model.Team, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t := s.getUserActiveTeamLocked(userID)
	if t == nil {
		return nil, nil
	}
	return t, nil
}

// getUserActiveTeamLocked is an internal helper that assumes the read lock is held.
func (s *MemoryStore) getUserActiveTeamLocked(userID string) *model.Team {
	for _, team := range s.teams {
		if team.Status != "recruiting" && team.Status != "active" {
			continue
		}
		members := s.teamMembers[team.ID]
		for _, m := range members {
			if m.UserID == userID {
				return &team
			}
		}
	}
	return nil
}

// AddTeamDraw increments the draw count for a team and its member.
func (s *MemoryStore) AddTeamDraw(userID, teamID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.teams[teamID]
	if !ok {
		return 0, errors.New("team not found")
	}

	members := s.teamMembers[teamID]
	found := false
	for i, m := range members {
		if m.UserID == userID {
			m.Draws++
			members[i] = m
			found = true
			break
		}
	}
	if !found {
		return 0, errors.New("user is not a member of this team")
	}

	s.teamMembers[teamID] = members
	team.CurrentDraws++
	s.teams[teamID] = team

	return team.CurrentDraws, nil
}

// CompleteTeam marks a team as completed and creates a team reward.
// Each member (including captain) gets GoalDraws * 10 points.
// The captain gets an additional 5% bonus.
func (s *MemoryStore) CompleteTeam(teamID string) (*model.TeamReward, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.teams[teamID]
	if !ok {
		return nil, errors.New("team not found")
	}

	now := time.Now().UTC()
	team.Status = "completed"
	team.CompletedAt = &now
	s.teams[teamID] = team

	// Calculate rewards
	basePoints := team.GoalDraws * 10
	captainPoints := int(float64(basePoints) * 1.05)

	// Award points to members (update user member balances)
	members := s.teamMembers[teamID]
	for _, m := range members {
		pts := basePoints
		if m.UserID == team.CaptainID {
			pts = captainPoints
		}
		if member, ok := s.members[m.UserID]; ok {
			member.Points += int64(pts)
			s.members[m.UserID] = member
		}
		// Also log the points
		s.pointsLog = append(s.pointsLog, model.UserPointsLog{
			ID:       s.nextTaskID,
			UserID:   m.UserID,
			Points:   int64(pts),
			Balance:  s.members[m.UserID].Points,
			Reason:   "team_reward",
			Remark:   fmt.Sprintf("组队奖励：完成 %d 次开盒", team.GoalDraws),
			CreatedAt: now,
		})
		s.nextTaskID++
	}

	reward := model.TeamReward{
		TeamID:      teamID,
		CaptainID:   team.CaptainID,
		RewardType:  "points",
		RewardQty:   captainPoints + basePoints*(len(members)-1),
		Description: fmt.Sprintf("组队完成 %d 次开盒，每位成员获得 %d 积分，队长获得 %d 积分", team.GoalDraws, basePoints, captainPoints),
	}
	s.teamRewards[teamID] = reward

	return &reward, nil
}

// ExpireTeam sets a team's status to expired.
func (s *MemoryStore) ExpireTeam(teamID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.teams[teamID]
	if !ok {
		return errors.New("team not found")
	}
	team.Status = "expired"
	s.teams[teamID] = team
	return nil
}

// GetExpiredTeams returns all teams that have recruiting or active status
// but have passed their expiration time.
func (s *MemoryStore) GetExpiredTeams() []model.Team {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	var expired []model.Team
	for _, team := range s.teams {
		if (team.Status == "recruiting" || team.Status == "active") && team.ExpiresAt.Before(now) {
			expired = append(expired, team)
		}
	}
	return expired
}

// IncrementAssistProgress increments the current count by 1 and returns the updated progress
func (s *MemoryStore) IncrementAssistProgress(inviterID string, assistType model.AssistType) *model.AssistProgress {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s", inviterID, assistType)
	p := s.assistProgress[key]
	p.Current++
	s.assistProgress[key] = p
	return &p
}

// ClaimAssistReward marks the assist progress as claimed
func (s *MemoryStore) ClaimAssistReward(inviterID string, assistType model.AssistType) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s", inviterID, assistType)
	p := s.assistProgress[key]
	p.Claimed = true
	s.assistProgress[key] = p
}

// GetAssistProgress returns the assist progress for the given inviter and type
func (s *MemoryStore) GetAssistProgress(inviterID string, assistType model.AssistType) *model.AssistProgress {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", inviterID, assistType)
	p, ok := s.assistProgress[key]
	if !ok {
		return nil
	}
	return &p
}

// ============================================================
// 🆕 礼物赠送系统
// ============================================================

// CreateGift 创建礼物（赠送道具）
func (s *MemoryStore) CreateGift(giverID, receiverID, prizeID, campaignID string) (*model.GiftRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 查找奖品信息
	prizes, ok := s.prizes[campaignID]
	if !ok {
		return nil, fmt.Errorf("campaign not found: %s", campaignID)
	}
	var prizeName, prizeLevel string
	for _, p := range prizes {
		if p.ID == prizeID {
			prizeName = p.Name
			prizeLevel = p.Level
			break
		}
	}
	if prizeName == "" {
		return nil, fmt.Errorf("prize not found: %s", prizeID)
	}

	// 查找赠送者的库存
	invIdx := -1
	for i, inv := range s.inventory {
		if inv.UserID == giverID && inv.PrizeID == prizeID && inv.CampaignID == campaignID {
			invIdx = i
			break
		}
	}
	if invIdx == -1 {
		return nil, fmt.Errorf("giver does not own this prize in inventory")
	}

	// 稀有/隐藏/限定款收取包装费
	feePoints := int64(0)
	if prizeLevel == model.PrizeLevelRare || prizeLevel == model.PrizeLevelSecret || prizeLevel == model.PrizeLevelLimited {
		feePoints = 500
		member, ok := s.members[giverID]
		if !ok {
			member = model.UserMember{UserID: giverID, Level: model.MemberNormal, Points: 0}
		}
		if member.Points < feePoints {
			return nil, store.ErrInsufficientPoints
		}
		member.Points -= feePoints
		s.members[giverID] = member

		s.pointsLog = append(s.pointsLog, model.UserPointsLog{
			ID:        int64(len(s.pointsLog) + 1),
			UserID:    giverID,
			Points:    -feePoints,
			Balance:   member.Points,
			Reason:    "gift_fee",
			Remark:    fmt.Sprintf("赠送%s包装费", prizeName),
			CreatedAt: time.Now().UTC(),
		})
	}

	// 从赠送者库存中删除该道具
	s.inventory = append(s.inventory[:invIdx], s.inventory[invIdx+1:]...)

	// 创建礼物记录
	s.giftIDSeq++
	now := time.Now().UTC()
	gift := model.GiftRecord{
		ID:         "gift_" + strconv.Itoa(s.giftIDSeq),
		GiverID:    giverID,
		ReceiverID: receiverID,
		PrizeID:    prizeID,
		PrizeName:  prizeName,
		PrizeLevel: prizeLevel,
		FeePoints:  feePoints,
		Status:     "sent",
		CreatedAt:  now,
		ExpiresAt:  now.Add(24 * time.Hour),
	}
	s.giftRecords = append(s.giftRecords, gift)

	// 缓存奖品映射信息
	s.giftPrizeInfo[gift.ID] = prizeName + ":" + prizeLevel

	return &gift, nil
}

// GetGift 查询礼物
func (s *MemoryStore) GetGift(giftID string) (*model.GiftRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.giftRecords {
		if s.giftRecords[i].ID == giftID {
			return &s.giftRecords[i], nil
		}
	}
	return nil, fmt.Errorf("gift not found: %s", giftID)
}

// ReceiveGift 接收礼物（将道具添加到接收者库存）
func (s *MemoryStore) ReceiveGift(giftID string) (*model.ReceiveGiftResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i := range s.giftRecords {
		if s.giftRecords[i].ID == giftID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil, fmt.Errorf("gift not found: %s", giftID)
	}

	gift := &s.giftRecords[idx]
	if gift.Status != "sent" {
		return nil, fmt.Errorf("gift status is %s, cannot receive", gift.Status)
	}

	// 标记为已接收
	now := time.Now().UTC()
	gift.Status = "received"
	gift.ReceivedAt = &now

	// 添加道具到接收者库存
	newItem := model.UserInventory{
		ID:         "inv_" + randomSuffix(12),
		UserID:     gift.ReceiverID,
		PrizeID:    gift.PrizeID,
		PrizeName:  gift.PrizeName,
		PrizeLevel: gift.PrizeLevel,
		CampaignID: "",
		Source:     "gift",
		CreatedAt:  now,
	}
	s.inventory = append(s.inventory, newItem)

	// 从缓存中获取奖品信息
	info := s.giftPrizeInfo[giftID]
	parts := strings.SplitN(info, ":", 2)
	result := &model.ReceiveGiftResult{
		GiftID:    giftID,
		NewItemID: newItem.ID,
	}
	if len(parts) == 2 {
		result.PrizeName = parts[0]
		result.PrizeLevel = parts[1]
	} else {
		result.PrizeName = gift.PrizeName
		result.PrizeLevel = gift.PrizeLevel
	}

	return result, nil
}

// GetUserGifts 获取用户收到的礼物列表
func (s *MemoryStore) GetUserGifts(userID string) ([]model.GiftRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.GiftRecord, 0)
	for _, g := range s.giftRecords {
		if g.ReceiverID == userID && g.Status == "sent" {
			result = append(result, g)
		}
	}
	return result, nil
}

// GetUserSentGifts 获取用户送出的礼物列表
func (s *MemoryStore) GetUserSentGifts(userID string) ([]model.GiftRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.GiftRecord, 0)
	for _, g := range s.giftRecords {
		if g.GiverID == userID {
			result = append(result, g)
		}
	}
	return result, nil
}

// ExpireGift 过期礼物（将道具退回赠送者库存）
func (s *MemoryStore) ExpireGift(giftID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i := range s.giftRecords {
		if s.giftRecords[i].ID == giftID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("gift not found: %s", giftID)
	}

	gift := &s.giftRecords[idx]
	if gift.Status != "sent" {
		return fmt.Errorf("gift status is %s, cannot expire", gift.Status)
	}

	// 标记为已过期
	gift.Status = "expired"

	// 退还道具到赠送者库存
	s.inventory = append(s.inventory, model.UserInventory{
		ID:         "inv_" + randomSuffix(12),
		UserID:     gift.GiverID,
		PrizeID:    gift.PrizeID,
		PrizeName:  gift.PrizeName,
		PrizeLevel: gift.PrizeLevel,
		CampaignID: "",
		Source:     "gift_return",
		CreatedAt:  time.Now().UTC(),
	})

	return nil
}

// ============================================================
// 🆕 分享卡片系统
// ============================================================

// CreateShareCard 创建分享卡片
func (s *MemoryStore) CreateShareCard(userID, cardType, title, description, prizeName, prizeLevel, inviteLink string) (*model.ShareCard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.shareCardIDSeq++
	now := time.Now().UTC()

	card := model.ShareCard{
		ID:          "card_" + strconv.Itoa(s.shareCardIDSeq),
		UserID:      userID,
		CardType:    cardType,
		Title:       title,
		Description: description,
		PrizeName:   prizeName,
		PrizeLevel:  prizeLevel,
		InviteLink:  inviteLink,
		CreatedAt:   now,
	}
	s.shareCards[userID] = append(s.shareCards[userID], card)

	return &card, nil
}

// GetShareCards 获取用户的分享卡片列表
func (s *MemoryStore) GetShareCards(userID string) ([]model.ShareCard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cards := s.shareCards[userID]
	if cards == nil {
		return []model.ShareCard{}, nil
	}
	return cards, nil
}

// ============================================================
// 🆕 v1.6 碎片拼图 MemoryStore 实现
// ============================================================

// GetActivePuzzleTemplates 获取当前活跃的拼图模板列表
func (s *MemoryStore) GetActivePuzzleTemplates() []model.PuzzleTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []model.PuzzleTemplate
	for _, t := range s.puzzleTemplates {
		if t.IsActive {
			active = append(active, t)
		}
	}
	return active
}

// GetPuzzleTemplate 获取拼图模板详情
func (s *MemoryStore) GetPuzzleTemplate(templateID string) (*model.PuzzleTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.puzzleTemplates {
		if t.ID == templateID {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("puzzle template not found: %s", templateID)
}

// GetOrCreatePuzzleProgress 获取或创建用户拼图进度
func (s *MemoryStore) GetOrCreatePuzzleProgress(userID, templateID string) (*model.PuzzleProgress, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s", userID, templateID)
	progress, ok := s.puzzleProgresses[key]
	if ok {
		return progress, nil
	}

	// Find template to get total pieces
	var template *model.PuzzleTemplate
	for i := range s.puzzleTemplates {
		if s.puzzleTemplates[i].ID == templateID {
			template = &s.puzzleTemplates[i]
			break
		}
	}
	if template == nil {
		return nil, fmt.Errorf("puzzle template not found: %s", templateID)
	}

	progress = &model.PuzzleProgress{
		UserID:      userID,
		TemplateID:  templateID,
		Collected:   []int{},
		TotalPieces: template.TotalPieces,
		IsCompleted: false,
	}
	s.puzzleProgresses[key] = progress
	return progress, nil
}

// AddPuzzlePiece 收集碎片，返回是否是新收集
func (s *MemoryStore) AddPuzzlePiece(userID, templateID string, pieceIndex int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s", userID, templateID)
	progress, ok := s.puzzleProgresses[key]
	if !ok {
		return false, fmt.Errorf("puzzle progress not found for user=%s template=%s", userID, templateID)
	}

	// Check if already collected
	for _, idx := range progress.Collected {
		if idx == pieceIndex {
			return false, nil
		}
	}

	progress.Collected = append(progress.Collected, pieceIndex)
	return true, nil
}

// ComposePuzzle 合成拼图（集齐所有碎片后兑换奖励）
func (s *MemoryStore) ComposePuzzle(userID, templateID string) (*model.ComposePuzzleResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s", userID, templateID)
	progress, ok := s.puzzleProgresses[key]
	if !ok {
		return nil, fmt.Errorf("puzzle progress not found for user=%s template=%s", userID, templateID)
	}

	if progress.IsCompleted {
		return nil, fmt.Errorf("puzzle already completed")
	}

	// Find template
	var template *model.PuzzleTemplate
	for i := range s.puzzleTemplates {
		if s.puzzleTemplates[i].ID == templateID {
			template = &s.puzzleTemplates[i]
			break
		}
	}
	if template == nil {
		return nil, fmt.Errorf("puzzle template not found: %s", templateID)
	}

	// Check all pieces collected
	if len(progress.Collected) < template.TotalPieces {
		return nil, fmt.Errorf("not all pieces collected: have %d, need %d", len(progress.Collected), template.TotalPieces)
	}

	// Set completed
	now := time.Now().UTC()
	progress.IsCompleted = true
	progress.CompletedAt = &now

	result := &model.ComposePuzzleResult{
		TemplateID:   templateID,
		TemplateName: template.Name,
		RewardType:   template.RewardType,
		RewardName:   template.RewardName,
		RewardQty:    template.RewardQty,
	}

	// Award reward
	switch template.RewardType {
	case "points":
		member, ok := s.members[userID]
		if !ok {
			member = model.UserMember{UserID: userID, Level: model.MemberNormal, Points: 0, CreatedAt: now, UpdatedAt: now}
		}
		member.Points += int64(template.RewardQty)
		member.UpdatedAt = now
		s.members[userID] = member
		s.pointsLog = append(s.pointsLog, model.UserPointsLog{
			ID:        int64(len(s.pointsLog) + 1),
			UserID:    userID,
			Points:    int64(template.RewardQty),
			Balance:   member.Points,
			Reason:    "puzzle_compose",
			Remark:    "拼图合成奖励: " + template.Name,
			CreatedAt: now,
		})
	case "draw_ticket":
		s.addUserItemLocked(userID, model.ItemFreeDraw, template.RewardQty)
	case "prize":
		s.inventory = append(s.inventory, model.UserInventory{
			ID:         s.GenerateID(),
			UserID:     userID,
			PrizeID:    template.RewardID,
			PrizeName:  template.RewardName,
			PrizeLevel: "",
			CampaignID: template.CampaignID,
			Source:     "puzzle_compose",
			CreatedAt:  now,
		})
	}

	return result, nil
}

// GetPuzzleInfo 获取拼图详细信息（进度+模板）
func (s *MemoryStore) GetPuzzleInfo(userID, templateID string) (*model.PuzzleInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", userID, templateID)
	progress, ok := s.puzzleProgresses[key]
	if !ok {
		return nil, fmt.Errorf("puzzle progress not found for user=%s template=%s", userID, templateID)
	}

	var template *model.PuzzleTemplate
	for i := range s.puzzleTemplates {
		if s.puzzleTemplates[i].ID == templateID {
			template = &s.puzzleTemplates[i]
			break
		}
	}
	if template == nil {
		return nil, fmt.Errorf("puzzle template not found: %s", templateID)
	}

	collectedSet := make(map[int]bool)
	for _, idx := range progress.Collected {
		collectedSet[idx] = true
	}

	var collectedNames []string
	var missingNames []string
	for i, name := range template.PieceNames {
		if collectedSet[i] {
			collectedNames = append(collectedNames, name)
		} else {
			missingNames = append(missingNames, name)
		}
	}

	progressPercent := 0.0
	if template.TotalPieces > 0 {
		progressPercent = float64(len(progress.Collected)) / float64(template.TotalPieces) * 100
	}

	return &model.PuzzleInfo{
		Template:        template,
		Progress:        progress,
		CollectedNames:  collectedNames,
		MissingNames:    missingNames,
		ProgressPercent: progressPercent,
	}, nil
}

// CreatePuzzleTeam 创建拼图小队
func (s *MemoryStore) CreatePuzzleTeam(captainID, templateID string) (*model.PuzzleTeam, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify template exists
	var totalPieces int
	found := false
	for _, t := range s.puzzleTemplates {
		if t.ID == templateID {
			totalPieces = t.TotalPieces
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("puzzle template not found: %s", templateID)
	}

	s.puzzleTeamIDSeq++
	now := time.Now().UTC()
	team := &model.PuzzleTeam{
		ID:          s.GenerateID(),
		TemplateID:  templateID,
		CaptainID:   captainID,
		Members:     []string{captainID},
		Shared:      []int{},
		TotalPieces: totalPieces,
		IsCompleted: false,
		CreatedAt:   now,
	}
	s.puzzleTeams[team.ID] = team
	return team, nil
}

// JoinPuzzleTeam 加入拼图小队
func (s *MemoryStore) JoinPuzzleTeam(userID, teamID string) (*model.PuzzleTeam, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.puzzleTeams[teamID]
	if !ok {
		return nil, fmt.Errorf("puzzle team not found: %s", teamID)
	}

	// Check if already a member
	for _, m := range team.Members {
		if m == userID {
			return team, nil
		}
	}

	team.Members = append(team.Members, userID)
	return team, nil
}

// GetPuzzleTeam 获取拼图小队信息
func (s *MemoryStore) GetPuzzleTeam(teamID string) (*model.PuzzleTeam, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	team, ok := s.puzzleTeams[teamID]
	if !ok {
		return nil, fmt.Errorf("puzzle team not found: %s", teamID)
	}
	return team, nil
}

// GetUserPuzzleTeams 获取用户加入的拼图小队
func (s *MemoryStore) GetUserPuzzleTeams(userID string) ([]model.PuzzleTeam, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var teams []model.PuzzleTeam
	for _, team := range s.puzzleTeams {
		for _, m := range team.Members {
			if m == userID {
				teams = append(teams, *team)
				break
			}
		}
	}
	return teams, nil
}

// GetUserPuzzleProgresses 获取用户所有拼图进度
func (s *MemoryStore) GetUserPuzzleProgresses(userID string) ([]model.PuzzleInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prefix := userID + ":"
	var infos []model.PuzzleInfo
	for key, progress := range s.puzzleProgresses {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		// Find matching template
		for _, t := range s.puzzleTemplates {
			if t.ID == progress.TemplateID {
				collectedSet := make(map[int]bool)
				for _, idx := range progress.Collected {
					collectedSet[idx] = true
				}
				var collectedNames, missingNames []string
				for i, name := range t.PieceNames {
					if collectedSet[i] {
						collectedNames = append(collectedNames, name)
					} else {
						missingNames = append(missingNames, name)
					}
				}
				progressPercent := 0.0
				if t.TotalPieces > 0 {
					progressPercent = float64(len(progress.Collected)) / float64(t.TotalPieces) * 100
				}
				infos = append(infos, model.PuzzleInfo{
					Template:        &t,
					Progress:        progress,
					CollectedNames:  collectedNames,
					MissingNames:    missingNames,
					ProgressPercent: progressPercent,
				})
				break
			}
		}
	}
	return infos, nil
}

// SharePuzzlePiece 在拼图小队中共享碎片
func (s *MemoryStore) SharePuzzlePiece(userID, teamID string, pieceIndex int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.puzzleTeams[teamID]
	if !ok {
		return false, fmt.Errorf("puzzle team not found: %s", teamID)
	}

	// Verify user is a member
	isMember := false
	for _, m := range team.Members {
		if m == userID {
			isMember = true
			break
		}
	}
	if !isMember {
		return false, fmt.Errorf("user %s is not a member of team %s", userID, teamID)
	}

	// Check if piece already shared
	for _, idx := range team.Shared {
		if idx == pieceIndex {
			return false, nil
		}
	}

	team.Shared = append(team.Shared, pieceIndex)
	return true, nil
}

// ============================================================
// 🆕 v1.6 预约抢购 MemoryStore 实现
// ============================================================

// GetFlashSales 获取抢购活动列表
func (s *MemoryStore) GetFlashSales() []model.FlashSale {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.FlashSale, len(s.flashSales))
	copy(result, s.flashSales)
	return result
}

// GetFlashSale 获取抢购活动详情
func (s *MemoryStore) GetFlashSale(flashID string) (*model.FlashSale, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.flashSales {
		if s.flashSales[i].ID == flashID {
			return &s.flashSales[i], nil
		}
	}
	return nil, fmt.Errorf("flash sale not found: %s", flashID)
}

// SubscribeFlash 预约抢购活动
func (s *MemoryStore) SubscribeFlash(userID, flashID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify flash exists
	found := false
	for _, f := range s.flashSales {
		if f.ID == flashID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("flash sale not found: %s", flashID)
	}

	// Check if already subscribed
	for _, sub := range s.flashSubscriptions {
		if sub.UserID == userID && sub.FlashID == flashID {
			return nil // already subscribed
		}
	}

	s.flashSubscriptions = append(s.flashSubscriptions, model.FlashSubscription{
		UserID:    userID,
		FlashID:   flashID,
		CreatedAt: time.Now().UTC(),
	})
	return nil
}

// UnsubscribeFlash 取消预约
func (s *MemoryStore) UnsubscribeFlash(userID, flashID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, sub := range s.flashSubscriptions {
		if sub.UserID == userID && sub.FlashID == flashID {
			s.flashSubscriptions = append(s.flashSubscriptions[:i], s.flashSubscriptions[i+1:]...)
			return nil
		}
	}
	return nil
}

// IsFlashSubscribed 检查用户是否已预约
func (s *MemoryStore) IsFlashSubscribed(userID, flashID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, sub := range s.flashSubscriptions {
		if sub.UserID == userID && sub.FlashID == flashID {
			return true, nil
		}
	}
	return false, nil
}

// PurchaseFlash 执行抢购（扣积分减库存）
func (s *MemoryStore) PurchaseFlash(userID, flashID string) (*model.FlashPurchaseResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find flash sale
	var flash *model.FlashSale
	for i := range s.flashSales {
		if s.flashSales[i].ID == flashID {
			flash = &s.flashSales[i]
			break
		}
	}
	if flash == nil {
		return &model.FlashPurchaseResult{
			FlashID: flashID, Success: false, Message: "抢购活动不存在",
		}, fmt.Errorf("flash sale not found: %s", flashID)
	}

	// Check stock
	if flash.RemainingStock <= 0 {
		return &model.FlashPurchaseResult{
			FlashID: flashID, FlashName: flash.Name, Success: false, Message: "库存不足",
		}, fmt.Errorf("flash sale out of stock: %s", flashID)
	}

	// Get or create member
	member, ok := s.members[userID]
	if !ok {
		member = model.UserMember{
			UserID:    userID,
			Level:     model.MemberNormal,
			Points:    0,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
	}

	// Check eligibility: member level >= MinVipLevel
	if member.Level < model.MemberLevel(flash.MinVipLevel) && flash.MinVipLevel != "" {
		return &model.FlashPurchaseResult{
			FlashID: flashID, FlashName: flash.Name, Success: false, Message: "会员等级不足",
		}, fmt.Errorf("insufficient vip level: need %s, have %s", flash.MinVipLevel, member.Level)
	}

	// Check total draws >= MinTotalDraws
	if member.TotalDraws < flash.MinTotalDraws {
		return &model.FlashPurchaseResult{
			FlashID: flashID, FlashName: flash.Name, Success: false, Message: "抽奖次数不足",
		}, fmt.Errorf("insufficient total draws: need %d, have %d", flash.MinTotalDraws, member.TotalDraws)
	}

	// Check points
	if member.Points < flash.PricePoints {
		return &model.FlashPurchaseResult{
			FlashID: flashID, FlashName: flash.Name, Success: false, Message: "积分不足",
		}, fmt.Errorf("insufficient points: need %d, have %d", flash.PricePoints, member.Points)
	}

	// Deduct points
	now := time.Now().UTC()
	member.Points -= flash.PricePoints
	member.UpdatedAt = now
	s.members[userID] = member

	// Decrement stock
	flash.RemainingStock--

	// Log points
	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID:        int64(len(s.pointsLog) + 1),
		UserID:    userID,
		Points:    -flash.PricePoints,
		Balance:   member.Points,
		Reason:    "flash_purchase",
		Remark:    "抢购: " + flash.Name,
		CreatedAt: now,
	})

	// Add to inventory
	s.inventory = append(s.inventory, model.UserInventory{
		ID:         s.GenerateID(),
		UserID:     userID,
		PrizeID:    "",
		PrizeName:  flash.Name,
		PrizeLevel: "",
		CampaignID: flash.CampaignID,
		Source:     "flash",
		CreatedAt:  now,
	})

	return &model.FlashPurchaseResult{
		FlashID:   flashID,
		FlashName: flash.Name,
		Success:   true,
		Message:   "抢购成功",
	}, nil
}

// GetUserFlashSubscriptions 获取用户预约列表
func (s *MemoryStore) GetUserFlashSubscriptions(userID string) ([]model.FlashSubscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var subs []model.FlashSubscription
	for _, sub := range s.flashSubscriptions {
		if sub.UserID == userID {
			subs = append(subs, sub)
		}
	}
	return subs, nil
}

// CreateFlashSale 创建抢购活动（管理端）
func (s *MemoryStore) CreateFlashSale(input model.FlashSale) (*model.FlashSale, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.flashIDSeq++
	now := time.Now().UTC()
	flash := &model.FlashSale{
		ID:             "flash_" + strconv.Itoa(s.flashIDSeq),
		CampaignID:     input.CampaignID,
		Name:           input.Name,
		Description:    input.Description,
		PricePoints:    input.PricePoints,
		TotalStock:     input.TotalStock,
		RemainingStock: input.TotalStock,
		MinVipLevel:    input.MinVipLevel,
		MinTotalDraws:  input.MinTotalDraws,
		StartAt:        input.StartAt,
		EndAt:          input.EndAt,
		Status:         "upcoming",
		CreatedAt:      now,
	}
	s.flashSales = append(s.flashSales, *flash)
	return flash, nil
}

// UpdateFlashSaleStatus 更新抢购状态
func (s *MemoryStore) UpdateFlashSaleStatus(flashID, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.flashSales {
		if s.flashSales[i].ID == flashID {
			s.flashSales[i].Status = status
			return nil
		}
	}
	return fmt.Errorf("flash sale not found: %s", flashID)
}

// ============================================================
// 🆕 活动系统实现
// ============================================================

// seedActivities 初始化默认活动
func (s *MemoryStore) seedActivities() {
	now := time.Now().UTC()

	// 活动1: UP池
	act1 := model.Activity{
		ID:        "v2_weekly_up",
		Name:      "每周UP·星月传说",
		Type:      model.ActivityUPPool,
		Rules: model.ActivityRules{
			UPPrizeID:    "prize_001",
			UPMultiplier: 5.0,
			UPLevel:      "rare",
			UPCampaignID: "camp_launch_001",
		},
		StartAt:   now.Add(-1 * time.Hour),
		EndAt:     now.Add(7 * 24 * time.Hour),
		Status:    "active",
		SortOrder: 1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.activities = append(s.activities, act1)

	// 活动2: 签到双倍
	act2 := model.Activity{
		ID:        "v2_checkin_boost",
		Name:      "周末签到双倍",
		Type:      model.ActivityCheckinBoost,
		Rules: model.ActivityRules{
			CheckinMultiplier: 2,
		},
		StartAt:   now,
		EndAt:     now.Add(48 * time.Hour),
		Status:    "active",
		SortOrder: 2,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.activities = append(s.activities, act2)
	s.activityRewards = append(s.activityRewards, model.ActivityReward{
		ID:         "reward_ck_1",
		ActivityID: act2.ID,
		Condition:  "活动期间签到3天",
		RewardType: "points",
		RewardQty:  50,
		RewardName: "签到奖励积分",
	})

	// 活动3: 限时折扣
	act3 := model.Activity{
		ID:        "v2_discount",
		Name:      "限时8折十连",
		Type:      model.ActivityDiscount,
		Rules: model.ActivityRules{
			DiscountRate:   0.8,
			DiscountTarget: "ten_draw",
		},
		StartAt:   now,
		EndAt:     now.Add(72 * time.Hour),
		Status:    "active",
		SortOrder: 3,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.activities = append(s.activities, act3)
	s.activityRewards = append(s.activityRewards, model.ActivityReward{
		ID:         "reward_dc_1",
		ActivityID: act3.ID,
		Condition:  "活动期间十连3次",
		RewardType: "draw_ticket",
		RewardQty:  1,
		RewardName: "免费抽奖券",
	})
}

// GetActiveActivities 获取所有进行中的活动（时间范围内且状态为active）
func (s *MemoryStore) GetActiveActivities() ([]model.Activity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	var active []model.Activity
	for _, a := range s.activities {
		if a.Status == "active" && !now.Before(a.StartAt) && !now.After(a.EndAt) {
			active = append(active, a)
		}
	}
	return active, nil
}

// GetAllActivities 返回所有活动
func (s *MemoryStore) GetAllActivities() ([]model.Activity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.Activity, len(s.activities))
	copy(result, s.activities)
	return result, nil
}

// GetActivity 根据ID获取活动
func (s *MemoryStore) GetActivity(activityID string) (*model.Activity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, a := range s.activities {
		if a.ID == activityID {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("activity not found: %s", activityID)
}

// CreateActivity 创建活动
func (s *MemoryStore) CreateActivity(input model.ActivityCreateRequest) (*model.Activity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.activityIDSeq++
	now := time.Now().UTC()
	act := &model.Activity{
		ID:          "act_" + strconv.Itoa(s.activityIDSeq),
		Name:        input.Name,
		Description: input.Description,
		Type:        input.Type,
		BannerURL:   input.BannerURL,
		Rules:       input.Rules,
		SortOrder:   input.SortOrder,
		Status:      "draft",
		StartAt:     input.StartAt,
		EndAt:       input.EndAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.activities = append(s.activities, *act)
	return act, nil
}

// UpdateActivity 更新活动
func (s *MemoryStore) UpdateActivity(activityID string, input model.ActivityUpdateRequest) (*model.Activity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.activities {
		if s.activities[i].ID == activityID {
			if input.Name != "" {
				s.activities[i].Name = input.Name
			}
			if input.Description != "" {
				s.activities[i].Description = input.Description
			}
			if input.BannerURL != "" {
				s.activities[i].BannerURL = input.BannerURL
			}
			if input.Rules != nil {
				s.activities[i].Rules = *input.Rules
			}
			if input.SortOrder != nil {
				s.activities[i].SortOrder = *input.SortOrder
			}
			if input.Status != "" {
				s.activities[i].Status = input.Status
			}
			if input.StartAt != nil {
				s.activities[i].StartAt = *input.StartAt
			}
			if input.EndAt != nil {
				s.activities[i].EndAt = *input.EndAt
			}
			s.activities[i].UpdatedAt = time.Now().UTC()
			return &s.activities[i], nil
		}
	}
	return nil, fmt.Errorf("activity not found: %s", activityID)
}

// DeleteActivity 删除活动
func (s *MemoryStore) DeleteActivity(activityID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.activities {
		if s.activities[i].ID == activityID {
			s.activities = append(s.activities[:i], s.activities[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("activity not found: %s", activityID)
}

// GetActivityRewards 获取活动的奖励列表
func (s *MemoryStore) GetActivityRewards(activityID string) ([]model.ActivityReward, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rewards []model.ActivityReward
	for _, r := range s.activityRewards {
		if r.ActivityID == activityID {
			rewards = append(rewards, r)
		}
	}
	return rewards, nil
}

// JoinActivity 用户参与活动
func (s *MemoryStore) JoinActivity(userID, activityID string) (*model.ActivityParticipation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	part := &model.ActivityParticipation{
		ID:            s.GenerateID(),
		UserID:        userID,
		ActivityID:    activityID,
		RewardClaimed: false,
		JoinedAt:      now,
	}
	s.activityParticipations = append(s.activityParticipations, *part)
	return part, nil
}

// GetUserActivityParticipation 获取用户在特定活动的参与记录
func (s *MemoryStore) GetUserActivityParticipation(userID, activityID string) (*model.ActivityParticipation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.activityParticipations {
		if p.UserID == userID && p.ActivityID == activityID {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("activity participation not found: user=%s, activity=%s", userID, activityID)
}

// GetUserActivityParticipations 获取用户的所有活动参与记录
func (s *MemoryStore) GetUserActivityParticipations(userID string) ([]model.ActivityParticipation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var parts []model.ActivityParticipation
	for _, p := range s.activityParticipations {
		if p.UserID == userID {
			parts = append(parts, p)
		}
	}
	return parts, nil
}

// ClaimActivityReward 领取活动奖励
func (s *MemoryStore) ClaimActivityReward(userID, activityID, rewardID string) (*model.ActivityReward, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 查找参与记录
	var part *model.ActivityParticipation
	for i := range s.activityParticipations {
		if s.activityParticipations[i].UserID == userID && s.activityParticipations[i].ActivityID == activityID {
			part = &s.activityParticipations[i]
			break
		}
	}
	if part == nil {
		return nil, fmt.Errorf("activity participation not found: user=%s, activity=%s", userID, activityID)
	}

	// 查找奖励
	var reward *model.ActivityReward
	for i := range s.activityRewards {
		if s.activityRewards[i].ID == rewardID && s.activityRewards[i].ActivityID == activityID {
			reward = &s.activityRewards[i]
			break
		}
	}
	if reward == nil {
		return nil, fmt.Errorf("activity reward not found: %s", rewardID)
	}

	// 标记已领取
	part.RewardClaimed = true

	// 如果奖励类型是积分，同步增加用户积分
	if reward.RewardType == "points" {
		member, exists := s.members[userID]
		if !exists {
			return nil, fmt.Errorf("member not found: %s", userID)
		}
		member.Points += reward.RewardQty
		member.UpdatedAt = time.Now().UTC()
		s.members[userID] = member
	}

	return reward, nil
}
