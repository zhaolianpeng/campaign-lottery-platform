package service

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"campaign-lottery-platform/backend/internal/model"
	"campaign-lottery-platform/backend/internal/probability"
	"campaign-lottery-platform/backend/internal/store"
)

// Service encapsulates business logic for the blind box lottery platform.
type Service struct {
	store       store.Store
	pityTracker probability.PityTracker
}

// New creates a new Service with a data store and pity tracker.
func New(dataStore store.Store) *Service {
	return &Service{
		store:       dataStore,
		pityTracker: probability.NewMemoryPityTracker(),
	}
}

// ============================================================
// 用户认证
// ============================================================

func (s *Service) GuestLogin(nickname string) (model.User, model.Session, error) {
	return s.store.CreateGuestSession(nickname)
}

func (s *Service) UserFromToken(token string) (model.User, error) {
	return s.store.UserFromToken(token)
}

// ============================================================
// 盲盒系列
// ============================================================

func (s *Service) CampaignList() []model.Campaign {
	return s.store.Campaigns()
}

// CampaignListWithProgress 返回系列列表，如果提供了token则带用户收集进度
func (s *Service) CampaignListWithProgress(token string) ([]map[string]any, error) {
	campaigns := s.store.Campaigns()
	result := make([]map[string]any, 0, len(campaigns))

	// 尝试获取用户ID（token可选）
	var userID string
	if token != "" {
		user, err := s.store.UserFromToken(token)
		if err == nil {
			userID = user.ID
		}
	}

	for _, campaign := range campaigns {
		item := map[string]any{
			"campaign": campaign,
			"prizes":   s.store.PrizeList(campaign.ID),
		}
		// 如果用户已登录，附带收集进度
		if userID != "" {
			progress, err := s.store.GetSeriesProgress(userID, campaign.ID, campaign.Name)
			if err == nil {
				item["progress"] = progress
			}
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *Service) CampaignPrizeList(campaignID string) []model.Prize {
	return s.store.PrizeList(campaignID)
}

// BlindBoxCampaignProbabilities 返回带公示概率的盲盒系列详情
func (s *Service) BlindBoxCampaignProbabilities(campaignID string) (map[string]any, error) {
	campaign, err := s.store.GetCampaign(campaignID)
	if err != nil {
		return nil, err
	}
	prizes := s.store.PrizeList(campaignID)

	pityCfg := s.buildPityConfig(campaign)
	totalWeight := float64(campaign.MissWeight)
	prizeInfo := make([]map[string]any, 0, len(prizes))
	secretWeight := 0.0

	for _, p := range prizes {
		if p.Status != "active" || p.ProbabilityWeight <= 0 {
			continue
		}
		w := float64(p.ProbabilityWeight)
		totalWeight += w
		if p.Level == model.PrizeLevelSecret {
			secretWeight = w
		}
		prizeInfo = append(prizeInfo, map[string]any{
			"id":            p.ID,
			"name":          p.Name,
			"level":         p.Level,
			"stock":         p.Stock,
			"base_prob":     fmt.Sprintf("%.2f%%", float64(p.ProbabilityWeight)/totalWeight*100),
		})
	}

	result := map[string]any{
		"campaign":     campaign,
		"prizes":       prizeInfo,
		"pity_config":  pityCfg,
		"miss_weight":  campaign.MissWeight,
	}

	if pityCfg.Enabled && secretWeight > 0 {
		// 计算含软保底的期望概率
		result["pity_analysis"] = map[string]any{
			"soft_pity_start":      pityCfg.SoftPityN,
			"hard_pity_at":         pityCfg.HardPityN,
			"pity_factor":          pityCfg.PityFactor,
			"target_prize":         pityCfg.TargetWeight,
			"base_secret_prob":     fmt.Sprintf("%.4f%%", secretWeight/totalWeight*100),
		}
	}

	return result, nil
}

// ============================================================
// 工具函数
// ============================================================

// buildPityConfig 从 Campaign 构建概率引擎的保底配置
func (s *Service) buildPityConfig(campaign model.Campaign) probability.PityConfig {
	pc := campaign.PityConfig
	if !pc.Enabled {
		return probability.PityConfig{Enabled: false}
	}
	if pc.SoftPityN <= 0 {
		pc.SoftPityN = 60
	}
	if pc.PityFactor <= 0 {
		pc.PityFactor = 0.015
	}
	if pc.HardPityN <= 0 {
		pc.HardPityN = 90
	}
	return probability.PityConfig{
		Enabled:      true,
		SoftPityN:    pc.SoftPityN,
		PityFactor:   pc.PityFactor,
		HardPityN:    pc.HardPityN,
		TargetWeight: 0,
	}
}

// isUPPoolActive 检查UP池是否当前生效
func isUPPoolActive(campaign model.Campaign) bool {
	pc := campaign.PityConfig
	if !pc.UPPoolEnabled || pc.UPPrizeID == "" || pc.UPLevel == "" {
		return false
	}
	now := time.Now().UTC()
	if now.Before(pc.UPStartAt) || now.After(pc.UPEndAt) {
		return false
	}
	return true
}

// getUPPrizeInfo 获取UP池目标奖品信息（用于前端展示）
func getUPPrizeInfo(active bool, prizes []model.Prize) map[string]any {
	if !active {
		return nil
	}
	for _, p := range prizes {
		if p.Status == "active" && p.Level == model.PrizeLevelSecret {
			return map[string]any{
				"name":  p.Name,
				"level": p.Level,
			}
		}
	}
	return nil
}

// IsUPPoolActive 外部可调用的UP池检查
func (s *Service) IsUPPoolActive(campaignID string) (bool, error) {
	campaign, err := s.store.GetCampaign(campaignID)
	if err != nil {
		return false, err
	}
	return isUPPoolActive(campaign), nil
}

// UPPoolInfo 获取UP池信息
func (s *Service) UPPoolInfo(token string, campaignID string) (map[string]any, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	campaign, err := s.store.GetCampaign(campaignID)
	if err != nil {
		return nil, err
	}
	active := isUPPoolActive(campaign)
	prizes := s.store.PrizeList(campaignID)
	upInfo := getUPPrizeInfo(active, prizes)

	hasGuarantee := false
	if active {
		hasGuarantee = s.pityTracker.GetUPPoolGuarantee(user.ID, campaignID)
	}

	return map[string]any{
		"active":             active,
		"prize":              upInfo,
		"has_guarantee":      hasGuarantee,
		"up_prize_id":        campaign.PityConfig.UPPrizeID,
		"up_multiplier":      campaign.PityConfig.UPMultiplier,
		"up_level":           campaign.PityConfig.UPLevel,
		"up_start_at":        campaign.PityConfig.UPStartAt,
		"up_end_at":          campaign.PityConfig.UPEndAt,
		"consecutive_misses": s.pityTracker.Get(user.ID, campaignID).ConsecutiveMisses,
	}, nil
}

// ============================================================
// 会员等级
// ============================================================

// calcMemberLevel 根据累计消费积分计算会员等级
func (s *Service) calcMemberLevel(totalSpent int64) model.MemberLevel {
	switch {
	case totalSpent >= 15000:
		return model.MemberDiamond
	case totalSpent >= 5000:
		return model.MemberDiamond
	case totalSpent >= 2000:
		return model.MemberGold
	case totalSpent >= 500:
		return model.MemberSilver
	default:
		return model.MemberNormal
	}
}

// ============================================================
// 原 Draw 兼容
// ============================================================

func (s *Service) Draw(token string, campaignID string) (model.DrawResult, error) {
	cfg := model.DrawConfig{CampaignID: campaignID, DrawCount: 1}
	result, err := s.BlindBoxDraw(token, cfg)
	if err != nil {
		return model.DrawResult{}, err
	}
	if len(result.Draws) == 0 {
		return model.DrawResult{}, fmt.Errorf("draw returned no results")
	}
	d := result.Draws[0]
	prizeName := d.PrizeName
	if !d.IsWin {
		prizeName = "未中奖"
	}
	rec := model.DrawRecord{
		ID:         d.RecordID,
		CampaignID: campaignID,
		PrizeName:  prizeName,
		Result:     map[bool]string{true: "win", false: "miss"}[d.IsWin],
		DrawnAt:    time.Now().UTC(),
		ChanceAfter: result.RemainingChances,
	}
	if d.IsWin {
		rec.PrizeID = &d.PrizeID
	}
	return model.DrawResult{
		Record:           rec,
		RemainingChances: result.RemainingChances,
	}, nil
}

func (s *Service) UserDrawRecords(token string) ([]model.DrawRecord, error) {
	return s.store.UserDrawRecords(token)
}

// ============================================================
// 管理端
// ============================================================

func (s *Service) AdminLogin(username string, password string) (string, error) {
	return s.store.AdminLogin(username, password)
}

func (s *Service) AdminOverview(token string) (model.AdminOverview, error) {
	return s.store.AdminOverview(token)
}

func (s *Service) AdminDrawRecords(token string) ([]model.DrawRecord, error) {
	return s.store.AdminDrawRecords(token)
}

func (s *Service) AdminCampaigns(token string) ([]model.Campaign, error) {
	return s.store.AdminCampaigns(token)
}

func (s *Service) CreateCampaign(token string, input model.CampaignMutation) (model.Campaign, error) {
	return s.store.CreateCampaign(token, input)
}

func (s *Service) UpdateCampaign(token string, campaignID string, input model.CampaignMutation) (model.Campaign, error) {
	return s.store.UpdateCampaign(token, campaignID, input)
}

func (s *Service) DeleteCampaign(token string, campaignID string) error {
	return s.store.DeleteCampaign(token, campaignID)
}

func (s *Service) AdminPrizes(token string, campaignID string) ([]model.Prize, error) {
	return s.store.AdminPrizes(token, campaignID)
}

func (s *Service) CreatePrize(token string, campaignID string, input model.PrizeMutation) (model.Prize, error) {
	return s.store.CreatePrize(token, campaignID, input)
}

func (s *Service) UpdatePrize(token string, prizeID string, input model.PrizeMutation) (model.Prize, error) {
	return s.store.UpdatePrize(token, prizeID, input)
}

func (s *Service) DeletePrize(token string, prizeID string) error {
	return s.store.DeletePrize(token, prizeID)
}

func (s *Service) FulfillmentTasks(token string) ([]model.FulfillmentTask, error) {
	return s.store.FulfillmentTasks(token)
}

func (s *Service) UpdateFulfillmentTask(token string, taskID int64, input model.FulfillmentTaskMutation) (model.FulfillmentTask, error) {
	return s.store.UpdateFulfillmentTask(token, taskID, input)
}

// DrawStatistics 获取抽奖统计数据
func (s *Service) DrawStatistics(token string, campaignID string) (*model.DrawStatistics, error) {
	return s.store.GetDrawStatistics(token, campaignID)
}

// AdminUpdatePityConfig 管理员更新活动的保底配置
func (s *Service) AdminUpdatePityConfig(token string, campaignID string, cfg model.PityConfig) (*model.Campaign, error) {
	if _, err := s.store.AdminOverview(token); err != nil {
		return nil, err
	}
	campaign, err := s.store.GetCampaign(campaignID)
	if err != nil {
		return nil, err
	}
	campaign.PityConfig = cfg
	mutation := model.CampaignMutation{
		Name: campaign.Name, Slug: campaign.Slug, Status: campaign.Status,
		StartsAt: campaign.StartsAt, EndsAt: campaign.EndsAt,
		DailyDrawLimit: campaign.DailyDrawLimit, MissWeight: campaign.MissWeight,
		BannerImageURL: campaign.BannerImageURL, CampaignSummary: campaign.CampaignSummary,
		PityConfig: cfg,
	}
	camp, err := s.store.UpdateCampaign(token, campaignID, mutation)
	if err != nil {
		return nil, err
	}
	return &camp, nil
}

// AdminGetCampaign 管理员获取活动详情
func (s *Service) AdminGetCampaign(token, campaignID string) (*model.Campaign, error) {
	if _, err := s.store.AdminOverview(token); err != nil {
		return nil, err
	}
	campaign, err := s.store.GetCampaign(campaignID)
	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

// ============================================================
// v1.5 社交裂变 - 分享卡片
// ============================================================

// CreateShareCard 创建分享卡片
func (s *Service) CreateShareCard(token string, cardType string, prizeName, prizeLevel string) (*model.ShareCard, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	title, description := s.getShareCardText(cardType, prizeName, prizeLevel)
	inviteLink := fmt.Sprintf("https://boxmagic.app/invite?from=%s", user.ID)
	return s.store.CreateShareCard(user.ID, cardType, title, description, prizeName, prizeLevel, inviteLink)
}

// GetShareCards 获取我的分享卡片
func (s *Service) GetShareCards(token string) ([]model.ShareCard, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetShareCards(user.ID)
}

// getShareCardText 根据卡片类型生成文案
func (s *Service) getShareCardText(cardType, prizeName, prizeLevel string) (string, string) {
	switch cardType {
	case "draw_win":
		levelLabel := map[string]string{"common": "普通", "rare": "稀有", "secret": "隐藏", "limited": "限定"}
		lbl := levelLabel[prizeLevel]
		if lbl == "" {
			lbl = prizeLevel
		}
		if prizeName != "" {
			return fmt.Sprintf("🎉 我抽到了「%s」(%s)！", prizeName, lbl),
				fmt.Sprintf("第N抽开出%s款，运气爆棚！你也来试试吧 ✨", lbl)
		}
		return "🎉 我抽到了稀有盲盒！", "运气爆棚！你也来试试吧 ✨"
	case "collection":
		return "🏆 我集齐了一套系列！", "集齐整套盲盒，成就感满满！"
	case "craft":
		return "🔮 合成成功！", "我用重复盲盒合成了更稀有的款式！"
	case "team":
		return "🤝 组队开盒，奖励翻倍！", "加入我的队伍，一起开盒拿大奖！"
	default:
		return "🎁 来BOX·MAGIC抽盲盒！", "超多精美盲盒等你来抽，首抽免费！"
	}
}

// ---- 邀请助力 ----

// GenerateInviteLink 生成邀请链接
func (s *Service) GenerateInviteLink(token string) (*model.ShareCard, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.CreateShareCard(user.ID, "invite",
		"🎁 好友邀请你免费抽盲盒！",
		"点击链接注册，你和好友各得50积分！",
		"", "",
		fmt.Sprintf("https://boxmagic.app/invite?from=%s", user.ID))
}

// GetInviteRecords 获取邀请记录
func (s *Service) GetInviteRecords(token string) ([]model.InviteRecord, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetInviteRecords(user.ID)
}

// GetInviteStats 获取邀请统计
func (s *Service) GetInviteStats(token string) (*model.InviteStats, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetInviteStats(user.ID)
}

// RecordInvite 记录新用户通过邀请注册
func (s *Service) RecordInvite(inviterID, inviteeID string) (*model.InviteRecord, error) {
	return s.store.CreateInviteRecord(inviterID, inviteeID)
}

// ---- 好友助力 ----

// AssistAction 好友助力
func (s *Service) AssistAction(token string, assistType model.AssistType, helperID string) (*model.AssistProgress, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	inviterID := user.ID

	// 防刷检查：同一个好友24h内只能助力一次同类型
	recorded, err := s.store.IsAssistActionRecorded(inviterID, helperID, assistType)
	if err != nil {
		return nil, err
	}
	if recorded {
		return nil, fmt.Errorf("该好友今天已经助力过了")
	}

	// 记录助力动作
	if err := s.store.RecordAssistAction(inviterID, helperID, assistType); err != nil {
		return nil, err
	}

	// 增加助力进度
	return s.store.IncrementAssistProgress(inviterID, assistType)
}

// GetAssistProgress 查看助力进度
func (s *Service) GetAssistProgress(token string, assistType model.AssistType) (*model.AssistProgress, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	progress, err := s.store.GetOrCreateAssistProgress(user.ID, assistType)
	if err != nil {
		return nil, err
	}
	return progress, nil
}

// GetAssistAllProgress 查看所有助力进度
func (s *Service) GetAssistAllProgress(token string) (map[string]*model.AssistProgress, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	result := make(map[string]*model.AssistProgress)
	for _, at := range []model.AssistType{model.AssistFreeDraw, model.AssistPityReduce, model.AssistCraftBoost} {
		p, _ := s.store.GetOrCreateAssistProgress(user.ID, at)
		if p != nil {
			result[string(at)] = p
		}
	}
	return result, nil
}

// ClaimAssistReward 领取助力奖励
func (s *Service) ClaimAssistReward(token string, assistType model.AssistType) (*model.AssistClaimResult, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	progress, err := s.store.GetAssistProgress(user.ID, assistType)
	if err != nil {
		return nil, err
	}
	if progress == nil {
		return nil, fmt.Errorf("没有进行中的助力活动")
	}
	if progress.Claimed {
		return nil, fmt.Errorf("该助力奖励已领取")
	}
	if progress.Current < progress.TargetCount {
		return nil, fmt.Errorf("助力未完成，需要 %d 次，当前 %d 次", progress.TargetCount, progress.Current)
	}

	// 标记已领取
	if _, err := s.store.ClaimAssistReward(user.ID, assistType); err != nil {
		return nil, err
	}

	// 发放奖励
	switch assistType {
	case model.AssistFreeDraw:
		s.store.AddUserItem(user.ID, model.ItemFreeDraw, 1)
		return &model.AssistClaimResult{
			AssistType: assistType, RewardType: "free_draw",
			Description: "助力完成！获得1次免费抽奖机会 🎉",
		}, nil
	case model.AssistPityReduce:
		return &model.AssistClaimResult{
			AssistType: assistType, RewardType: "pity_reduce",
			Description: "助力完成！保底次数减少10次 ⚡",
		}, nil
	case model.AssistCraftBoost:
		return &model.AssistClaimResult{
			AssistType: assistType, RewardType: "craft_boost",
			Description: "助力完成！下次合成概率+20% 🎯",
		}, nil
	}
	return nil, fmt.Errorf("未知助力类型")
}

// ---- 组队开盒 ----

// CreateTeam 创建队伍
func (s *Service) CreateTeam(token string, input model.CreateTeamRequest) (*model.TeamInfo, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		input.Name = user.Nickname + "的队伍"
	}
	if input.MaxMembers < 2 {
		input.MaxMembers = 2
	}
	if input.MaxMembers > 5 {
		input.MaxMembers = 5
	}
	if input.GoalDraws < 10 {
		input.GoalDraws = 10
	}

	team, err := s.store.CreateTeam(user.ID, input)
	if err != nil {
		return nil, err
	}
	return s.getTeamInfo(team.ID)
}

// JoinTeam 加入队伍
func (s *Service) JoinTeam(token string, teamID string) (*model.TeamInfo, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	if _, err := s.store.JoinTeam(user.ID, teamID); err != nil {
		return nil, err
	}
	return s.getTeamInfo(teamID)
}

// LeaveTeam 离开队伍
func (s *Service) LeaveTeam(token string) error {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return err
	}
	team, err := s.store.GetUserActiveTeam(user.ID)
	if err != nil {
		return err
	}
	if team == nil {
		return fmt.Errorf("你不在任何队伍中")
	}
	return s.store.LeaveTeam(user.ID, team.ID)
}

// GetMyTeam 获取我的队伍信息
func (s *Service) GetMyTeam(token string) (*model.TeamInfo, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	team, err := s.store.GetUserActiveTeam(user.ID)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, nil
	}
	return s.getTeamInfo(team.ID)
}

