package model

import "time"

// ============================================================
// 原有模型（保留全部）
// ============================================================

type User struct {
	ID        string    `json:"id"`
	Nickname  string    `json:"nickname"`
	CreatedAt time.Time `json:"created_at"`
}

type Session struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Campaign struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Slug            string    `json:"slug"`
	Status          string    `json:"status"`
	StartsAt        time.Time `json:"starts_at"`
	EndsAt          time.Time `json:"ends_at"`
	DailyDrawLimit  int       `json:"daily_draw_limit"`
	MissWeight      int       `json:"miss_weight"`
	BannerImageURL  string    `json:"banner_image_url"`
	CampaignSummary string    `json:"campaign_summary"`
	PityConfig      PityConfig `json:"pity_config,omitempty"`
}

type Prize struct {
	ID                string `json:"id"`
	CampaignID        string `json:"campaign_id"`
	Name              string `json:"name"`
	Level             string `json:"level"`
	Stock             int    `json:"stock"`
	ProbabilityWeight int    `json:"probability_weight"`
	Status            string `json:"status"`
}

type DrawRecord struct {
	ID          string    `json:"id"`
	CampaignID  string    `json:"campaign_id"`
	UserID      string    `json:"user_id"`
	PrizeID     *string   `json:"prize_id,omitempty"`
	PrizeName   string    `json:"prize_name"`
	Result      string    `json:"result"`
	DrawnAt     time.Time `json:"drawn_at"`
	ChanceAfter int       `json:"chance_after"`
}

type AdminOverview struct {
	TotalUsers      int            `json:"total_users"`
	TotalDraws      int            `json:"total_draws"`
	TotalWins       int            `json:"total_wins"`
	Campaigns       []Campaign     `json:"campaigns"`
	PrizeSummaries  []PrizeSummary `json:"prize_summaries"`
	RecentDraws     []DrawRecord   `json:"recent_draws"`
	UserDrawBalance map[string]int `json:"user_draw_balance"`
}

type PrizeSummary struct {
	PrizeID    string `json:"prize_id"`
	PrizeName  string `json:"prize_name"`
	PrizeLevel string `json:"prize_level"`
	Stock      int    `json:"stock"`
}

