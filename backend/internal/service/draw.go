package service

import (
	"fmt"
	"math/rand/v2"
	"time"

	"campaign-lottery-platform/backend/internal/model"
	"campaign-lottery-platform/backend/internal/probability"
	"campaign-lottery-platform/backend/internal/store"
)

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
			cardCfg := model.CardConfigs[card.CardType]
			unitPrice = int(float64(unitPrice) * cardCfg.DiscountRate)
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

	// 6. 执行抽奖
	var drawResults []probability.DrawResult
	if isTenPull {
		drawResults = engine.DrawMultiple(drawCount, pityCfg, s.pityTracker, user.ID, campaign.ID)
	} else {
		r := engine.Draw(pityCfg, s.pityTracker, user.ID, campaign.ID)
		drawResults = []probability.DrawResult{r}
	}

	// 🆕 UP池处理（50/50大小保底）
	upPoolActive := isUPPoolActive(campaign)
	upPrizeInfo := getUPPrizeInfo(upPoolActive, prizes)
	_ = upPrizeInfo

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
			IsUPPoolWin: d.IsUPPoolWin,
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
			sr.IsNew = (count <= 1)
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
	member.TotalSpent += pointsCost
	member.TotalDraws += int64(drawCount)
	// 自动计算会员等级
	member.Level = s.calcMemberLevel(member.TotalSpent)
	s.store.UpdateUserMember(member)

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
	// UP池保底状态
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