// GetTeamInfo 获取指定队伍信息
func (s *Service) GetTeamInfo(teamID string) (*model.TeamInfo, error) {
	return s.getTeamInfo(teamID)
}

// getTeamInfo 组装队伍信息
func (s *Service) getTeamInfo(teamID string) (*model.TeamInfo, error) {
	team, err := s.store.GetTeam(teamID)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, nil
	}
	members, err := s.store.GetTeamMembers(teamID)
	if err != nil {
		return nil, err
	}

	captainName := "队长"
	remainingHours := int(time.Until(team.ExpiresAt).Hours())
	if remainingHours < 0 {
		remainingHours = 0
	}

	reward, _ := s.store.CompleteTeam(teamID)

	return &model.TeamInfo{
		Team:           team,
		Members:        members,
		CaptainName:    captainName,
		RemainingHours: remainingHours,
		Reward:         reward,
	}, nil
}

// AddTeamDrawHook 抽奖后调用，累计队伍开盒次数
func (s *Service) AddTeamDrawHook(userID string) {
	team, err := s.store.GetUserActiveTeam(userID)
	if err != nil || team == nil {
		return
	}
	total, err := s.store.AddTeamDraw(userID, team.ID)
	if err != nil {
		return
	}
	if total >= team.GoalDraws {
		s.store.CompleteTeam(team.ID)
	}
}

