package store

import (
	"errors"

	"campaign-lottery-platform/backend/internal/model"
)

// Error sentinels (preserved from original)
var (
	ErrUnauthorized         = errors.New("unauthorized")
	ErrCampaignNotFound     = errors.New("campaign not found")
	ErrCampaignInactive     = errors.New("campaign inactive")
	ErrNoDrawChances        = errors.New("no draw chances")
	ErrBadAdminAuth         = errors.New("bad admin credentials")
	ErrAdminUnauthorized    = errors.New("admin unauthorized")
	ErrInsufficientPoints   = errors.New("insufficient points")
	ErrShareLimitReached    = errors.New("daily share limit reached")
)

// Store defines the data access interface for the blind box lottery platform.
type Store interface {
	// ---- 原有接口 ----
	CreateGuestSession(nickname string) (model.User, model.Session, error)
	Campaigns() []model.Campaign
	PrizeList(campaignID string) []model.Prize
	UserFromToken(token string) (model.User, error)
	Draw(token string, campaignID string) (model.DrawResult, error)
	UserDrawRecords(token string) ([]model.DrawRecord, error)
	AdminLogin(username string, password string) (string, error)
	AdminOverview(token string) (model.AdminOverview, error)
	AdminDrawRecords(token string) ([]model.DrawRecord, error)
	AdminCampaigns(token string) ([]model.Campaign, error)
	CreateCampaign(token string, input model.CampaignMutation) (model.Campaign, error)
	UpdateCampaign(token string, campaignID string, input model.CampaignMutation) (model.Campaign, error)
	DeleteCampaign(token string, campaignID string) error
	AdminPrizes(token string, campaignID string) ([]model.Prize, error)
	CreatePrize(token string, campaignID string, input model.PrizeMutation) (model.Prize, error)
	UpdatePrize(token string, prizeID string, input model.PrizeMutation) (model.Prize, error)
	DeletePrize(token string, prizeID string) error
	FulfillmentTasks(token string) ([]model.FulfillmentTask, error)
	UpdateFulfillmentTask(token string, taskID int64, input model.FulfillmentTaskMutation) (model.FulfillmentTask, error)
	Seed() error

	// ---- 盲盒扩展接口 ----

	// GetCampaign 获取单个活动详情
	GetCampaign(campaignID string) (model.Campaign, error)

	// CreateDrawRecord 中奖并创建抽奖记录（自动扣库存），isTenPull 表示是否十连抽
	CreateDrawRecord(userID, campaignID, prizeID string, isTenPull bool) (model.DrawRecord, error)

	// CreateMissRecord 未中奖记录
	CreateMissRecord(userID, campaignID string, isTenPull bool) (model.DrawRecord, error)

	// CheckDrawQuota 检查剩余抽奖次数
	CheckDrawQuota(userID, campaignID string, dailyLimit int) (int, error)

	// DeductDrawQuota 扣减抽奖次数，返回剩余次数
	DeductDrawQuota(userID, campaignID string, count int) (int, error)

	// GetUserInventory 获取用户库存
	GetUserInventory(userID string) ([]model.UserInventory, error)

	// GetSeriesProgress 获取用户系列收集进度
	GetSeriesProgress(userID, campaignID, campaignName string) (*model.SeriesProgress, error)

	// UserHasPrize 检查用户是否拥有指定奖品
	UserHasPrize(userID, prizeID string) (bool, error)

	// ---- 交换市场 ----
	ExchangeOffers() []model.ExchangeOffer
	CreateExchangeOffer(userID string, input model.ExchangeOfferMutation) (model.ExchangeOffer, error)
	CancelExchangeOffer(userID, offerID string) error
	AcceptExchangeOffer(userID, offerID string) (model.ExchangeOffer, error)

	// ---- 积分/会员 ----
	GetUserMember(userID string) (*model.UserMember, error)
	GetPointsLog(userID string) ([]model.UserPointsLog, error)
	RedeemPrize(userID string, input model.RedeemRequest) (*model.RedeemResult, error)
	// UpdateUserMember 更新用户的会员/积分信息
	UpdateUserMember(member *model.UserMember) error

	// ---- 数据统计 ----
	GetDrawStatistics(token, campaignID string) (*model.DrawStatistics, error)

	// ---- 集卡系统扩展 ----
	// DailyCheckIn 每日签到，points 参数是本次应增加的积分
	DailyCheckIn(userID string, points int64) (*model.CheckInResult, error)
	// GetCheckInStreak 获取连续签到天数
	GetCheckInStreak(userID string) (int, error)
	// CheckCollectionCompletion 检查用户是否集齐系列所有款式，返回奖励（nil=未集齐）
	CheckCollectionCompletion(userID, campaignID string) (*model.CollectionReward, error)
	// GrantCollectionReward 发放集齐奖励（隐藏款解锁）
	GrantCollectionReward(userID string, reward *model.CollectionReward) error
	// GetLeaderboard 获取收集排行榜（按所有系列收集进度排名）
	GetLeaderboard(limit int) ([]model.LeaderboardEntry, error)
	// GetCampaignHint 获取系列摇盒提示文案
	GetCampaignHint(campaignID string) *model.HintMessage
	// ShareReward 分享奖励，返回今日还可分享次数
	ShareReward(userID string, points int64) (*model.ShareRewardResult, error)
	// GetShareDailyCount 获取今日已分享次数
	GetShareDailyCount(userID string) (int, error)
	// GetPrizeCount 获取用户某系列某款式的数量
	GetPrizeCount(userID, prizeID string) (int, error)
}
