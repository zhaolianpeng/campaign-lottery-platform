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
	store        store.Store
	pityTracker  probability.PityTracker
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
			"target_prize":         pityCfg.TargetPrize,
			"base_secret_prob":     fmt.Sprintf("%.4f%%", secretWeight/totalWeight*100),
		}
	}

	return result, nil
}

// ============================================================
// 盲盒抽奖（核心业务逻辑）
// ============================================================

// BlindBoxDraw 执行盲盒抽奖，使用概率引擎（混合模型+软保底）
func (s *Service) BlindBoxDraw(token string, cfg model.DrawConfig) (*model.BlindBoxDrawResult, error) {
	// 1. 验证用户
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}

	// 2. 获取活动
	campaign, err := s.store.GetCampaign(cfg.CampaignID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if now.Before(campaign.StartsAt) || now.After(campaign.EndsAt) || campaign.Status != "online" {
		return nil, store.ErrCampaignInactive
	}

	// 3. 检查每日次数
	remaining, err := s.store.CheckDrawQuota(user.ID, campaign.ID, campaign.DailyDrawLimit)
	if err != nil {
		return nil, err
	}
	if remaining <= 0 {
		return nil, store.ErrNoDrawChances
	}

	drawCount := cfg.DrawCount
	if drawCount <= 0 {
		drawCount = 1
	}
	if drawCount > remaining {
		drawCount = remaining
	}
	isTenPull := drawCount >= 2

	// 检查月卡
	card, _ := s.store.GetUserCard(user.ID)
	freeRemaining := 0
	if card != nil {
		freeRemaining, _ = s.store.GetFreeDrawRemaining(user.ID)
	}

	// 4. 检查并扣减积分（含月卡免费抽+折扣）
	member, err := s.store.GetUserMember(user.ID)
	if err != nil {
		return nil, err
	}
	// 月卡: 先用免费抽
	actualFree := 0
	if freeRemaining > 0 {
		actualFree = freeRemaining
		if actualFree > drawCount {
			actualFree = drawCount
		}
		for i := 0; i < actualFree; i++ {
			s.store.ConsumeFreeDraw(user.ID)
		}
	}

	// 剩余的计费
	chargeCount := drawCount - actualFree
	pointsCost := int64(0)
	if chargeCount > 0 {
		unitPrice := 100
		if isTenPull {
			unitPrice = 95 // 十连基础折扣
		}
		// 月卡用户的额外折扣
		if card != nil {
			cfg := model.CardConfigs[card.CardType]
			unitPrice = int(float64(unitPrice) * cfg.DiscountRate)
		}
		pointsCost = int64(chargeCount * unitPrice)
		if member.Points < pointsCost {
			return nil, store.ErrInsufficientPoints
		}
		member.Points -= pointsCost
	}

	// 5. 构建概率引擎
	prizes := s.store.PrizeList(campaign.ID)
	pw := make([]probability.PrizeWeight, 0, len(prizes))
	secretID := ""
	secretWeight := 0.0
	for _, p := range prizes {
		if p.Status != "active" || p.Stock <= 0 {
			continue
		}
		pw = append(pw, probability.PrizeWeight{
			ID:     p.ID,
			Weight: float64(p.ProbabilityWeight),
			Level:  p.Level,
		})
		if p.Level == model.PrizeLevelSecret {
			secretID = p.ID
			secretWeight = float64(p.ProbabilityWeight)
		}
	}

	pityCfg := s.buildPityConfig(campaign)
	engineCfg := probability.EngineConfig{
		TargetPrizeID: secretID,
		MissWeight:    float64(campaign.MissWeight),
		Pity:          pityCfg,
	}
	engine := probability.NewEngine(engineCfg, pw)

	// 6. 执行抽奖
	var drawResults []probability.DrawResult
	if isTenPull {
		drawResults = engine.DrawMultiple(drawCount, pityCfg, s.pityTracker, user.ID, campaign.ID)
	} else {
		r := engine.Draw(pityCfg, s.pityTracker, user.ID, campaign.ID)
		drawResults = []probability.DrawResult{r}
	}

	// 🆕 7. UP池处理（50/50大小保底）
	upPoolActive := isUPPoolActive(campaign)
	upPrizeInfo := getUPPrizeInfo(upPoolActive, prizes)

	// 7a. 检查大保底状态并强制执行
	if upPoolActive {
		hasGuarantee := s.pityTracker.GetUPPoolGuarantee(user.ID, campaign.ID)
		if hasGuarantee {
			// 大保底生效：第一次命中UP目标等级时强制改为UP款式
			for i, d := range drawResults {
				if d.PrizeID != "" && d.PrizeLevel == campaign.PityConfig.UPLevel {
					drawResults[i].PrizeID = campaign.PityConfig.UPPrizeID
					drawResults[i].PrizeLevel = campaign.PityConfig.UPLevel
					drawResults[i].IsUPPoolWin = true
					s.pityTracker.SetUPPoolGuarantee(user.ID, campaign.ID, false) // 消耗大保底
					break // 一次抽奖只触发一次大保底
				}
			}
		}
	}

	// 7b. 50/50判定：非大保底状态下命中目标等级
	if upPoolActive {
		for i, d := range drawResults {
			if d.PrizeID != "" && d.PrizeLevel == campaign.PityConfig.UPLevel && !d.IsUPPoolWin {
				// 50/50 判定
				if rand.IntN(2) == 0 { // 50%概率不歪
					drawResults[i].PrizeID = campaign.PityConfig.UPPrizeID
					drawResults[i].IsUPPoolWin = true
				} else {
					// 歪了 → 设置大保底标记
					s.pityTracker.SetUPPoolGuarantee(user.ID, campaign.ID, true)
				}
			}
		}
	}

	// 8. 持久化结果
	singleResults := make([]model.SingleDrawResult, 0, len(drawResults))
	for _, d := range drawResults {
		sr := model.SingleDrawResult{
			IsWin:       d.PrizeID != "",
			IsHardPity:  d.IsHardPity,
			IsUPPoolWin: d.IsUPPoolWin, // 🆕 UP池中奖标记
		}
		if d.PrizeID != "" {
			rec, err := s.store.CreateDrawRecord(user.ID, campaign.ID, d.PrizeID, isTenPull)
			if err != nil {
				return nil, fmt.Errorf("save draw record: %w", err)
			}
			sr.RecordID = rec.ID
			sr.PrizeID = d.PrizeID
			sr.PrizeName = rec.PrizeName
			sr.PrizeLevel = d.PrizeLevel
			// 检查是否为新款式
			count, _ := s.store.GetPrizeCount(user.ID, d.PrizeID)
			sr.IsNew = (count <= 1) // 库存中只有刚加的1条或0条 -> 新款式
		} else {
			rec, err := s.store.CreateMissRecord(user.ID, campaign.ID, isTenPull)
			if err != nil {
				return nil, fmt.Errorf("save miss record: %w", err)
			}
			sr.RecordID = rec.ID
			sr.PrizeName = "未中奖"
		}
		singleResults = append(singleResults, sr)
	}

	// 8. 更新每日次数
	newRemaining, err := s.store.DeductDrawQuota(user.ID, campaign.ID, drawCount)
	if err != nil {
		return nil, err
	}

	// 9. 扣减积分 + 更新消费记录（自动更新会员等级）
	// 注：member.Points 已在第4步中扣减，这里只更新消费统计
	member.TotalSpent += pointsCost
	member.TotalDraws += int64(drawCount)
	// 自动计算会员等级
	member.Level = s.calcMemberLevel(member.TotalSpent)
	s.store.UpdateUserMember(member) // 需要在Store接口加

	// 记录积分变动日志
	s.store.LogPoints(user.ID, -pointsCost, member.Points, "draw", fmt.Sprintf("抽奖消耗: %s x%d", campaign.Name, drawCount))

	// 10. 检查集齐奖励
	reward, _ := s.store.CheckCollectionCompletion(user.ID, campaign.ID)
	var rewardInfo *model.CollectionReward
	if reward != nil {
		if err := s.store.GrantCollectionReward(user.ID, reward); err == nil {
			rewardInfo = reward
		}
	}

	// 11. 获取保底状态
	state := s.pityTracker.Get(user.ID, campaign.ID)
	ps := &model.PityStatus{
		ConsecutiveMisses: state.ConsecutiveMisses,
		PityMultiplier:    state.PityMultiplier,
	}
	if pityCfg.Enabled {
		ps.SoftPityN = pityCfg.SoftPityN
		ps.HardPityN = pityCfg.HardPityN
		if pityCfg.HardPityN > 0 {
			ps.MissesToHardPity = pityCfg.HardPityN - state.ConsecutiveMisses
			if ps.MissesToHardPity < 0 {
				ps.MissesToHardPity = 0
			}
		}
	}
	// 🆕 UP池保底状态
	if upPoolActive {
		ps.HasUPPoolGuarantee = s.pityTracker.GetUPPoolGuarantee(user.ID, campaign.ID)
	}

	return &model.BlindBoxDrawResult{
		Draws:            singleResults,
		RemainingChances: newRemaining,
		PityStatus:       ps,
		CollectionReward: rewardInfo,
	}, nil
}