// ---- 礼物赠送 ----

// SendGift 赠送盲盒给好友
func (s *Service) SendGift(token string, input model.SendGiftRequest) (*model.GiftRecord, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	if input.ReceiverID == "" {
		return nil, fmt.Errorf("请指定接收者")
	}
	if input.PrizeID == "" {
		return nil, fmt.Errorf("请指定要赠送的盲盒")
	}
	return s.store.CreateGift(user.ID, input.ReceiverID, input.PrizeID, input.CampaignID)
}

// ReceiveGift 接收礼物
func (s *Service) ReceiveGift(token string, giftID string) (*model.ReceiveGiftResult, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	gift, err := s.store.GetGift(giftID)
	if err != nil {
		return nil, err
	}
	if gift == nil {
		return nil, fmt.Errorf("礼物不存在")
	}
	if gift.ReceiverID != user.ID {
		return nil, fmt.Errorf("这不是给你的礼物")
	}
	if gift.Status != "sent" {
		return nil, fmt.Errorf("礼物已%s，无法领取", gift.Status)
	}
	return s.store.ReceiveGift(giftID)
}

// GetMyGifts 获取我的待收礼物
func (s *Service) GetMyGifts(token string) ([]model.GiftRecord, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetUserGifts(user.ID)
}

// GetSentGifts 获取我送出的礼物
func (s *Service) GetSentGifts(token string) ([]model.GiftRecord, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetUserSentGifts(user.ID)
}

