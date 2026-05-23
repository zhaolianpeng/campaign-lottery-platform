package model

import "time"

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
	PityConfig      PityConfig `json:"pity_config"`       // 概率保底配置
	SeriesImageURL  string     `json:"series_image_url"`  // 系列主图
	SeriesItemCount int        `json:"series_item_count"` // 系列总款式数
	DrawPrice       int        `json:"draw_price"`        // 单抽价格（分）
	TenDrawPrice    int        `json:"ten_draw_price"`    // 十连抽价格（分）
}

// BlindBoxPrize 盲盒礼品（扩展 Prize）
type BlindBoxPrize struct {
	Prize
	ImageURL    string `json:"image_url"`     // 礼品图片
	SortOrder   int    `json:"sort_order"`    // 排序
	DisplayProb string `json:"display_prob"`  // 对外公示概率（如 "7.0%"）
}

// DrawConfig 抽奖配置请求体
type DrawConfig struct {
	CampaignID string `json:"campaign_id"`
	DrawCount  int    `json:"draw_count"` // 1=单抽, >=2=十连抽
}

// PityStatus 保底状态返回
type PityStatus struct {
	ConsecutiveMisses int     `json:"consecutive_misses"`   // 连续未中次数
	PityMultiplier    float64 `json:"pity_multiplier"`      // 当前概率倍数
	SoftPityN         int     `json:"soft_pity_n"`          // 软保底触发次数
	HardPityN         int     `json:"hard_pity_n"`          // 硬保底次数
	MissesToHardPity  int     `json:"misses_to_hard_pity"`  // 距离硬保底还差几次
	HasUPPoolGuarantee bool   `json:"has_up_pool_guarantee,omitempty"` // 🆕 是否持有大保底
}

// BlindBoxDrawResult 盲盒抽奖结果
type BlindBoxDrawResult struct {
	Draws            []SingleDrawResult `json:"draws"`                        // 本次抽奖结果列表
	RemainingChances int                `json:"remaining_chances"`            // 剩余抽奖次数
	PityStatus       *PityStatus        `json:"pity_status,omitempty"`        // 当前保底状态
	CollectionReward *CollectionReward  `json:"collection_reward,omitempty"`  // 集齐奖励（如触发）
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
	IsUPPoolWin   bool   `json:"is_up_pool_win,omitempty"` // 🆕 是否是UP池中奖
}

// SeriesProgress 用户系列收集进度
type SeriesProgress struct {
	CampaignID      string           `json:"campaign_id"`
	CampaignName    string           `json:"campaign_name"`
	TotalItems      int              `json:"total_items"`       // 系列总款式数
	CollectedItems  int              `json:"collected_items"`   // 已收集款式数
	ProgressPercent float64          `json:"progress_percent"`  // 收集进度百分比
	Duplicates      int              `json:"duplicates"`        // 重复款数量
	CollectedPrizes []CollectedPrize `json:"collected_prizes"`  // 已收集详情
	MissingPrizes   []PrizeSummary   `json:"missing_prizes"`    // 缺失款式
}

// CollectedPrize 已收集的奖品
type CollectedPrize struct {
	Prize
	Count int `json:"count"` // 拥有数量（>=1）
}

// ExchangeOffer 交换市场挂单
type ExchangeOffer struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	UserNickname  string    `json:"user_nickname,omitempty"`
	HavePrizeID   string    `json:"have_prize_id"`
	HavePrizeName string    `json:"have_prize_name"`
	WantPrizeID   string    `json:"want_prize_id"`
	WantPrizeName string    `json:"want_prize_name"`
	Status        string    `json:"status"` // pending / matched / completed / cancelled
	CreatedAt     time.Time `json:"created_at"`
}

// ExchangeOfferMutation 交换挂单请求体
type ExchangeOfferMutation struct {
	HavePrizeID string `json:"have_prize_id"`
	WantPrizeID string `json:"want_prize_id"`
}

// ExchangeMatchRequest 交换匹配请求体
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

// ============================================================
// 🆕 v1.6 碎片拼图 + 预约抢购 系统
// ============================================================