// BuyCard 购买月卡/周卡/季卡
func (s *Service) BuyCard(token string, input model.BuyCardRequest) (*model.BuyCardResult, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.BuyCard(user.ID, input.CardType)
}

// GetUserCard 获取用户月卡信息
func (s *Service) GetUserCard(token string) (*model.UserCard, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	card, err := s.store.GetUserCard(user.ID)
	if err != nil {
		return nil, err
	}
	return card, nil
}

// PityStatus 查询用户在当前活动的保底状态
func (s *Service) PityStatus(token string, campaignID string) (*model.PityStatus, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	campaign, err := s.store.GetCampaign(campaignID)
	if err != nil {
		return nil, err
	}

	state := s.pityTracker.Get(user.ID, campaignID)
	pityCfg := s.buildPityConfig(campaign)

	ps := &model.PityStatus{
		ConsecutiveMisses: state.ConsecutiveMisses,
		PityMultiplier:    state.PityMultiplier,
	}
	if pityCfg.Enabled {
		ps.SoftPityN = pityCfg.SoftPityN
		ps.HardPityN = pityCfg.HardPityN
		if pityCfg.HardPityN > 0 {
			ps.MissesToHardPity = pityCfg.HardPityN - state.ConsecutiveMisses
			if ps.MissesToHardPity < 0 {
				ps.MissesToHardPity = 0
			}
		}
	}
	return ps, nil
}