// GetGiftDetail 获取礼物详情
func (s *Service) GetGiftDetail(giftID string) (*model.GiftRecord, error) {
	return s.store.GetGift(giftID)
}

// ============================================================
// 拼图系统
// ============================================================

// GetPuzzleTemplates 获取当前活跃的拼图模板列表
func (s *Service) GetPuzzleTemplates(token string) ([]model.PuzzleTemplate, error) {
	_, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetActivePuzzleTemplates(), nil
}

// GetPuzzleProgress 获取用户指定拼图进度
func (s *Service) GetPuzzleProgress(token, templateID string) (*model.PuzzleInfo, error) {
	user, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetPuzzleInfo(user.ID, templateID)
}

// GetAllPuzzleProgress 获取用户所有拼图进度
func (s *Service) GetAllPuzzleProgress(token string) ([]model.PuzzleInfo, error) {
	user, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetUserPuzzleProgresses(user.ID)
}

// AddPuzzlePiece 添加碎片
func (s *Service) AddPuzzlePiece(token, templateID string, pieceIndex int) (*model.PuzzlePieceDrop, error) {
	user, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	template, err := s.store.GetPuzzleTemplate(templateID)
	if err != nil {
		return nil, err
	}
	if pieceIndex < 0 || pieceIndex >= template.TotalPieces {
		return nil, fmt.Errorf("碎片索引超出范围：0-%d", template.TotalPieces-1)
	}
	isNew, err := s.store.AddPuzzlePiece(user.ID, templateID, pieceIndex)
	if err != nil {
		return nil, err
	}
	pieceName := ""
	if pieceIndex >= 0 && pieceIndex < len(template.PieceNames) {
		pieceName = template.PieceNames[pieceIndex]
	}
	return &model.PuzzlePieceDrop{
		TemplateID: templateID,
		PieceIndex: pieceIndex,
		PieceName:  pieceName,
		IsNew:      isNew,
	}, nil
}

