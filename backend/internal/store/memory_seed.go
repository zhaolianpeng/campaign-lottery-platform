package store

import (
	"time"

	"campaign-lottery-platform/backend/internal/model"
)

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

// seedActivities 初始化默认活动
func (s *MemoryStore) seedActivities() {
	now := time.Now().UTC()

	// 活动1: UP池
	act1 := model.Activity{
		ID:   "v2_weekly_up",
		Name: "每周UP·星月传说",
		Type: model.ActivityUPPool,
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
		ID:   "v2_checkin_boost",
		Name: "周末签到双倍",
		Type: model.ActivityCheckinBoost,
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
		ID:   "v2_discount",
		Name: "限时8折十连",
		Type: model.ActivityDiscount,
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
