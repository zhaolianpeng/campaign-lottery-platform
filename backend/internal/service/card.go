package service

import (
	"fmt"
	"time"

	"campaign-lottery-platform/backend/internal/model"
	"campaign-lottery-platform/backend/internal/store"
)

// ============================================================
// 月卡/周卡/季卡
// ============================================================

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

// ============================================================
// 月卡系统
// ============================================================

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

// ============================================================
// 战令系统
// ============================================================

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