// ComposePuzzle 合成拼图
func (s *Service) ComposePuzzle(token, templateID string) (*model.ComposePuzzleResult, error) {
	user, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.ComposePuzzle(user.ID, templateID)
}

// GetDrawPuzzleDrop 抽奖时概率掉落拼图碎片
func (s *Service) GetDrawPuzzleDrop(token, campaignID string) (*model.PuzzlePieceDrop, error) {
	user, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	templates := s.store.GetActivePuzzleTemplates()
	var matched *model.PuzzleTemplate
	for i := range templates {
		if templates[i].CampaignID == campaignID {
			matched = &templates[i]
			break
		}
	}
	if matched == nil {
		return nil, nil
	}
	// 30% 概率掉落
	if rand.Float64() >= 0.3 {
		return nil, nil
	}
	pieceIndex := rand.IntN(matched.TotalPieces)
	isNew, err := s.store.AddPuzzlePiece(user.ID, matched.ID, pieceIndex)
	if err != nil {
		return nil, err
	}
	pieceName := ""
	if pieceIndex >= 0 && pieceIndex < len(matched.PieceNames) {
		pieceName = matched.PieceNames[pieceIndex]
	}
	return &model.PuzzlePieceDrop{
		TemplateID: matched.ID,
		PieceIndex: pieceIndex,
		PieceName:  pieceName,
		IsNew:      isNew,
	}, nil
}