// ============================================================
// 用户库存 / 收集进度
// ============================================================

// UserInventory 返回用户的库存列表
func (s *Service) UserInventory(token string) ([]model.UserInventory, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetUserInventory(user.ID)
}

// SeriesProgress 返回用户在某个系列的收集进度
func (s *Service) SeriesProgress(token string, campaignID string) (*model.SeriesProgress, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	campaign, err := s.store.GetCampaign(campaignID)
	if err != nil {
		return nil, err
	}
	return s.store.GetSeriesProgress(user.ID, campaignID, campaign.Name)
}

// ============================================================
// 交换市场
// ============================================================

func (s *Service) ExchangeOffers() []model.ExchangeOffer {
	return s.store.ExchangeOffers()
}

func (s *Service) CreateExchangeOffer(token string, input model.ExchangeOfferMutation) (model.ExchangeOffer, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return model.ExchangeOffer{}, err
	}
	// 验证用户是否拥有 HavePrize
	hasIt, err := s.store.UserHasPrize(user.ID, input.HavePrizeID)
	if err != nil || !hasIt {
		return model.ExchangeOffer{}, fmt.Errorf("you don't own this prize")
	}
	return s.store.CreateExchangeOffer(user.ID, input)
}

func (s *Service) CancelExchangeOffer(token string, offerID string) error {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return err
	}
	return s.store.CancelExchangeOffer(user.ID, offerID)
}

func (s *Service) AcceptExchangeOffer(token string, offerID string) (model.ExchangeOffer, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return model.ExchangeOffer{}, err
	}
	return s.store.AcceptExchangeOffer(user.ID, offerID)
}

// ============================================================
// 积分/会员
// ============================================================

func (s *Service) UserMember(token string) (*model.UserMember, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetUserMember(user.ID)
}

func (s *Service) PointsLog(token string) ([]model.UserPointsLog, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetPointsLog(user.ID)
}

func (s *Service) RedeemPrize(token string, input model.RedeemRequest) (*model.RedeemResult, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.RedeemPrize(user.ID, input)
}

// BlendPrizes 合成：消耗多个重复款式，合成更高级款式
func (s *Service) BlendPrizes(token string, input model.BlendRequest) (*model.BlendResult, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.BlendPrizes(user.ID, input.SourcePrizeID, input.CampaignID)
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
	// TODO: implement with actual DB aggregation
	return s.store.GetDrawStatistics(token, campaignID)
}

