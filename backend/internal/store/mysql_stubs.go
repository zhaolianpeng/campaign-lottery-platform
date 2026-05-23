package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"campaign-lottery-platform/backend/internal/model"
)

// ============================================================
// Stub methods — use memory store for these features
// ============================================================

// ---- 集卡系统 stubs ----

func (store *MySQLStore) DailyCheckIn(userID string, points int64) (*model.CheckInResult, error) {
	return nil, fmt.Errorf("mysql daily check-in not implemented, use memory store")
}

func (store *MySQLStore) GetCheckInStreak(userID string) (int, error) {
	return 0, fmt.Errorf("mysql check-in streak not implemented, use memory store")
}

func (store *MySQLStore) CheckCollectionCompletion(userID, campaignID string) (*model.CollectionReward, error) {
	return nil, fmt.Errorf("mysql collection completion not implemented, use memory store")
}

func (store *MySQLStore) GrantCollectionReward(userID string, reward *model.CollectionReward) error {
	return fmt.Errorf("mysql collection reward not implemented, use memory store")
}

func (store *MySQLStore) GetLeaderboard(limit int) ([]model.LeaderboardEntry, error) {
	return nil, fmt.Errorf("mysql leaderboard not implemented, use memory store")
}

func (store *MySQLStore) GetCampaignHint(campaignID string) *model.HintMessage {
	return nil
}

func (store *MySQLStore) ShareReward(userID string, points int64) (*model.ShareRewardResult, error) {
	return nil, fmt.Errorf("mysql share reward not implemented, use memory store")
}

func (store *MySQLStore) GetShareDailyCount(userID string) (int, error) {
	return 0, fmt.Errorf("mysql share daily count not implemented, use memory store")
}

func (store *MySQLStore) GetPrizeCount(userID, prizeID string) (int, error) {
	return 0, fmt.Errorf("mysql prize count not implemented, use memory store")
}

func (store *MySQLStore) BlendPrizes(userID string, sourcePrizeID string, campaignID string) (*model.BlendResult, error) {
	return nil, fmt.Errorf("mysql blend prizes not implemented, use memory store")
}

// ---- 积分/会员 stubs (supplementary) ----

func (store *MySQLStore) UpdateUserMember(member *model.UserMember) error {
	return fmt.Errorf("mysql update user member not implemented, use memory store")
}

func (store *MySQLStore) LogPoints(userID string, points int64, balance int64, reason, remark string) error {
	return fmt.Errorf("mysql log points not implemented, use memory store")
}

// ---- 月卡系统 stubs ----

func (store *MySQLStore) GetMonthCard(userID string) (*model.MonthCard, error) {
	return nil, fmt.Errorf("mysql monthcard not implemented, use memory store")
}

func (store *MySQLStore) BuyMonthCard(userID string, cardType model.MonthCardType, pointsCost int64) (*model.MonthCard, error) {
	return nil, fmt.Errorf("mysql monthcard not implemented, use memory store")
}

func (store *MySQLStore) UseFreeDraw(userID string) (int, error) {
	return 0, fmt.Errorf("mysql monthcard not implemented, use memory store")
}

func (store *MySQLStore) GetTodayFreeDrawUsed(userID string) (int, error) {
	return 0, fmt.Errorf("mysql monthcard not implemented, use memory store")
}

// ---- 用户卡（月卡/周卡/季卡）stubs ----

func (store *MySQLStore) GetUserCard(userID string) (*model.UserCard, error) {
	return nil, nil
}

func (store *MySQLStore) BuyCard(userID string, cardType model.CardType) (*model.BuyCardResult, error) {
	return nil, fmt.Errorf("mysql buy card not implemented, use memory store")
}

func (store *MySQLStore) ConsumeFreeDraw(userID string) (bool, error) {
	return false, nil
}

func (store *MySQLStore) GetFreeDrawRemaining(userID string) (int, error) {
	return 0, nil
}

// ---- 战令系统 stubs ----

func (store *MySQLStore) GetActiveSeason() (*model.BattlePassSeason, error) {
	return nil, fmt.Errorf("mysql battlepass not implemented, use memory store")
}

func (store *MySQLStore) GetUserBattlePass(userID string, seasonID int) (*model.BattlePass, error) {
	return nil, fmt.Errorf("mysql battlepass not implemented, use memory store")
}

func (store *MySQLStore) BuyBattlePass(userID string, seasonID int, pointsCost int64) (*model.BattlePass, error) {
	return nil, fmt.Errorf("mysql battlepass not implemented, use memory store")
}

func (store *MySQLStore) AddBattlePassXP(userID string, seasonID int, xp int) (*model.BattlePass, error) {
	return nil, fmt.Errorf("mysql battlepass not implemented, use memory store")
}

func (store *MySQLStore) ClaimBattlePassReward(userID string, seasonID int, level int) (bool, error) {
	return false, fmt.Errorf("mysql battlepass not implemented, use memory store")
}