// CreatePuzzleTeam 创建拼图小队
func (s *Service) CreatePuzzleTeam(token, templateID string) (*model.PuzzleTeam, error) {
	user, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.CreatePuzzleTeam(user.ID, templateID)
}

// JoinPuzzleTeam 加入拼图小队
func (s *Service) JoinPuzzleTeam(token, teamID string) (*model.PuzzleTeam, error) {
	user, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.JoinPuzzleTeam(user.ID, teamID)
}

// GetMyPuzzleTeams 获取用户加入的拼图小队
func (s *Service) GetMyPuzzleTeams(token string) ([]model.PuzzleTeam, error) {
	user, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetUserPuzzleTeams(user.ID)
}

// GetPuzzleTeamInfo 获取拼图小队信息
func (s *Service) GetPuzzleTeamInfo(teamID string) (*model.PuzzleTeam, error) {
	return s.store.GetPuzzleTeam(teamID)
}

// ============================================================
// 抢购系统
// ============================================================

// levelOrder 会员等级排序（数字越大等级越高）
var levelOrder = map[string]int{
	"normal":  0,
	"silver":  1,
	"gold":    2,
	"diamond": 3,
}

// meetsLevelRequirement 检查用户等级是否满足最低要求
func meetsLevelRequirement(userLevel, minLevel string) bool {
	u, ok1 := levelOrder[userLevel]
	m, ok2 := levelOrder[minLevel]
	if !ok1 || !ok2 {
		return false
	}
	return u >= m
}

