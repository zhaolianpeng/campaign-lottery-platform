package model

import "time"

// ============================================================
// 核心模型
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
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Slug            string     `json:"slug"`
	Status          string     `json:"status"`
	StartsAt        time.Time  `json:"starts_at"`
	EndsAt          time.Time  `json:"ends_at"`
	DailyDrawLimit  int        `json:"daily_draw_limit"`
	MissWeight      int        `json:"miss_weight"`
	BannerImageURL  string     `json:"banner_image_url"`
	CampaignSummary string     `json:"campaign_summary"`
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