// AdminUpdatePityConfig 管理员更新活动的保底配置
func (s *Service) AdminUpdatePityConfig(token string, campaignID string, cfg model.PityConfig) (*model.Campaign, error) {
	// 先验证管理员身份
	if _, err := s.store.AdminOverview(token); err != nil {
		return nil, err
	}
	// 获取当前活动
	campaign, err := s.store.GetCampaign(campaignID)
	if err != nil {
		return nil, err
	}
	campaign.PityConfig = cfg
	// 转成 CampaignMutation 更新
	mutation := model.CampaignMutation{
		Name: campaign.Name, Slug: campaign.Slug, Status: campaign.Status,
		StartsAt: campaign.StartsAt, EndsAt: campaign.EndsAt,
		DailyDrawLimit: campaign.DailyDrawLimit, MissWeight: campaign.MissWeight,
		BannerImageURL: campaign.BannerImageURL, CampaignSummary: campaign.CampaignSummary,
		PityConfig: cfg,
	}
	return s.store.UpdateCampaign(token, campaignID, mutation)
}

// 🆕 AdminGetCampaign 管理员获取活动详情
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
		ID:        d.RecordID,
		CampaignID: campaignID,
		PrizeName: prizeName,
		Result:    map[bool]string{true: "win", false: "miss"}[d.IsWin],
		DrawnAt:   time.Now().UTC(),
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

// 🆕 isUPPoolActive 检查UP池是否当前生效
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

// 🆕 getUPPrizeInfo 获取UP池目标奖品信息（用于前端展示）
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

// 🆕 IsUPPoolActive 外部可调用的UP池检查
func (s *Service) IsUPPoolActive(campaignID string) (bool, error) {
	campaign, err := s.store.GetCampaign(campaignID)
	if err != nil {
		return false, err
	}
	return isUPPoolActive(campaign), nil
}

// 🆕 UPPoolInfo 获取UP池信息
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
		"active":         active,
		"prize":          upInfo,
		"has_guarantee":  hasGuarantee,
		"up_prize_id":    campaign.PityConfig.UPPrizeID,
		"up_multiplier":  campaign.PityConfig.UPMultiplier,
		"up_level":       campaign.PityConfig.UPLevel,
		"up_start_at":    campaign.PityConfig.UPStartAt,
		"up_end_at":      campaign.PityConfig.UPEndAt,
		"consecutive_misses": s.pityTracker.Get(user.ID, campaignID).ConsecutiveMisses,
	}, nil
}

// ============================================================
// 会员等级 & 用户积分系统
// ============================================================

// calcMemberLevel 根据累计消费积分计算会员等级
// Lv1青铜: 注册即得
// Lv2白银: ≥500分
// Lv3黄金: ≥2000分
// Lv4铂金: ≥5000分
// Lv5钻石: ≥15000分
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

// DailyCheckIn 每日签到
// 验证token，基础+5分，连续签到≥7天额外+20分
func (s *Service) DailyCheckIn(token string) (*model.CheckInResult, error) {
	user, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	result, err := s.store.DailyCheckIn(user.ID, 5)
	if err != nil {
		return nil, err
	}
	if result.StreakDays >= 7 {
		bonusResult, err := s.store.DailyCheckIn(user.ID, 20)
		if err != nil {
			return nil, err
		}
		bonusResult.PointsAwarded = result.PointsAwarded + 20
		bonusResult.IsBonus = true
		member, _ := s.store.GetUserMember(user.ID)
		s.store.LogPoints(user.ID, bonusResult.PointsAwarded, member.Points, "daily", "每日签到")
		return bonusResult, nil
	}
	member, _ := s.store.GetUserMember(user.ID)
	s.store.LogPoints(user.ID, result.PointsAwarded, member.Points, "daily", "每日签到")
	return result, nil
}

// ShareReward 分享奖励
// 每次+2分，每日上限10次
func (s *Service) ShareReward(token string) (*model.ShareRewardResult, error) {
	user, err := s.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	count, err := s.store.GetShareDailyCount(user.ID)
	if err != nil {
		return nil, err
	}
	if count >= 10 {
		return nil, store.ErrShareLimitReached
	}
	result, err := s.store.ShareReward(user.ID, 2)
	if err != nil {
		return nil, err
	}
	member, _ := s.store.GetUserMember(user.ID)
	s.store.LogPoints(user.ID, 2, member.Points, "share", "分享奖励")
	return result, nil
}

// GetLeaderboard 获取收集排行榜
func (s *Service) GetLeaderboard(limit int) ([]model.LeaderboardEntry, error) {
	return s.store.GetLeaderboard(limit)
}