// PuzzleTemplate 拼图模板
type PuzzleTemplate struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	CampaignID  string    `json:"campaign_id"`
	TotalPieces int       `json:"total_pieces"` // 6/12/24块
	PieceNames  []string  `json:"piece_names"`  // 每块碎片名称
	RewardType  string    `json:"reward_type"`  // prize / points / draw_ticket
	RewardID    string    `json:"reward_id,omitempty"`
	RewardQty   int       `json:"reward_qty"`
	RewardName  string    `json:"reward_name"`
	PeriodType  string    `json:"period_type"` // weekly / monthly / festival
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// PuzzleProgress 用户拼图进度
type PuzzleProgress struct {
	UserID      string     `json:"user_id"`
	TemplateID  string     `json:"template_id"`
	Collected   []int      `json:"collected"`         // 已收集的碎片位置索引
	TotalPieces int        `json:"total_pieces"`
	IsCompleted bool       `json:"is_completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	TeamID      string     `json:"team_id,omitempty"` // 社群拼图队伍ID
}

// PuzzleTeam 拼图小队
type PuzzleTeam struct {
	ID          string    `json:"id"`
	TemplateID  string    `json:"template_id"`
	CaptainID   string    `json:"captain_id"`
	Members     []string  `json:"members"`       // 成员userID列表
	Shared      []int     `json:"shared"`        // 小队已共享的碎片位置
	TotalPieces int       `json:"total_pieces"`
	IsCompleted bool      `json:"is_completed"`
	CreatedAt   time.Time `json:"created_at"`
}

// PuzzlePieceDrop 碎片掉落结果（抽奖时可能附带）
type PuzzlePieceDrop struct {
	TemplateID string `json:"template_id"`
	PieceIndex int    `json:"piece_index"`
	PieceName  string `json:"piece_name"`
	IsNew      bool   `json:"is_new"` // 是否是新收集到的
}

// PuzzleInfo 拼图信息（给前端展示）
type PuzzleInfo struct {
	Template        *PuzzleTemplate `json:"template"`
	Progress        *PuzzleProgress `json:"progress"`
	CollectedNames  []string        `json:"collected_names"`
	MissingNames    []string        `json:"missing_names"`
	ProgressPercent float64         `json:"progress_percent"`
}

// CreatePuzzleTeamRequest 创建拼图小队请求
type CreatePuzzleTeamRequest struct {
	TemplateID string `json:"template_id"`
}

// JoinPuzzleTeamRequest 加入拼图小队请求
type JoinPuzzleTeamRequest struct {
	TeamID string `json:"team_id"`
}

// ComposePuzzleRequest 合成拼图请求
type ComposePuzzleRequest struct {
	TemplateID string `json:"template_id"`
}

// ComposePuzzleResult 合成拼图结果
type ComposePuzzleResult struct {
	TemplateID   string `json:"template_id"`
	TemplateName string `json:"template_name"`
	RewardType   string `json:"reward_type"`
	RewardName   string `json:"reward_name"`
	RewardQty    int    `json:"reward_qty"`
}

// ============================================================
// 🆕 v2.0 活动系统
// ============================================================

// ActivityType 活动类型
type ActivityType string

const (
	ActivityUPPool       ActivityType = "up_pool"        // UP池概率提升
	ActivityDiscount     ActivityType = "discount"       // 限时折扣
	ActivityFestival     ActivityType = "festival"       // 节日活动
	ActivityCheckinBoost ActivityType = "checkin_boost"  // 签到加倍
	ActivityCraftBoost   ActivityType = "craft_boost"    // 合成加成
	ActivityFlashSale    ActivityType = "flash_sale"     // 限时抢购
)

// Activity 活动
type Activity struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        ActivityType `json:"type"`
	BannerURL   string       `json:"banner_url,omitempty"`
	Rules       ActivityRules `json:"rules"`         // 活动规则（JSON配置）
	SortOrder   int          `json:"sort_order"`
	Status      string       `json:"status"`         // draft / active / paused / ended
	StartAt     time.Time    `json:"start_at"`
	EndAt       time.Time    `json:"end_at"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// ActivityRules 活动规则配置
type ActivityRules struct {
	// UP池规则
	UPPrizeID    string  `json:"up_prize_id,omitempty"`
	UPMultiplier float64 `json:"up_multiplier,omitempty"`
	UPLevel      string  `json:"up_level,omitempty"`
	UPCampaignID string  `json:"up_campaign_id,omitempty"`

	// 折扣规则
	DiscountRate   float64 `json:"discount_rate,omitempty"`    // 0.8 = 8折
	DiscountTarget string  `json:"discount_target,omitempty"`  // single_draw / ten_draw / shop

	// 签到加倍
	CheckinMultiplier int `json:"checkin_multiplier,omitempty"` // 签到积分倍数

	// 合成加成
	CraftBoostRate float64 `json:"craft_boost_rate,omitempty"` // 合成成功率加成

	// 节日礼包
	GiftPackID string `json:"gift_pack_id,omitempty"`

	// 抢购（复用FlashSale系统）
	FlashID string `json:"flash_id,omitempty"`
}

// ActivityParticipation 用户活动参与记录
type ActivityParticipation struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	ActivityID    string    `json:"activity_id"`
	Data          string    `json:"data,omitempty"` // 参与数据（如抽奖次数等，JSON）
	RewardClaimed bool      `json:"reward_claimed"`
	JoinedAt      time.Time `json:"joined_at"`
}

// ActivityReward 活动奖励
type ActivityReward struct {
	ID         string `json:"id"`
	ActivityID string `json:"activity_id"`
	Condition  string `json:"condition"`   // 条件描述
	RewardType string `json:"reward_type"` // points / draw_ticket / prize / item
	RewardQty  int    `json:"reward_qty"`
	RewardName string `json:"reward_name"`
	RewardID   string `json:"reward_id,omitempty"`
}

// ActivityCreateRequest 创建活动请求
type ActivityCreateRequest struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        ActivityType `json:"type"`
	BannerURL   string       `json:"banner_url,omitempty"`
	Rules       ActivityRules `json:"rules"`
	SortOrder   int          `json:"sort_order"`
	StartAt     time.Time    `json:"start_at"`
	EndAt       time.Time    `json:"end_at"`
}

// ActivityUpdateRequest 更新活动请求
type ActivityUpdateRequest struct {
	Name        string        `json:"name,omitempty"`
	Description string        `json:"description,omitempty"`
	BannerURL   string        `json:"banner_url,omitempty"`
	Rules       *ActivityRules `json:"rules,omitempty"`
	SortOrder   *int          `json:"sort_order,omitempty"`
	Status      string        `json:"status,omitempty"`
	StartAt     *time.Time    `json:"start_at,omitempty"`
	EndAt       *time.Time    `json:"end_at,omitempty"`
}

// ClaimActivityRewardRequest 领取活动奖励请求
type ClaimActivityRewardRequest struct {
	ActivityID string `json:"activity_id"`
	RewardID   string `json:"reward_id"`
}

// ActivityListInfo 活动列表项（前端展示）
type ActivityListInfo struct {
	Activity *Activity       `json:"activity"`
	Joined   bool            `json:"joined"`
	CanClaim bool            `json:"can_claim"`
	Rewards  []ActivityReward `json:"rewards,omitempty"`
}
