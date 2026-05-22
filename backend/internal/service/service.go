package service

import (
	"fmt"
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
			"target_prize":         campaign.TargetPrizeID,
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

	// 4. 构建概率引擎
	prizes := s.store.PrizeList(campaign.ID)
	pw := make([]probability.PrizeWeight, 0, len(prizes))
	secretID := ""
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
		}
	}

	pityCfg := s.buildPityConfig(campaign)
	engineCfg := probability.EngineConfig{
		TargetPrizeID: secretID,
		MissWeight:    float64(campaign.MissWeight),
		Pity:          pityCfg,
	}
	engine := probability.NewEngine(engineCfg, pw)

	// 5. 执行抽奖
	var drawResults []probability.DrawResult
	if isTenPull {
		drawResults = engine.DrawMultiple(drawCount, pityCfg, s.pityTracker, user.ID, campaign.ID)
	} else {
		r := engine.Draw(pityCfg, s.pityTracker, user.ID, campaign.ID)
		drawResults = []probability.DrawResult{r}
	}

	// 6. 持久化结果
	singleResults := make([]model.SingleDrawResult, 0, len(drawResults))
	for _, d := range drawResults {
		sr := model.SingleDrawResult{
			IsWin:    d.PrizeID != "",
			IsHardPity: d.IsHardPity,
		}
		if d.PrizeID != "" {
			// 通过 store 完成库存扣减和记录写入
			rec, err := s.store.CreateDrawRecord(user.ID, campaign.ID, d.PrizeID, isTenPull)
			if err != nil {
				return nil, fmt.Errorf("save draw record: %w", err)
			}
			sr.RecordID = rec.ID
			sr.PrizeID = d.PrizeID
			sr.PrizeName = rec.PrizeName
			sr.PrizeLevel = d.PrizeLevel
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

	// 7. 更新每日次数
	usedCount, err := s.store.DeductDrawQuota(user.ID, campaign.ID, drawCount)
	if err != nil {
		return nil, err
	}
	newRemaining := campaign.DailyDrawLimit - usedCount
	if newRemaining < 0 {
		newRemaining = 0
	}

	// 8. 获取保底状态
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

	return &model.BlindBoxDrawResult{
		Draws:            singleResults,
		RemainingChances: newRemaining,
		PityStatus:       ps,
	}, nil
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
	return probability.PityConfig{
		Enabled:    campaign.PityEnabled,
		SoftPityN:  campaign.SoftPityN,
		PityFactor: campaign.PityFactor,
		HardPityN:  campaign.HardPityN,
	}
}