func (store *MySQLStore) GetBattlePassTasks(seasonID int) ([]model.BattlePassTask, error) {
	return nil, fmt.Errorf("mysql battlepass not implemented, use memory store")
}

func (store *MySQLStore) GetBattlePassTaskProgress(userID string, seasonID int) ([]model.BattlePassTaskProgress, error) {
	return nil, fmt.Errorf("mysql battlepass not implemented, use memory store")
}

func (store *MySQLStore) UpdateTaskProgress(userID string, taskID int) error {
	return fmt.Errorf("mysql battlepass not implemented, use memory store")
}

func (store *MySQLStore) GetBattlePassRewards(seasonID int) ([]model.BattlePassReward, error) {
	return nil, fmt.Errorf("mysql battlepass not implemented, use memory store")
}

// ---- 商店 + 道具 stubs ----

func (store *MySQLStore) GetShopItems() []model.ShopItem {
	return nil
}

func (store *MySQLStore) BuyShopItem(userID string, itemID string, quantity int) (*model.BuyShopItemResult, error) {
	return nil, fmt.Errorf("mysql shop not implemented, use memory store")
}

func (store *MySQLStore) GetUserItemQty(userID string, itemType model.ItemType) (int, error) {
	return 0, fmt.Errorf("mysql user items not implemented, use memory store")
}

func (store *MySQLStore) AddUserItem(userID string, itemType model.ItemType, qty int) error {
	return fmt.Errorf("mysql user items not implemented, use memory store")
}

func (store *MySQLStore) UseUserItem(userID string, itemType model.ItemType, qty int) (bool, error) {
	return false, fmt.Errorf("mysql user items not implemented, use memory store")
}

func (store *MySQLStore) GetUserItems(userID string) ([]model.UserItem, error) {
	return nil, fmt.Errorf("mysql user items not implemented, use memory store")
}

// ---- 首充礼包 stubs ----

func (store *MySQLStore) GetFirstRechargeStatus(userID string) (*model.UserFirstRecharge, error) {
	return nil, fmt.Errorf("mysql first recharge not implemented, use memory store")
}

func (store *MySQLStore) ClaimFirstRecharge(userID string, packID string) (*model.ClaimFirstRechargeResult, error) {
	return nil, fmt.Errorf("mysql first recharge not implemented, use memory store")
}

// ---- 社交裂变 stubs ----

