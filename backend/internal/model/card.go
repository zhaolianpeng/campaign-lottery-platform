package model

import "time"

// ============================================================
// 卡系统模型（合并原 CardConfig/UserCard + MonthCard 系统）
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

// UserCard 用户已购卡（统一模型，合并原 UserCard + MonthCard）
type UserCard struct {
	ID            string    `json:"id,omitempty"`
	UserID        string    `json:"user_id"`
	CardType      CardType  `json:"card_type"`
	Price         int64     `json:"price,omitempty"`     // 购买价格（分）
	StartedAt     time.Time `json:"started_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	DailyFreeUsed int       `json:"daily_free_used"`    // 今日已用免费次数
	FreeDate      string    `json:"free_date"`          // 记录免费次数日期 "2006-01-02"
	CreatedAt     time.Time `json:"created_at,omitempty"`
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

// ============================================================
// 月卡系统（原 MonthCard 类型，保留为兼容别名）
// ============================================================

// MonthCardType 月卡类型（与 CardType 值相同，保持兼容）
type MonthCardType string

const (
	MonthCardWeekly  MonthCardType = "weekly"  // 周卡 9.9元
	MonthCardMonthly MonthCardType = "monthly" // 月卡 28元
	MonthCardSeason  MonthCardType = "season"  // 季卡 68元
)

// MonthCard 用户月卡信息（兼容旧接口，推荐使用 UserCard）
type MonthCard struct {
	ID           string        `json:"id"`
	UserID       string        `json:"user_id"`
	CardType     MonthCardType `json:"card_type"`
	Price        int64         `json:"price"`          // 购买价格（分）
	FreeDraws    int           `json:"free_draws"`     // 每日免费抽次数
	DrawDiscount float64       `json:"draw_discount"`  // 折扣（0.8 = 8折）
	StartedAt    time.Time     `json:"started_at"`
	ExpiresAt    time.Time     `json:"expires_at"`
	CreatedAt    time.Time     `json:"created_at"`
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
	HasCard       bool      `json:"has_card"`
	CardType      string    `json:"card_type,omitempty"`
	FreeDraws     int       `json:"free_draws"`      // 每日免费抽次数
	DrawDiscount  float64   `json:"draw_discount"`   // 折扣比例
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
	DaysLeft      int       `json:"days_left"`       // 剩余天数
	TodayFreeUsed int       `json:"today_free_used"` // 今日已用免费抽
}

// ============================================================
// 积分系统
// ============================================================

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
// 会员等级系统
// ============================================================

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
	UserID     string      `json:"user_id"`
	Level      MemberLevel `json:"level"`
	Points     int64       `json:"points"`      // 积分
	TotalDraws int64       `json:"total_draws"` // 累计抽奖次数
	TotalSpent int64       `json:"total_spent"` // 累计消费（分）
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}
