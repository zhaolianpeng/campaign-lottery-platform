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
	ErrNoActiveSeason       = errors.New("no active battle pass season")
	ErrAlreadyPurchased     = errors.New("already purchased")
	ErrNotEligible          = errors.New("not eligible")
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
// LogPoints 记录积分变动日志
LogPoints(userID string, points int64, balance int64, reason, remark string) error

	// ---- 月卡/付费卡系统 ----
	// GetUserCard 获取用户当前生效的卡（nil=无）
	GetUserCard(userID string) (*model.UserCard, error)
	// BuyCard 购买卡，扣减积分
	BuyCard(userID string, cardType model.CardType) (*model.BuyCardResult, error)
	// ConsumeFreeDraw 消耗一次免费抽，返回是否成功
	ConsumeFreeDraw(userID string) (bool, error)
	// GetFreeDrawRemaining 查询今日剩余免费抽次数
	GetFreeDrawRemaining(userID string) (int, error)

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

	// BlendPrizes 合成：消耗指定数量的重复款式，获得更高级款式
	// sourcePrizeID 是要消耗的款式ID，campaignID 是系列ID
	// 返回合成的结果
	BlendPrizes(userID string, sourcePrizeID string, campaignID string) (*model.BlendResult, error)

	// 🆕 ---- 月卡系统 ----
	// GetMonthCard 获取用户月卡信息（nil = 无月卡）
	GetMonthCard(userID string) (*model.MonthCard, error)
	// BuyMonthCard 购买月卡，pointsCost 是积分扣减数
	BuyMonthCard(userID string, cardType model.MonthCardType, pointsCost int64) (*model.MonthCard, error)
	// UseFreeDraw 消耗一次今日免费抽，返回剩余免费抽次数
	UseFreeDraw(userID string) (int, error)
	// GetTodayFreeDrawUsed 获取今日已用免费抽次数
	GetTodayFreeDrawUsed(userID string) (int, error)

	// 🆕 ---- 战令系统 ----
	// GetActiveSeason 获取当前活跃赛季
	GetActiveSeason() (*model.BattlePassSeason, error)
	// GetUserBattlePass 获取用户战令进度
	GetUserBattlePass(userID string, seasonID int) (*model.BattlePass, error)
	// BuyBattlePass 购买付费战令
	BuyBattlePass(userID string, seasonID int, pointsCost int64) (*model.BattlePass, error)
	// AddBattlePassXP 增加战令经验值，返回更新后的等级和xp
	AddBattlePassXP(userID string, seasonID int, xp int) (*model.BattlePass, error)
	// ClaimBattlePassReward 领取战令等级奖励
	ClaimBattlePassReward(userID string, seasonID int, level int) (bool, error)
	// GetBattlePassTasks 获取战令任务列表
	GetBattlePassTasks(seasonID int) ([]model.BattlePassTask, error)
	// GetBattlePassTaskProgress 获取用户任务进度
	GetBattlePassTaskProgress(userID string, seasonID int) ([]model.BattlePassTaskProgress, error)
	// UpdateTaskProgress 更新任务进度（增加1）
	UpdateTaskProgress(userID string, taskID int) error
	// GetBattlePassRewards 获取战令奖励配置
	GetBattlePassRewards(seasonID int) ([]model.BattlePassReward, error)

	// 🆕 ---- 社交裂变 辅助方法 ----
	// GenerateID 生成唯一ID
	GenerateID() string

	// 🆕 ---- 社交裂变：邀请助力 ----
	// CreateInviteRecord 记录邀请关系
	CreateInviteRecord(inviterID, inviteeID string) (*model.InviteRecord, error)
	// GetInviteRecords 获取用户的邀请记录
	GetInviteRecords(userID string) ([]model.InviteRecord, error)
	// GetInviteStats 获取邀请统计
	GetInviteStats(userID string) (*model.InviteStats, error)
	// GetOrCreateAssistProgress 获取或创建助力进度
	GetOrCreateAssistProgress(inviterID string, assistType model.AssistType) (*model.AssistProgress, error)
	// IsAssistActionRecorded 检查某个好友今天是否已助力过（防刷）
	IsAssistActionRecorded(inviterID, helperID string, assistType model.AssistType) (bool, error)
	// RecordAssistAction 记录好友助力动作
	RecordAssistAction(inviterID, helperID string, assistType model.AssistType) error
	// IncrementAssistProgress 增加助力进度（+1），返回更新后的进度
	IncrementAssistProgress(inviterID string, assistType model.AssistType) (*model.AssistProgress, error)
	// ClaimAssistReward 领取助力奖励（标记已领取）
	ClaimAssistReward(inviterID string, assistType model.AssistType) (*model.AssistProgress, error)
	// GetAssistProgress 查询助力进度
	GetAssistProgress(inviterID string, assistType model.AssistType) (*model.AssistProgress, error)

	// 🆕 ---- 社交裂变：组队开盒 ----
	// CreateTeam 创建队伍
	CreateTeam(captainID string, input model.CreateTeamRequest) (*model.Team, error)
	// JoinTeam 加入队伍
	JoinTeam(userID string, teamID string) (*model.TeamMember, error)
	// LeaveTeam 离开队伍
	LeaveTeam(userID, teamID string) error
	// GetTeam 获取队伍信息
	GetTeam(teamID string) (*model.Team, error)
	// GetTeamMembers 获取队伍成员列表
	GetTeamMembers(teamID string) ([]model.TeamMember, error)
	// GetUserActiveTeam 获取用户当前活跃队伍
	GetUserActiveTeam(userID string) (*model.Team, error)
	// AddTeamDraw 队伍开盒次数+1
	AddTeamDraw(userID, teamID string) (int, error) // 返回队伍总次数
	// CompleteTeam 完成队伍目标，发放奖励
	CompleteTeam(teamID string) (*model.TeamReward, error)
	// ExpireTeam 过期队伍
	ExpireTeam(teamID string) error
	// GetExpiredTeams 获取已过期但未处理的队伍
	GetExpiredTeams() ([]model.Team, error)

	// 🆕 ---- 社交裂变：礼物赠送 ----
	// CreateGift 创建礼物赠送记录
	CreateGift(giverID, receiverID, prizeID, campaignID string) (*model.GiftRecord, error)
	// GetGift 获取礼物详情
	GetGift(giftID string) (*model.GiftRecord, error)
	// ReceiveGift 接收礼物
	ReceiveGift(giftID string) (*model.ReceiveGiftResult, error)
	// GetUserGifts 获取用户收到的待领取礼物
	GetUserGifts(userID string) ([]model.GiftRecord, error)
	// GetUserSentGifts 获取用户发送的礼物
	GetUserSentGifts(userID string) ([]model.GiftRecord, error)
	// ExpireGift 过期未领取的礼物
	ExpireGift(giftID string) error

	// 🆕 ---- 社交裂变：分享卡片 ----
	// CreateShareCard 创建分享卡片
	CreateShareCard(userID string, cardType string, title, description string, prizeName, prizeLevel, inviteLink string) (*model.ShareCard, error)
	// GetShareCards 获取用户分享卡片
	GetShareCards(userID string) ([]model.ShareCard, error)

	// 🆕 ---- 限时商店 + 付费道具 ----
	// GetShopItems 获取商店商品列表
	GetShopItems() []model.ShopItem
	// BuyShopItem 购买商店商品，返回购买结果
	BuyShopItem(userID string, itemID string, quantity int) (*model.BuyShopItemResult, error)
	// GetUserItemQty 查询用户某种道具数量
	GetUserItemQty(userID string, itemType model.ItemType) (int, error)
	// AddUserItem 给用户添加道具
	AddUserItem(userID string, itemType model.ItemType, qty int) error
	// UseUserItem 消耗用户道具，返回是否成功
	UseUserItem(userID string, itemType model.ItemType, qty int) (bool, error)
	// GetUserItems 获取用户所有道具
	GetUserItems(userID string) ([]model.UserItem, error)

	// 🆕 ---- 首充礼包 ----
	// GetFirstRechargeStatus 获取用户首充状态（哪些礼包已领）
	GetFirstRechargeStatus(userID string) (*model.UserFirstRecharge, error)
	// ClaimFirstRecharge 领取首充礼包
	ClaimFirstRecharge(userID string, packID string) (*model.ClaimFirstRechargeResult, error)

	// 🆕 ---- v1.6 碎片拼图 ----
	// GetActivePuzzleTemplates 获取当前活跃的拼图模板列表
	GetActivePuzzleTemplates() []model.PuzzleTemplate
	// GetPuzzleTemplate 获取拼图模板详情
	GetPuzzleTemplate(templateID string) (*model.PuzzleTemplate, error)
	// GetOrCreatePuzzleProgress 获取或创建用户拼图进度
	GetOrCreatePuzzleProgress(userID, templateID string) (*model.PuzzleProgress, error)
	// AddPuzzlePiece 收集碎片（添加位置索引到用户进度），返回是否是新收集
	AddPuzzlePiece(userID, templateID string, pieceIndex int) (bool, error)
	// ComposePuzzle 合成拼图（集齐所有碎片后兑换奖励）
	ComposePuzzle(userID, templateID string) (*model.ComposePuzzleResult, error)
	// GetPuzzleInfo 获取拼图详细信息（进度+模板）
	GetPuzzleInfo(userID, templateID string) (*model.PuzzleInfo, error)
	// CreatePuzzleTeam 创建拼图小队
	CreatePuzzleTeam(captainID, templateID string) (*model.PuzzleTeam, error)
	// JoinPuzzleTeam 加入拼图小队
	JoinPuzzleTeam(userID, teamID string) (*model.PuzzleTeam, error)
	// GetPuzzleTeam 获取拼图小队信息
	GetPuzzleTeam(teamID string) (*model.PuzzleTeam, error)
	// GetUserPuzzleTeams 获取用户加入的拼图小队
	GetUserPuzzleTeams(userID string) ([]model.PuzzleTeam, error)
	// GetUserPuzzleProgresses 获取用户所有拼图进度
	GetUserPuzzleProgresses(userID string) ([]model.PuzzleInfo, error)
	// SharePuzzlePiece 在拼图小队中共享碎片
	SharePuzzlePiece(userID, teamID string, pieceIndex int) (bool, error)

	// 🆕 ---- v1.6 预约/抢购 ----
	// GetFlashSales 获取抢购活动列表
	GetFlashSales() []model.FlashSale
	// GetFlashSale 获取抢购活动详情
	GetFlashSale(flashID string) (*model.FlashSale, error)
	// SubscribeFlash 预约抢购活动
	SubscribeFlash(userID, flashID string) error
	// UnsubscribeFlash 取消预约
	UnsubscribeFlash(userID, flashID string) error
	// IsFlashSubscribed 检查用户是否已预约
	IsFlashSubscribed(userID, flashID string) (bool, error)
	// PurchaseFlash 执行抢购（扣积分减库存）
	PurchaseFlash(userID, flashID string) (*model.FlashPurchaseResult, error)
	// GetUserFlashSubscriptions 获取用户预约列表
	GetUserFlashSubscriptions(userID string) ([]model.FlashSubscription, error)
	// CreateFlashSale 创建抢购活动（管理端）
	CreateFlashSale(input model.FlashSale) (*model.FlashSale, error)
	// UpdateFlashSaleStatus 更新抢购状态
	UpdateFlashSaleStatus(flashID, status string) error

	// 🆕 ---- v2.0 活动系统 ----
	// GetActiveActivities 获取当前活跃活动列表
	GetActiveActivities() []model.Activity
	// GetAllActivities 获取所有活动（管理端）
	GetAllActivities() []model.Activity
	// GetActivity 获取活动详情
	GetActivity(activityID string) (*model.Activity, error)
	// CreateActivity 创建活动
	CreateActivity(input model.ActivityCreateRequest) (*model.Activity, error)
	// UpdateActivity 更新活动
	UpdateActivity(activityID string, input model.ActivityUpdateRequest) (*model.Activity, error)
	// DeleteActivity 删除活动
	DeleteActivity(activityID string) error
	// GetActivityRewards 获取活动奖励列表
	GetActivityRewards(activityID string) ([]model.ActivityReward, error)
	// JoinActivity 用户参与活动
	JoinActivity(userID, activityID string) (*model.ActivityParticipation, error)
	// GetUserActivityParticipation 获取用户活动参与记录
	GetUserActivityParticipation(userID, activityID string) (*model.ActivityParticipation, error)
	// GetUserActivityParticipations 获取用户所有活动参与
	GetUserActivityParticipations(userID string) ([]model.ActivityParticipation, error)
	// ClaimActivityReward 领取活动奖励
	ClaimActivityReward(userID, activityID, rewardID string) (*model.ActivityReward, error)
}