func (store *MySQLStore) GenerateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (store *MySQLStore) CreateInviteRecord(inviterID, inviteeID string) (*model.InviteRecord, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetInviteRecords(userID string) ([]model.InviteRecord, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetInviteStats(userID string) (*model.InviteStats, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetOrCreateAssistProgress(inviterID string, assistType model.AssistType) (*model.AssistProgress, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) IsAssistActionRecorded(inviterID, helperID string, assistType model.AssistType) (bool, error) {
	return false, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) RecordAssistAction(inviterID, helperID string, assistType model.AssistType) error {
	return fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) IncrementAssistProgress(inviterID string, assistType model.AssistType) (*model.AssistProgress, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) ClaimAssistReward(inviterID string, assistType model.AssistType) (*model.AssistProgress, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetAssistProgress(inviterID string, assistType model.AssistType) (*model.AssistProgress, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

// ---- 社交裂变：组队开盒 stubs ----

func (store *MySQLStore) CreateTeam(captainID string, input model.CreateTeamRequest) (*model.Team, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) JoinTeam(userID string, teamID string) (*model.TeamMember, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) LeaveTeam(userID, teamID string) error {
	return fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetTeam(teamID string) (*model.Team, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetTeamMembers(teamID string) ([]model.TeamMember, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetUserActiveTeam(userID string) (*model.Team, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) AddTeamDraw(userID, teamID string) (int, error) {
	return 0, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) CompleteTeam(teamID string) (*model.TeamReward, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) ExpireTeam(teamID string) error {
	return fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetExpiredTeams() ([]model.Team, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

// ---- 社交裂变：礼物赠送 stubs ----

func (store *MySQLStore) CreateGift(giverID, receiverID, prizeID, campaignID string) (*model.GiftRecord, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetGift(giftID string) (*model.GiftRecord, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) ReceiveGift(giftID string) (*model.ReceiveGiftResult, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetUserGifts(userID string) ([]model.GiftRecord, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetUserSentGifts(userID string) ([]model.GiftRecord, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) ExpireGift(giftID string) error {
	return fmt.Errorf("mysql social not implemented, use memory store")
}

// ---- 社交裂变：分享卡片 stubs ----

func (store *MySQLStore) CreateShareCard(userID string, cardType string, title, description string, prizeName, prizeLevel, inviteLink string) (*model.ShareCard, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

func (store *MySQLStore) GetShareCards(userID string) ([]model.ShareCard, error) {
	return nil, fmt.Errorf("mysql social not implemented, use memory store")
}

// ---- 碎片拼图 stubs ----

func (store *MySQLStore) GetActivePuzzleTemplates() []model.PuzzleTemplate {
	return nil
}

func (store *MySQLStore) GetPuzzleTemplate(templateID string) (*model.PuzzleTemplate, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) GetOrCreatePuzzleProgress(userID, templateID string) (*model.PuzzleProgress, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) AddPuzzlePiece(userID, templateID string, pieceIndex int) (bool, error) {
	return false, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) ComposePuzzle(userID, templateID string) (*model.ComposePuzzleResult, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) GetPuzzleInfo(userID, templateID string) (*model.PuzzleInfo, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) CreatePuzzleTeam(captainID, templateID string) (*model.PuzzleTeam, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) JoinPuzzleTeam(userID, teamID string) (*model.PuzzleTeam, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) GetPuzzleTeam(teamID string) (*model.PuzzleTeam, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) GetUserPuzzleTeams(userID string) ([]model.PuzzleTeam, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) GetUserPuzzleProgresses(userID string) ([]model.PuzzleInfo, error) {
	return nil, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

func (store *MySQLStore) SharePuzzlePiece(userID, teamID string, pieceIndex int) (bool, error) {
	return false, fmt.Errorf("mysql puzzle not implemented, use memory store")
}

// ---- 抢购/预约 stubs ----

func (store *MySQLStore) GetFlashSales() []model.FlashSale {
	return nil
}

func (store *MySQLStore) GetFlashSale(flashID string) (*model.FlashSale, error) {
	return nil, fmt.Errorf("mysql flash not implemented, use memory store")
}

func (store *MySQLStore) SubscribeFlash(userID, flashID string) error {
	return fmt.Errorf("mysql flash not implemented, use memory store")
}

func (store *MySQLStore) UnsubscribeFlash(userID, flashID string) error {
	return fmt.Errorf("mysql flash not implemented, use memory store")
}

func (store *MySQLStore) IsFlashSubscribed(userID, flashID string) (bool, error) {
	return false, fmt.Errorf("mysql flash not implemented, use memory store")
}

func (store *MySQLStore) PurchaseFlash(userID, flashID string) (*model.FlashPurchaseResult, error) {
	return nil, fmt.Errorf("mysql flash not implemented, use memory store")
}

func (store *MySQLStore) GetUserFlashSubscriptions(userID string) ([]model.FlashSubscription, error) {
	return nil, fmt.Errorf("mysql flash not implemented, use memory store")
}

func (store *MySQLStore) CreateFlashSale(input model.FlashSale) (*model.FlashSale, error) {
	return nil, fmt.Errorf("mysql flash not implemented, use memory store")
}

func (store *MySQLStore) UpdateFlashSaleStatus(flashID, status string) error {
	return fmt.Errorf("mysql flash not implemented, use memory store")
}

// ---- 活动系统 stubs ----

func (store *MySQLStore) GetActiveActivities() []model.Activity {
	return nil
}

func (store *MySQLStore) GetAllActivities() []model.Activity {
	return nil
}

func (store *MySQLStore) GetActivity(activityID string) (*model.Activity, error) {
	return nil, fmt.Errorf("mysql activity not implemented, use memory store")
}

func (store *MySQLStore) CreateActivity(input model.ActivityCreateRequest) (*model.Activity, error) {
	return nil, fmt.Errorf("mysql activity not implemented, use memory store")
}

func (store *MySQLStore) UpdateActivity(activityID string, input model.ActivityUpdateRequest) (*model.Activity, error) {
	return nil, fmt.Errorf("mysql activity not implemented, use memory store")
}

func (store *MySQLStore) DeleteActivity(activityID string) error {
	return fmt.Errorf("mysql activity not implemented, use memory store")
}

func (store *MySQLStore) GetActivityRewards(activityID string) ([]model.ActivityReward, error) {
	return nil, fmt.Errorf("mysql activity not implemented, use memory store")
}

func (store *MySQLStore) JoinActivity(userID, activityID string) (*model.ActivityParticipation, error) {
	return nil, fmt.Errorf("mysql activity not implemented, use memory store")
}

func (store *MySQLStore) GetUserActivityParticipation(userID, activityID string) (*model.ActivityParticipation, error) {
	return nil, fmt.Errorf("mysql activity not implemented, use memory store")
}

func (store *MySQLStore) GetUserActivityParticipations(userID string) ([]model.ActivityParticipation, error) {
	return nil, fmt.Errorf("mysql activity not implemented, use memory store")
}

func (store *MySQLStore) ClaimActivityReward(userID, activityID, rewardID string) (*model.ActivityReward, error) {
	return nil, fmt.Errorf("mysql activity not implemented, use memory store")
}