// GetFlashList 获取抢购活动列表（含订阅状态和资格）
func (s *Service) GetFlashList(token string) ([]model.FlashListInfo, error) {
	user, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	flashes := s.store.GetFlashSales()
	member, err := s.store.GetUserMember(user.ID)
	if err != nil {
		return nil, err
	}
	result := make([]model.FlashListInfo, 0, len(flashes))
	for _, f := range flashes {
		subscribed, _ := s.store.IsFlashSubscribed(user.ID, f.ID)
		purchasable := meetsLevelRequirement(string(member.Level), f.MinVipLevel) &&
			member.TotalDraws >= f.MinTotalDraws
		result = append(result, model.FlashListInfo{
			Flash:       &f,
			Subscribed:  subscribed,
			Purchasable: purchasable,
		})
	}
	return result, nil
}

// SubscribeFlash 预约抢购
func (s *Service) SubscribeFlash(token, flashID string) error {
	user, err := s.UserFromToken(token)
	if err != nil {
		return err
	}
	return s.store.SubscribeFlash(user.ID, flashID)
}

// UnsubscribeFlash 取消预约
func (s *Service) UnsubscribeFlash(token, flashID string) error {
	user, err := s.UserFromToken(token)
	if err != nil {
		return err
	}
	return s.store.UnsubscribeFlash(user.ID, flashID)
}

