package model

import "time"

// ============================================================
// 战令系统模型
// ============================================================

// BattlePassSeason 战令赛季
type BattlePassSeason struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	MaxLevel    int       `json:"max_level"`     // 满级（如 50）
	XPPerLevel  int       `json:"xp_per_level"`  // 每级所需经验
	StartAt     time.Time `json:"start_at"`
	EndAt       time.Time `json:"end_at"`
	Status      string    `json:"status"`        // upcoming / active / ended
}

// BattlePass 用户战令进度
type BattlePass struct {
	UserID        string    `json:"user_id"`
	SeasonID      int       `json:"season_id"`
	PassType      string    `json:"pass_type"`       // free / paid
	Level         int       `json:"level"`           // 当前等级
	XP            int       `json:"xp"`              // 当前经验值
	TotalXP       int       `json:"total_xp"`        // 累计获得经验
	ClaimedLevels []int     `json:"claimed_levels"`  // 已领取奖励的等级
	BoughtAt      time.Time `json:"bought_at,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
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
	UserID      string     `json:"user_id"`
	TaskID      int        `json:"task_id"`
	Progress    int        `json:"progress"`      // 当前进度
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// BattlePassReward 战令奖励配置
type BattlePassReward struct {
	Level      int    `json:"level"`
	PassType   string `json:"pass_type"`  // free / paid
	RewardType string `json:"reward_type"` // points / draw_ticket / prize / title
	RewardName string `json:"reward_name"`
	RewardQty  int    `json:"reward_qty"`
	RewardID   string `json:"reward_id,omitempty"` // 奖品ID（如果reward_type=prize）
}

// BattlePassInfo 战令信息（前端查询用）
type BattlePassInfo struct {
	Season        *BattlePassSeason       `json:"season"`
	UserPass      *BattlePass             `json:"user_pass,omitempty"`
	Tasks         []BattlePassTask        `json:"tasks"`
	TaskProgress  []BattlePassTaskProgress `json:"task_progress,omitempty"`
	Rewards       []BattlePassReward      `json:"rewards"`
	LevelProgress int                     `json:"level_progress"` // 当前等级经验进度
}