type FulfillmentTask struct {
	ID           int64      `json:"id"`
	DrawRecordID string     `json:"draw_record_id"`
	UserID       string     `json:"user_id"`
	PrizeID      string     `json:"prize_id"`
	Status       string     `json:"status"`
	PayloadJSON  string     `json:"payload_json"`
	OperatorNote string     `json:"operator_note"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	FulfilledAt  *time.Time `json:"fulfilled_at,omitempty"`
}

type CampaignMutation struct {
	Name            string    `json:"name"`
	Slug            string    `json:"slug"`
	Status          string    `json:"status"`
	StartsAt        time.Time `json:"starts_at"`
	EndsAt          time.Time `json:"ends_at"`
	DailyDrawLimit  int       `json:"daily_draw_limit"`
	MissWeight      int       `json:"miss_weight"`
	BannerImageURL  string    `json:"banner_image_url"`
	CampaignSummary string    `json:"campaign_summary"`
	PityConfig      PityConfig `json:"pity_config,omitempty"`
}

type PrizeMutation struct {
	Name              string `json:"name"`
	Level             string `json:"level"`
	Stock             int    `json:"stock"`
	ProbabilityWeight int    `json:"probability_weight"`
	Status            string `json:"status"`
}

type FulfillmentTaskMutation struct {
	Status       string `json:"status"`
	OperatorNote string `json:"operator_note"`
}

type DrawResult struct {
	Record           DrawRecord `json:"record"`
	RemainingChances int        `json:"remaining_chances"`
}

// ============================================================
// 盲盒扩展模型
// ============================================================

// PrizeLevel 礼品等级枚举
const (
	PrizeLevelCommon  = "common"  // 普通款
	PrizeLevelRare    = "rare"    // 稀有款
	PrizeLevelSecret  = "secret"  // 隐藏款
	PrizeLevelLimited = "limited" // 限定款
)

// PityConfig 保底概率配置（存储在 Campaign 中或独立配置表）
type PityConfig struct {
	Enabled      bool    `json:"enabled"`       // 是否启用保底
	SoftPityN    int     `json:"soft_pity_n"`   // 软保底开始递增的次数
	PityFactor   float64 `json:"pity_factor"`   // 概率递增因子 α
	HardPityN    int     `json:"hard_pity_n"`   // 硬保底次数（必出）
	TargetPrize  string  `json:"target_prize"`  // 保底目标奖品ID（通常是隐藏款）

	// 🆕 UP池配置（限时概率提升 + 50/50 大小保底）
	UPPoolEnabled bool      `json:"up_pool_enabled"`  // 是否启用UP池
	UPPrizeID     string    `json:"up_prize_id"`      // UP目标奖品ID
	UPMultiplier  float64   `json:"up_multiplier"`    // 概率提升倍数（如 5 = 5倍概率）
	UPLevel       string    `json:"up_level"`         // UP目标等级（rare / secret / limited）
	UPStartAt     time.Time `json:"up_start_at"`      // UP池开始时间
	UPEndAt       time.Time `json:"up_end_at"`        // UP池结束时间
}

// BlindBoxCampaign 盲盒系列（扩展 Campaign）
type BlindBoxCampaign struct {
	Campaign
	PityConfig     PityConfig `json:"pity_config"`     // 概率保底配置
	SeriesImageURL string     `json:"series_image_url"` // 系列主图
	SeriesItemCount int       `json:"series_item_count"`// 系列总款式数
	DrawPrice       int       `json:"draw_price"`       // 单抽价格（分）
	TenDrawPrice    int       `json:"ten_draw_price"`   // 十连抽价格（分）
}

// BlindBoxPrize 盲盒礼品（扩展 Prize）
type BlindBoxPrize struct {
	Prize
	ImageURL    string  `json:"image_url"`     // 礼品图片
	SortOrder   int     `json:"sort_order"`    // 排序
	DisplayProb string  `json:"display_prob"`  // 对外公示概率（如 "7.0%"）
}

// DrawConfig 抽奖配置请求体
type DrawConfig struct {
	CampaignID string `json:"campaign_id"`
	DrawCount  int    `json:"draw_count"` // 1=单抽, >=2=十连抽
}

// PityStatus 保底状态返回
type PityStatus struct {
	ConsecutiveMisses int     `json:"consecutive_misses"`  // 连续未中次数
	PityMultiplier    float64 `json:"pity_multiplier"`     // 当前概率倍数
	SoftPityN         int     `json:"soft_pity_n"`         // 软保底触发次数
	HardPityN         int     `json:"hard_pity_n"`         // 硬保底次数
	MissesToHardPity  int     `json:"misses_to_hard_pity"` // 距离硬保底还差几次
	HasUPPoolGuarantee bool   `json:"has_up_pool_guarantee,omitempty"` // 🆕 是否持有大保底
}

// BlindBoxDrawResult 盲盒抽奖结果
type BlindBoxDrawResult struct {
	Draws            []SingleDrawResult `json:"draws"`             // 本次抽奖结果列表
	RemainingChances int                `json:"remaining_chances"` // 剩余抽奖次数
	PityStatus       *PityStatus        `json:"pity_status,omitempty"` // 当前保底状态
	CollectionReward *CollectionReward  `json:"collection_reward,omitempty"` // 集齐奖励（如触发）
}

// SingleDrawResult 单次抽奖结果
type SingleDrawResult struct {
	RecordID      string `json:"record_id"`
	PrizeID       string `json:"prize_id,omitempty"`
	PrizeName     string `json:"prize_name"`
	PrizeLevel    string `json:"prize_level"`
	PrizeImageURL string `json:"prize_image_url,omitempty"`
	IsWin         bool   `json:"is_win"`
	IsHardPity    bool   `json:"is_hard_pity,omitempty"`
	IsNew         bool   `json:"is_new,omitempty"`
	IsUPPoolWin   bool   `json:"is_up_pool_win,omitempty"`  // 🆕 是否是UP池中奖
}

// SeriesProgress 用户系列收集进度
type SeriesProgress struct {
	CampaignID        string              `json:"campaign_id"`
	CampaignName      string              `json:"campaign_name"`
	TotalItems        int                 `json:"total_items"`        // 系列总款式数
	CollectedItems    int                 `json:"collected_items"`    // 已收集款式数
	ProgressPercent   float64             `json:"progress_percent"`   // 收集进度百分比
	Duplicates        int                 `json:"duplicates"`         // 重复款数量
	CollectedPrizes   []CollectedPrize    `json:"collected_prizes"`   // 已收集详情
	MissingPrizes     []PrizeSummary      `json:"missing_prizes"`     // 缺失款式
}

// CollectedPrize 已收集的奖品
type CollectedPrize struct {
	Prize
	Count int `json:"count"` // 拥有数量（>=1）
}

// ExchangeOffer 交换市场挂单
type ExchangeOffer struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	UserNickname string    `json:"user_nickname,omitempty"`
	HavePrizeID  string    `json:"have_prize_id"`
	HavePrizeName string   `json:"have_prize_name"`
	WantPrizeID  string    `json:"want_prize_id"`
	WantPrizeName string   `json:"want_prize_name"`
	Status       string    `json:"status"` // pending / matched / completed / cancelled
	CreatedAt    time.Time `json:"created_at"`
}

// ExchangeOfferMutation 交换挂单请求体
type ExchangeOfferMutation struct {
	HavePrizeID  string `json:"have_prize_id"`
	WantPrizeID  string `json:"want_prize_id"`
}

// ExchangeMatch 交换匹配请求体
type ExchangeMatchRequest struct {
	OfferID string `json:"offer_id"`
}

// UserInventory 用户库存条目
type UserInventory struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	PrizeID    string    `json:"prize_id"`
	PrizeName  string    `json:"prize_name"`
	PrizeLevel string    `json:"prize_level"`
	CampaignID string    `json:"campaign_id"`
	Source     string    `json:"source"` // draw / exchange / redeem
	CreatedAt  time.Time `json:"created_at"`
}

// MemberLevel 会员等级
type MemberLevel string

const (
	MemberNormal  MemberLevel = "normal"
	MemberSilver  MemberLevel = "silver"
	MemberGold    MemberLevel = "gold"
	MemberDiamond MemberLevel = "diamond"
)

// UserMember 用户会员信息
type UserMember struct {
	UserID      string      `json:"user_id"`
	Level       MemberLevel `json:"level"`
	Points      int64       `json:"points"`      // 积分
	TotalDraws  int64       `json:"total_draws"` // 累计抽奖次数
	TotalSpent  int64       `json:"total_spent"` // 累计消费（分）
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// ============================================================
// 月卡/付费卡系统
// ============================================================

// CardType 卡类型
type CardType string

const (
	CardWeekly  CardType = "weekly"  // 周卡
	CardMonthly CardType = "monthly" // 月卡
	CardSeason  CardType = "season"  // 季卡
)

// CardConfig 卡配置
type CardConfig struct {
	CardType       CardType `json:"card_type"`
	Price          int      `json:"price"`           // 价格（分）
	DurationDays   int      `json:"duration_days"`   // 有效期天数
	FreeDrawsDaily int      `json:"free_draws_daily"` // 每日免费抽次数
	DiscountRate   float64  `json:"discount_rate"`    // 折扣率 0.9=9折
	Description    string   `json:"description"`
}

// 卡配置表
var CardConfigs = map[CardType]CardConfig{
	CardWeekly:  {CardType: CardWeekly, Price: 990, DurationDays: 7, FreeDrawsDaily: 1, DiscountRate: 0.9, Description: "周卡·每日免费1抽+9折"},
	CardMonthly: {CardType: CardMonthly, Price: 2800, DurationDays: 30, FreeDrawsDaily: 2, DiscountRate: 0.8, Description: "月卡·每日免费2抽+8折"},
	CardSeason:  {CardType: CardSeason, Price: 6800, DurationDays: 90, FreeDrawsDaily: 2, DiscountRate: 0.75, Description: "季卡·每日免费2抽+7.5折+限定优先"},
}

// UserCard 用户已购卡
type UserCard struct {
	UserID        string    `json:"user_id"`
	CardType      CardType  `json:"card_type"`
	StartedAt     time.Time `json:"started_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	DailyFreeUsed int       `json:"daily_free_used"` // 今日已用免费次数
	FreeDate      string    `json:"free_date"`       // 记录免费次数日期 "2006-01-02"
}