// PurchaseFlash 执行抢购
func (s *Service) PurchaseFlash(token, flashID string) (*model.FlashPurchaseResult, error) {
	user, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.PurchaseFlash(user.ID, flashID)
}

// GetMyFlashSubscriptions 获取我的预约列表
func (s *Service) GetMyFlashSubscriptions(token string) ([]model.FlashSubscription, error) {
	user, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetUserFlashSubscriptions(user.ID)
}

// ============================================================
// Activity service methods
// ============================================================

// GetActivityList 获取活动列表（前端展示）
func (s *Service) GetActivityList(token string) ([]model.ActivityListInfo, error) {
	user, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	activities := s.store.GetActiveActivities()
	var result []model.ActivityListInfo
	for _, activity := range activities {
		info := model.ActivityListInfo{
			Activity: &activity,
			Joined:   false,
			CanClaim: false,
		}
		participation, _ := s.store.GetUserActivityParticipation(user.ID, activity.ID)
		if participation != nil {
			info.Joined = true
			info.CanClaim = !participation.RewardClaimed
		}
		rewards, _ := s.store.GetActivityRewards(activity.ID)
		if rewards != nil {
			info.Rewards = rewards
		}
		result = append(result, info)
	}
	if result == nil {
		result = []model.ActivityListInfo{}
	}
	return result, nil
}

// GetActivityDetail 获取活动详情
func (s *Service) GetActivityDetail(token, activityID string) (*model.ActivityListInfo, error) {
	user, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	activity, err := s.store.GetActivity(activityID)
	if err != nil {
		return nil, err
	}
	info := &model.ActivityListInfo{
		Activity: activity,
	}
	participation, _ := s.store.GetUserActivityParticipation(user.ID, activityID)
	if participation != nil {
		info.Joined = true
		info.CanClaim = !participation.RewardClaimed
	}
	rewards, _ := s.store.GetActivityRewards(activityID)
	if rewards != nil {
		info.Rewards = rewards
	}
	return info, nil
}

// JoinActivity 用户参与活动
func (s *Service) JoinActivity(token, activityID string) (*model.ActivityParticipation, error) {
	user, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.JoinActivity(user.ID, activityID)
}

// ClaimActivityReward 领取活动奖励
func (s *Service) ClaimActivityReward(token, activityID, rewardID string) (*model.ActivityReward, error) {
	user, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.ClaimActivityReward(user.ID, activityID, rewardID)
}