// GetCampaignHint 获取盲盒摇盒提示文案
func (s *Service) GetCampaignHint(campaignID string) *model.HintMessage {
	return s.store.GetCampaignHint(campaignID)
}

// 🆕 ---- 月卡系统 ----

// MonthCardStatus 查询用户月卡状态
func (s *Service) MonthCardStatus(token string) (*model.MonthCardStatus, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	card, _ := s.store.GetMonthCard(user.ID)
	if card == nil {
		return &model.MonthCardStatus{HasCard: false, FreeDraws: 0, DrawDiscount: 1.0}, nil
	}
	daysLeft := int(time.Until(card.ExpiresAt).Hours() / 24)
	if daysLeft < 0 {
		daysLeft = 0
	}
	used, _ := s.store.GetTodayFreeDrawUsed(user.ID)
	return &model.MonthCardStatus{
		HasCard: true, CardType: string(card.CardType),
		FreeDraws: card.FreeDraws, DrawDiscount: card.DrawDiscount,
		ExpiresAt: card.ExpiresAt, DaysLeft: daysLeft, TodayFreeUsed: used,
	}, nil
}

// BuyMonthCard 购买月卡
func (s *Service) BuyMonthCard(token string, cardType model.MonthCardType) (*model.MonthCardPurchaseResult, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	var price int64
	switch cardType {
	case model.MonthCardWeekly:
		price = 990
	case model.MonthCardMonthly:
		price = 2800
	case model.MonthCardSeason:
		price = 6800
	default:
		return nil, fmt.Errorf("invalid card type: %s", cardType)
	}
	member, err := s.store.GetUserMember(user.ID)
	if err != nil {
		return nil, err
	}
	if member.Points < price {
		return nil, store.ErrInsufficientPoints
	}
	card, err := s.store.BuyMonthCard(user.ID, cardType, price)
	if err != nil {
		return nil, err
	}
	member.Points -= price
	member.TotalSpent += price
	s.store.UpdateUserMember(member)
	s.store.LogPoints(user.ID, -price, member.Points, "month_card", fmt.Sprintf("购买%s", cardType))
	return &model.MonthCardPurchaseResult{Card: *card, NewPoints: member.Points}, nil
}

// 🆕 ---- 战令系统 ----

// BattlePassInfo 获取战令完整信息
func (s *Service) BattlePassInfo(token string) (*model.BattlePassInfo, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	season, err := s.store.GetActiveSeason()
	if err != nil {
		return nil, err
	}
	userPass, _ := s.store.GetUserBattlePass(user.ID, season.ID)
	tasks, _ := s.store.GetBattlePassTasks(season.ID)
	taskProgress, _ := s.store.GetBattlePassTaskProgress(user.ID, season.ID)
	rewards, _ := s.store.GetBattlePassRewards(season.ID)
	levelProgress := 0
	if userPass != nil && season.XPPerLevel > 0 {
		levelProgress = userPass.XP
	}
	return &model.BattlePassInfo{
		Season: season, UserPass: userPass,
		Tasks: tasks, TaskProgress: taskProgress,
		Rewards: rewards, LevelProgress: levelProgress,
	}, nil
}

// BuyBattlePass 购买付费战令
func (s *Service) BuyBattlePass(token string) (*model.BattlePass, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	season, err := s.store.GetActiveSeason()
	if err != nil {
		return nil, err
	}
	price := int64(4800) // 48元
	member, err := s.store.GetUserMember(user.ID)
	if err != nil {
		return nil, err
	}
	if member.Points < price {
		return nil, store.ErrInsufficientPoints
	}
	bp, err := s.store.BuyBattlePass(user.ID, season.ID, price)
	if err != nil {
		return nil, err
	}
	member.Points -= price
	member.TotalSpent += price
	s.store.UpdateUserMember(member)
	s.store.LogPoints(user.ID, -price, member.Points, "battle_pass", "购买战令")
	return bp, nil
}

// ClaimBattlePassReward 领取战令等级奖励
func (s *Service) ClaimBattlePassReward(token string, level int) (bool, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return false, err
	}
	season, err := s.store.GetActiveSeason()
	if err != nil {
		return false, err
	}
	return s.store.ClaimBattlePassReward(user.ID, season.ID, level)
}

// ============================================================
// 🆕 商店 + 道具 + 首充礼包 Service
// ============================================================

// ShopItems 获取商店商品列表
func (s *Service) ShopItems(token string) ([]model.ShopItem, error) {
	if _, err := s.store.UserFromToken(token); err != nil {
		return nil, err
	}
	return s.store.GetShopItems(), nil
}

