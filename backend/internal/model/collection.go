package model

import "time"

// ============================================================
// 集卡系统扩展模型
// ============================================================

// CheckInResult 每日签到结果
type CheckInResult struct {
	PointsAwarded int64 `json:"points_awarded"` // 本次获得积分
	StreakDays    int   `json:"streak_days"`    // 连续签到天数
	IsBonus       bool  `json:"is_bonus"`       // 是否触发连续奖励
	NewBalance    int64 `json:"new_balance"`    // 变动后余额
}

// CollectionReward 集齐系列奖励
type CollectionReward struct {
	CampaignID    string `json:"campaign_id"`
	CampaignName  string `json:"campaign_name"`
	RewardType    string `json:"reward_type"`     // hidden_prize / title / points
	RewardName    string `json:"reward_name"`
	RewardPrizeID string `json:"reward_prize_id,omitempty"`
	Description   string `json:"description"`
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
	TotalDraws     int64           `json:"total_draws"`
	TotalUsers     int64           `json:"total_users"`
	TotalWins      int64           `json:"total_wins"`
	WinRate        float64         `json:"win_rate"`
	PrizeBreakdown []PrizeStatItem `json:"prize_breakdown"`
	DailyDraws     []DailyDrawStat `json:"daily_draws,omitempty"`
}

// PrizeStatItem 奖品统计项
type PrizeStatItem struct {
	PrizeID   string  `json:"prize_id"`
	PrizeName string  `json:"prize_name"`
	Level     string  `json:"level"`
	Count     int64   `json:"count"`
	Percent   float64 `json:"percent"`
}

// DailyDrawStat 每日抽奖统计
type DailyDrawStat struct {
	Date  string `json:"date"` // "2006-01-02"
	Count int64  `json:"count"`
	Wins  int64  `json:"wins"`
}

// ============================================================
// 合成系统
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
// 🆕 v1.5 社交裂变系统
// ============================================================

// AssistType 助力类型
type AssistType string

const (
	AssistFreeDraw   AssistType = "free_draw"   // 助力免费抽：邀请3人→1次免费抽
	AssistPityReduce AssistType = "pity_reduce" // 助力保底：邀请5人→保底-10
	AssistCraftBoost AssistType = "craft_boost" // 助力合成：邀请2人→合成概率+20%
)

// InviteRecord 邀请记录
type InviteRecord struct {
	ID        string    `json:"id"`
	InviterID string    `json:"inviter_id"` // 邀请人
	InviteeID string    `json:"invitee_id"` // 被邀请人
	CreatedAt time.Time `json:"created_at"`
}

// AssistProgress 用户助力进度
type AssistProgress struct {
	InviterID   string     `json:"inviter_id"`
	AssistType  AssistType `json:"assist_type"`
	TargetCount int        `json:"target_count"` // 目标助力次数（3/5/2）
	Current     int        `json:"current"`      // 当前助力次数
	Claimed     bool       `json:"claimed"`      // 是否已领取奖励
	ExpiresAt   time.Time  `json:"expires_at"`   // 过期时间（24h）
	CreatedAt   time.Time  `json:"created_at"`
}

// AssistAction 好友助力动作记录（防刷）
type AssistAction struct {
	ID         string     `json:"id"`
	InviterID  string     `json:"inviter_id"`
	HelperID   string     `json:"helper_id"`   // 帮助助力的好友
	AssistType AssistType `json:"assist_type"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Team 队伍
type Team struct {
	ID           string     `json:"id"`
	CaptainID    string     `json:"captain_id"`
	Name         string     `json:"name"`
	MaxMembers   int        `json:"max_members"`    // 2~5人
	GoalDraws    int        `json:"goal_draws"`     // 开盒目标次数
	CurrentDraws int        `json:"current_draws"`  // 当前累计开盒次数
	StartsAt     time.Time  `json:"starts_at"`
	ExpiresAt    time.Time  `json:"expires_at"`     // 48小时有效
	Status       string     `json:"status"`         // recruiting / active / completed / expired
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// TeamMember 队伍成员
type TeamMember struct {
	TeamID   string    `json:"team_id"`
	UserID   string    `json:"user_id"`
	Nickname string    `json:"nickname,omitempty"`
	Draws    int       `json:"draws"`     // 个人开盒次数
	JoinedAt time.Time `json:"joined_at"`
}

// TeamReward 组队奖励
type TeamReward struct {
	TeamID      string `json:"team_id"`
	CaptainID   string `json:"captain_id"`
	RewardType  string `json:"reward_type"`  // points / draw_ticket / prize
	RewardQty   int    `json:"reward_qty"`
	Description string `json:"description"`
}

// GiftRecord 礼物赠送记录
type GiftRecord struct {
	ID          string     `json:"id"`
	GiverID     string     `json:"giver_id"`    // 赠送者
	ReceiverID  string     `json:"receiver_id"` // 接收者
	PrizeID     string     `json:"prize_id"`
	PrizeName   string     `json:"prize_name"`
	PrizeLevel  string     `json:"prize_level"`
	FeePoints   int64      `json:"fee_points"`   // 包装费（稀有赠送时消耗）
	Status      string     `json:"status"`       // sent / received / expired
	CreatedAt   time.Time  `json:"created_at"`
	ReceivedAt  *time.Time `json:"received_at,omitempty"`
	ExpiresAt   time.Time  `json:"expires_at"`   // 24h未领取过期
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
	Name       string `json:"name"`
	MaxMembers int    `json:"max_members"` // 2~5
	GoalDraws  int    `json:"goal_draws"` // 目标开盒次数
}

// JoinTeamRequest 加入队伍请求
type JoinTeamRequest struct {
	TeamID string `json:"team_id"`
}

// TeamInfo 队伍信息（前端展示用）
type TeamInfo struct {
	Team           *Team        `json:"team"`
	Members        []TeamMember `json:"members"`
	CaptainName    string       `json:"captain_name"`
	RemainingHours int          `json:"remaining_hours"`
	Reward         *TeamReward  `json:"reward,omitempty"`
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