// BuyCardRequest 购买卡请求
type BuyCardRequest struct {
	CardType CardType `json:"card_type"`
}

// BuyCardResult 购买结果
type BuyCardResult struct {
	CardType  CardType `json:"card_type"`
	ExpiresAt string   `json:"expires_at"`
	Price     int      `json:"price"`
	Points    int64    `json:"points"` // 扣减后剩余积分
}

// UserPointsLog 积分变动记录
type UserPointsLog struct {
	ID        int64     `json:"id"`
	UserID    string    `json:"user_id"`
	Points    int64     `json:"points"`    // 变动数量（正=增加，负=消耗）
	Balance   int64     `json:"balance"`   // 变动后余额
	Reason    string    `json:"reason"`    // draw / exchange / daily / redeem
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"created_at"`
}

// RedeemRequest 积分兑换请求
type RedeemRequest struct {
	PrizeID string `json:"prize_id"`
}

// RedeemResult 兑换结果
type RedeemResult struct {
	RecordID   string `json:"record_id"`
	PrizeID    string `json:"prize_id"`
	PrizeName  string `json:"prize_name"`
	PointsCost int64  `json:"points_cost"`
	Remaining  int64  `json:"remaining"`
}

// ============================================================
// 集卡系统扩展模型
// ============================================================

// CheckInResult 每日签到结果
type CheckInResult struct {
	PointsAwarded int64 `json:"points_awarded"`  // 本次获得积分
	StreakDays    int   `json:"streak_days"`      // 连续签到天数
	IsBonus       bool  `json:"is_bonus"`         // 是否触发连续奖励
	NewBalance    int64 `json:"new_balance"`      // 变动后余额
}

// CollectionReward 集齐系列奖励
type CollectionReward struct {
	CampaignID   string `json:"campaign_id"`
	CampaignName string `json:"campaign_name"`
	RewardType   string `json:"reward_type"`  // hidden_prize / title / points
	RewardName   string `json:"reward_name"`
	RewardPrizeID string `json:"reward_prize_id,omitempty"`
	Description  string `json:"description"`
}

// LeaderboardEntry 排行榜条目
type LeaderboardEntry struct {
	Rank            int     `json:"rank"`
	UserID          string  `json:"user_id"`
	Nickname        string  `json:"nickname"`
	CollectedCount  int     `json:"collected_count"`   // 已收集总款式数
	TotalCount      int     `json:"total_count"`       // 总款式数
	ProgressPercent float64 `json:"progress_percent"`  // 收集进度百分比
	SeriesCompleted int     `json:"series_completed"`  // 完整集齐系列数
}

// HintMessage 摇盒提示文案
type HintMessage struct {
	Type    string `json:"type"`    // hot / social / luck
	Content string `json:"content"` // 提示文案
}

// ShareRewardResult 分享奖励结果
type ShareRewardResult struct {
	PointsAwarded int64 `json:"points_awarded"`
	DailyLeft     int   `json:"daily_left"`  // 今日还可分享次数
	NewBalance    int64 `json:"new_balance"`
}

// DrawStatistics 抽奖统计
type DrawStatistics struct {
	TotalDraws      int64              `json:"total_draws"`
	TotalUsers      int64              `json:"total_users"`
	TotalWins       int64              `json:"total_wins"`
	WinRate         float64            `json:"win_rate"`
	PrizeBreakdown  []PrizeStatItem    `json:"prize_breakdown"`
	DailyDraws      []DailyDrawStat    `json:"daily_draws,omitempty"`
}

// PrizeStatItem 奖品统计项
type PrizeStatItem struct {
	PrizeID   string `json:"prize_id"`
	PrizeName string `json:"prize_name"`
	Level     string `json:"level"`
	Count     int64  `json:"count"`
	Percent   float64 `json:"percent"`
}

// DailyDrawStat 每日抽奖统计
type DailyDrawStat struct {
	Date  string `json:"date"` // "2006-01-02"
	Count int64  `json:"count"`
	Wins  int64  `json:"wins"`
}

// ============================================================
// 月卡/战令系统模型 🆕
// ============================================================

// MonthCardType 月卡类型
type MonthCardType string

const (
	MonthCardWeekly  MonthCardType = "weekly"  // 周卡 9.9元
	MonthCardMonthly MonthCardType = "monthly" // 月卡 28元
	MonthCardSeason  MonthCardType = "season"  // 季卡 68元
)

// MonthCard 用户月卡信息
type MonthCard struct {
	ID          string        `json:"id"`
	UserID      string        `json:"user_id"`
	CardType    MonthCardType `json:"card_type"`
	Price       int64         `json:"price"`         // 购买价格（分）
	FreeDraws   int           `json:"free_draws"`    // 每日免费抽次数
	DrawDiscount float64      `json:"draw_discount"` // 折扣（0.8 = 8折）
	StartedAt   time.Time     `json:"started_at"`
	ExpiresAt   time.Time     `json:"expires_at"`
	CreatedAt   time.Time     `json:"created_at"`
}

// MonthCardPurchaseRequest 购买月卡请求
type MonthCardPurchaseRequest struct {
	CardType MonthCardType `json:"card_type"` // weekly / monthly / season
}

// MonthCardPurchaseResult 购买月卡结果
type MonthCardPurchaseResult struct {
	Card      MonthCard `json:"card"`
	NewPoints int64     `json:"new_points"`
}

// MonthCardStatus 月卡状态（给前端查询）
type MonthCardStatus struct {
	HasCard      bool      `json:"has_card"`
	CardType     string    `json:"card_type,omitempty"`
	FreeDraws    int       `json:"free_draws"`     // 每日免费抽次数
	DrawDiscount float64   `json:"draw_discount"`  // 折扣比例
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	DaysLeft     int       `json:"days_left"`      // 剩余天数
	TodayFreeUsed int      `json:"today_free_used"` // 今日已用免费抽
}

// BattlePassSeason 战令赛季
type BattlePassSeason struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	MaxLevel int       `json:"max_level"` // 满级（如 50）
	XPPerLevel int     `json:"xp_per_level"` // 每级所需经验
	StartAt  time.Time `json:"start_at"`
	EndAt    time.Time `json:"end_at"`
	Status   string    `json:"status"` // upcoming / active / ended
}

// BattlePass 用户战令进度
type BattlePass struct {
	UserID       string    `json:"user_id"`
	SeasonID     int       `json:"season_id"`
	PassType     string    `json:"pass_type"`     // free / paid
	Level        int       `json:"level"`          // 当前等级
	XP           int       `json:"xp"`             // 当前经验值
	TotalXP      int       `json:"total_xp"`       // 累计获得经验
	ClaimedLevels []int    `json:"claimed_levels"` // 已领取奖励的等级
	BoughtAt     time.Time `json:"bought_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BattlePassTask 战令任务
type BattlePassTask struct {
	ID          int    `json:"id"`
	SeasonID    int    `json:"season_id"`
	Type        string `json:"type"`        // daily / weekly / season
	Name        string `json:"name"`
	Description string `json:"description"`
	XPReward    int    `json:"xp_reward"`   // 任务奖励经验
	Condition   string `json:"condition"`   // 完成条件描述（如"单抽3次"）
	TargetCount int    `json:"target_count"` // 目标次数
}

// BattlePassTaskProgress 用户任务进度
type BattlePassTaskProgress struct {
	UserID      string `json:"user_id"`
	TaskID      int    `json:"task_id"`
	Progress    int    `json:"progress"`     // 当前进度
	Completed   bool   `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// BattlePassReward 战令奖励配置
type BattlePassReward struct {
	Level      int    `json:"level"`
	PassType   string `json:"pass_type"` // free / paid
	RewardType string `json:"reward_type"` // points / draw_ticket / prize / title
	RewardName string `json:"reward_name"`
	RewardQty  int    `json:"reward_qty"`
	RewardID   string `json:"reward_id,omitempty"` // 奖品ID（如果reward_type=prize）
}

// BattlePassInfo 战令信息（前端查询用）
type BattlePassInfo struct {
	Season      *BattlePassSeason   `json:"season"`
	UserPass    *BattlePass         `json:"user_pass,omitempty"`
	Tasks       []BattlePassTask    `json:"tasks"`
	TaskProgress []BattlePassTaskProgress `json:"task_progress,omitempty"`
	Rewards     []BattlePassReward  `json:"rewards"`
	LevelProgress int               `json:"level_progress"` // 当前等级经验进度
}
// ============================================================

// BlendRecipe 合成配方
var BlendRecipes = map[string]struct {
	NeedCount    int    // 需要多少个
	ResultLevel  string // 合成结果级别
	Description  string // 描述
}{
	"common":  {NeedCount: 3, ResultLevel: "rare",   Description: "3个普通 → 1个稀有"},
	"rare":    {NeedCount: 5, ResultLevel: "secret", Description: "5个稀有 → 1个隐藏"},
	"secret":  {NeedCount: 3, ResultLevel: "limited", Description: "3个隐藏 → 1个限定"},
}

// BlendRequest 合成请求
type BlendRequest struct {
	SourcePrizeID string `json:"source_prize_id"` // 要合成的源款式ID
	CampaignID    string `json:"campaign_id"`      // 系列ID
}

// BlendResult 合成结果
type BlendResult struct {
	SourcePrizeID   string `json:"source_prize_id"`
	SourcePrizeName string `json:"source_prize_name"`
	SourceLevel     string `json:"source_level"`
	ResultPrizeID   string `json:"result_prize_id"`
	ResultPrizeName string `json:"result_prize_name"`
	ResultLevel     string `json:"result_level"`
	RemainingSrc    int    `json:"remaining_src"` // 合成后剩余的源款式数量
}

// ============================================================
// 🆕 限时商店 + 付费道具 + 首充礼包 系统
// ============================================================

// ItemType 道具类型
type ItemType string

const (
	ItemHintCard       ItemType = "hint_card"        // 提示卡：排除当前池1个款式
	ItemSeeThrough     ItemType = "see_through"      // 透卡：预览下一抽（仅普通款）
	ItemPityInherit    ItemType = "pity_inherit"     // 保底继承券：跨池保底转移
	ItemSpecifyVoucher ItemType = "specify_voucher"  // 指定款券：必得指定普通款
	ItemTenDrawTicket  ItemType = "ten_draw_ticket"  // 十连券
	ItemFreeDraw       ItemType = "free_draw"        // 免费抽券
)

// ShopItem 商店商品
type ShopItem struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	PricePoints int64      `json:"price_points"`    // 积分价格（0表示仅现金购买）
	PriceCash   int64      `json:"price_cash"`      // 现金价格（分，0表示仅积分购买）
	ItemType    ItemType   `json:"item_type"`        // 道具类型
	ItemQty     int        `json:"item_qty"`         // 数量
	Stock       int        `json:"stock"`            // 库存(-1=无限)
	DailyLimit  int        `json:"daily_limit"`      // 每日限购(0=不限)
	Category    string     `json:"category"`         // daily/weekly/festival/item
	IsActive    bool       `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	SortOrder   int        `json:"sort_order"`
}

// UserItem 用户道具库存
type UserItem struct {
	UserID   string   `json:"user_id"`
	ItemType ItemType `json:"item_type"`
	Quantity int      `json:"quantity"`
}

// UseItemRequest 使用道具请求
type UseItemRequest struct {
	ItemType   ItemType `json:"item_type"`
	CampaignID string   `json:"campaign_id,omitempty"` // 部分道具需要
	PrizeID    string   `json:"prize_id,omitempty"`    // 指定款券需要
}

// BuyShopItemRequest 购买商品请求
type BuyShopItemRequest struct {
	ShopItemID string `json:"shop_item_id"`
	Quantity   int    `json:"quantity"` // 购买数量（默认1）
}

// BuyShopItemResult 购买结果
type BuyShopItemResult struct {
	ItemType   ItemType `json:"item_type"`
	ItemName   string   `json:"item_name"`
	Quantity   int      `json:"quantity"`
	PointsCost int64    `json:"points_cost"`
	NewPoints  int64    `json:"new_points"`
	NewQty     int      `json:"new_qty"` // 购买后该道具数量
}

// 🆕 首充礼包系统

// FirstRechargePack 首充礼包定义
type FirstRechargePack struct {
	ID          string      `json:"id"`          // "tier_1"/"tier_2"/"tier_3"
	Name        string      `json:"name"`        // "首充6元"/"首充30元"/"首充98元"
	PricePoints int64       `json:"price_points"` // 积分价格
	CashPrice   int64       `json:"cash_price"`   // 现金价格（分）600/3000/9800
	Items       []PackItem  `json:"items"`
	Description string      `json:"description"`
	SortOrder   int         `json:"sort_order"`
}

// PackItem 礼包内容项
type PackItem struct {
	Type    string `json:"type"`     // "points"/"draw_ticket"/"prize"/"month_card"/"hint_card"/"see_through"
	Name    string `json:"name"`
	Qty     int    `json:"qty"`
	PrizeID string `json:"prize_id,omitempty"` // 如果是奖品
}

// 首充礼包配置表
var FirstRechargePacks = map[string]FirstRechargePack{
	"tier_1": {
		ID: "tier_1", Name: "首充6元", PricePoints: 600, CashPrice: 600,
		Items: []PackItem{
			{Type: "points", Name: "积分", Qty: 60},
			{Type: "prize", Name: "稀有款盲盒", Qty: 1},
			{Type: "hint_card", Name: "提示卡", Qty: 3},
		},
		Description: "60积分+稀有盲盒×1+提示卡×3", SortOrder: 1,
	},
	"tier_2": {
		ID: "tier_2", Name: "首充30元", PricePoints: 3000, CashPrice: 3000,
		Items: []PackItem{
			{Type: "points", Name: "积分", Qty: 300},
			{Type: "ten_draw_ticket", Name: "十连券", Qty: 1},
			{Type: "see_through", Name: "透卡", Qty: 5},
		},
		Description: "300积分+十连券×1+透卡×5+限定头像框", SortOrder: 2,
	},
	"tier_3": {
		ID: "tier_3", Name: "首充98元", PricePoints: 9800, CashPrice: 9800,
		Items: []PackItem{
			{Type: "points", Name: "积分", Qty: 980},
			{Type: "month_card", Name: "月卡(30天)", Qty: 1},
			{Type: "ten_draw_ticket", Name: "十连券", Qty: 2},
		},
		Description: "980积分+月卡×1+十连券×2+稀有款自选券", SortOrder: 3,
	},
}

// UserFirstRecharge 用户首充记录
type UserFirstRecharge struct {
	UserID   string   `json:"user_id"`
	Claimed  []string `json:"claimed"`  // 已领取的礼包ID列表
}

// ClaimFirstRechargeRequest 领取首充礼包请求
type ClaimFirstRechargeRequest struct {
	PackID string `json:"pack_id"`
}

// ClaimFirstRechargeResult 领取结果
type ClaimFirstRechargeResult struct {
	PackID    string     `json:"pack_id"`
	PackName  string     `json:"pack_name"`
	Items     []PackItem `json:"items"`
	NewPoints int64      `json:"new_points"`
}

// ============================================================
// 🆕 v1.5 社交裂变系统
// ============================================================

// AssistType 助力类型
type AssistType string

const (
	AssistFreeDraw    AssistType = "free_draw"    // 助力免费抽：邀请3人→1次免费抽
	AssistPityReduce  AssistType = "pity_reduce"  // 助力保底：邀请5人→保底-10
	AssistCraftBoost  AssistType = "craft_boost"  // 助力合成：邀请2人→合成概率+20%
)

// InviteRecord 邀请记录
type InviteRecord struct {
	ID         string    `json:"id"`
	InviterID  string    `json:"inviter_id"`   // 邀请人
	InviteeID  string    `json:"invitee_id"`   // 被邀请人
	CreatedAt  time.Time `json:"created_at"`
}

// AssistProgress 用户助力进度
type AssistProgress struct {
	InviterID   string     `json:"inviter_id"`
	AssistType  AssistType `json:"assist_type"`
	TargetCount int        `json:"target_count"` // 目标助力次数（3/5/2）
	Current     int        `json:"current"`       // 当前助力次数
	Claimed     bool       `json:"claimed"`       // 是否已领取奖励
	ExpiresAt   time.Time  `json:"expires_at"`    // 过期时间（24h）
	CreatedAt   time.Time  `json:"created_at"`
}

// AssistAction 好友助力动作记录（防刷）
type AssistAction struct {
	ID         string    `json:"id"`
	InviterID  string    `json:"inviter_id"`
	HelperID   string    `json:"helper_id"`   // 帮助助力的好友
	AssistType AssistType `json:"assist_type"`
	CreatedAt  time.Time `json:"created_at"`
}

// Team 队伍
type Team struct {
	ID          string    `json:"id"`
	CaptainID   string    `json:"captain_id"`
	Name        string    `json:"name"`
	MaxMembers  int       `json:"max_members"`  // 2~5人
	GoalDraws   int       `json:"goal_draws"`   // 开盒目标次数
	CurrentDraws int      `json:"current_draws"`// 当前累计开盒次数
	StartsAt    time.Time `json:"starts_at"`
	ExpiresAt   time.Time `json:"expires_at"`   // 48小时有效
	Status      string    `json:"status"`       // recruiting / active / completed / expired
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// TeamMember 队伍成员
type TeamMember struct {
	TeamID    string    `json:"team_id"`
	UserID    string    `json:"user_id"`
	Nickname  string    `json:"nickname,omitempty"`
	Draws     int       `json:"draws"`      // 个人开盒次数
	JoinedAt  time.Time `json:"joined_at"`
}

// TeamReward 组队奖励
type TeamReward struct {
	TeamID      string `json:"team_id"`
	CaptainID   string `json:"captain_id"`
	RewardType  string `json:"reward_type"` // points / draw_ticket / prize
	RewardQty   int    `json:"reward_qty"`
	Description string `json:"description"`
}

// GiftRecord 礼物赠送记录
type GiftRecord struct {
	ID           string    `json:"id"`
	GiverID      string    `json:"giver_id"`     // 赠送者
	ReceiverID   string    `json:"receiver_id"`  // 接收者
	PrizeID      string    `json:"prize_id"`
	PrizeName    string    `json:"prize_name"`
	PrizeLevel   string    `json:"prize_level"`
	FeePoints    int64     `json:"fee_points"`   // 包装费（稀有赠送时消耗）
	Status       string    `json:"status"`       // sent / received / expired
	CreatedAt    time.Time `json:"created_at"`
	ReceivedAt   *time.Time `json:"received_at,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`   // 24h未领取过期
}

// SendGiftRequest 赠送礼物请求
type SendGiftRequest struct {
	ReceiverID string `json:"receiver_id"`
	PrizeID    string `json:"prize_id"`
	CampaignID string `json:"campaign_id"`
}

// ReceiveGiftResult 接收礼物结果
type ReceiveGiftResult struct {
	GiftID     string `json:"gift_id"`
	PrizeName  string `json:"prize_name"`
	PrizeLevel string `json:"prize_level"`
	NewItemID  string `json:"new_item_id,omitempty"`
}

// ShareCard 分享卡片
type ShareCard struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	CardType    string    `json:"card_type"`    // draw_win / collection / craft / team / invite
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ImageURL    string    `json:"image_url,omitempty"`
	PrizeName   string    `json:"prize_name,omitempty"`
	PrizeLevel  string    `json:"prize_level,omitempty"`
	InviteLink  string    `json:"invite_link,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateTeamRequest 创建队伍请求
type CreateTeamRequest struct {
	Name      string `json:"name"`
	MaxMembers int   `json:"max_members"` // 2~5
	GoalDraws int    `json:"goal_draws"` // 目标开盒次数
}

// JoinTeamRequest 加入队伍请求
type JoinTeamRequest struct {
	TeamID string `json:"team_id"`
}

// TeamInfo 队伍信息（前端展示用）
type TeamInfo struct {
	Team          *Team         `json:"team"`
	Members       []TeamMember  `json:"members"`
	CaptainName   string        `json:"captain_name"`
	RemainingHours int          `json:"remaining_hours"`
	Reward        *TeamReward   `json:"reward,omitempty"`
}

// AssistClaimResult 领取助力奖励结果
type AssistClaimResult struct {
	AssistType  AssistType `json:"assist_type"`
	RewardType  string     `json:"reward_type"`  // free_draw / pity_reduce / craft_boost
	Description string     `json:"description"`
}

// InviteStats 邀请统计
type InviteStats struct {
	TotalInvites     int `json:"total_invites"`
	TotalAssists     int `json:"total_assists"`
	CompletedAssists int `json:"completed_assists"`
	FreeDrawsEarned  int `json:"free_draws_earned"`
}