// BuyShopItem 购买商店商品
func (s *Service) BuyShopItem(token string, input model.BuyShopItemRequest) (*model.BuyShopItemResult, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	result, err := s.store.BuyShopItem(user.ID, input.ShopItemID, input.Quantity)
	if err != nil {
		return nil, err
	}
	// 记录积分变动
	member, _ := s.store.GetUserMember(user.ID)
	s.store.LogPoints(user.ID, -result.PointsCost, member.Points, "shop", fmt.Sprintf("购买 %s x%d", result.ItemName, result.Quantity))
	return result, nil
}

// UserItems 获取用户所有道具
func (s *Service) UserItems(token string) ([]model.UserItem, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetUserItems(user.ID)
}

// UseItem 使用道具
func (s *Service) UseItem(token string, input model.UseItemRequest) (map[string]any, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}

	switch input.ItemType {
	case model.ItemHintCard:
		// 提示卡：排除当前池1个不想要的款式
		ok, err := s.store.UseUserItem(user.ID, model.ItemHintCard, 1)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("insufficient hint cards")
		}
		return map[string]any{
			"success": true, "message": "使用提示卡成功，排除了一个款式",
			"excluded_prize_id": input.PrizeID,
		}, nil

	case model.ItemSeeThrough:
		// 透卡：预览下一抽（仅普通款）
		ok, err := s.store.UseUserItem(user.ID, model.ItemSeeThrough, 1)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("insufficient see-through cards")
		}
		// 随机返回一个普通款作为预览
		if input.CampaignID != "" {
			prizes := s.store.PrizeList(input.CampaignID)
			for _, p := range prizes {
				if p.Level == model.PrizeLevelCommon && p.Status == "active" {
					return map[string]any{
						"success": true, "message": "预览成功",
						"preview_prize": map[string]any{
							"id": p.ID, "name": p.Name, "level": p.Level,
						},
					}, nil
				}
			}
		}
		return map[string]any{"success": true, "message": "预览完成，仅显示普通款"}, nil

	case model.ItemTenDrawTicket:
		// 十连券：直接在抽奖中使用
		ok, err := s.store.UseUserItem(user.ID, model.ItemTenDrawTicket, 1)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("insufficient ten-draw tickets")
		}
		return map[string]any{
			"success": true, "message": "使用十连券成功，可进行一次免费十连抽",
		}, nil

	default:
		return nil, fmt.Errorf("unsupported item type: %s", input.ItemType)
	}
}

// 🆕 首充礼包

// FirstRechargeStatus 获取首充状态（可领取的礼包列表）
func (s *Service) FirstRechargeStatus(token string) (*model.UserFirstRecharge, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetFirstRechargeStatus(user.ID)
}

// FirstRechargePacks 获取所有首充礼包配置
func (s *Service) FirstRechargePacks() map[string]model.FirstRechargePack {
	return model.FirstRechargePacks
}

// ClaimFirstRecharge 领取首充礼包
func (s *Service) ClaimFirstRecharge(token string, input model.ClaimFirstRechargeRequest) (*model.ClaimFirstRechargeResult, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.ClaimFirstRecharge(user.ID, input.PackID)
}

// ============================================================
// 🆕 v1.5 社交裂变 Service
// ============================================================

// ---- 分享卡片 ----

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
		// 免费抽：给用户添加免费抽券
		s.store.AddUserItem(user.ID, model.ItemFreeDraw, 1)
		return &model.AssistClaimResult{
			AssistType: assistType, RewardType: "free_draw",
			Description: "助力完成！获得1次免费抽奖机会 🎉",
		}, nil
	case model.AssistPityReduce:
		// 保底缩短：-10次
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
	// Only return reward if team completed

	return &model.TeamInfo{
		Team:           team,
		Members:        members,
		CaptainName:    captainName,
		RemainingHours: remainingHours,
		Reward:         reward,
	}, nil
}

// AddTeamDrawHook 抽奖后调用，累计队伍开盒次数（从盲盒抽奖Service调用）
func (s *Service) AddTeamDrawHook(userID string) {
	team, err := s.store.GetUserActiveTeam(userID)
	if err != nil || team == nil {
		return
	}
	total, err := s.store.AddTeamDraw(userID, team.ID)
	if err != nil {
		return
	}
	_ = total
	// 如果达到目标，自动完成
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
