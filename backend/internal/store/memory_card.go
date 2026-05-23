package store

import (
	"fmt"
	"time"

	"campaign-lottery-platform/backend/internal/model"
)

// ============================================================
// 月卡/付费卡系统 MemoryStore 实现
// ============================================================

func (s *MemoryStore) GetUserCard(userID string) (*model.UserCard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	card, ok := s.userCards[userID]
	if !ok || card.ExpiresAt.Before(time.Now().UTC()) {
		return nil, nil
	}
	// 每天重置免费次数
	today := time.Now().UTC().Format("2006-01-02")
	if card.FreeDate != today {
		card.DailyFreeUsed = 0
		card.FreeDate = today
		s.userCards[userID] = card // 存回去
	}
	return &card, nil
}

func (s *MemoryStore) BuyCard(userID string, cardType model.CardType) (*model.BuyCardResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, ok := model.CardConfigs[cardType]
	if !ok {
		return nil, fmt.Errorf("unknown card type: %s", cardType)
	}

	member, ok := s.members[userID]
	if !ok {
		member = model.UserMember{UserID: userID, Level: model.MemberNormal, Points: 0}
	}
	cost := int64(cfg.Price)
	if member.Points < cost {
		return nil, ErrInsufficientPoints
	}

	member.Points -= cost
	s.members[userID] = member

	now := time.Now().UTC()
	card := model.UserCard{
		UserID:    userID,
		CardType:  cardType,
		StartedAt: now,
		ExpiresAt: now.AddDate(0, 0, cfg.DurationDays),
		FreeDate:  now.Format("2006-01-02"),
	}
	s.userCards[userID] = card

	// 记录积分
	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID: int64(len(s.pointsLog) + 1), UserID: userID, Points: -cost,
		Balance: member.Points, Reason: "buy_card", Remark: "购买" + cfg.Description, CreatedAt: now,
	})

	return &model.BuyCardResult{
		CardType: cardType, ExpiresAt: card.ExpiresAt.Format("2006-01-02"),
		Price: cfg.Price, Points: member.Points,
	}, nil
}

func (s *MemoryStore) ConsumeFreeDraw(userID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	card, ok := s.userCards[userID]
	if !ok || card.ExpiresAt.Before(time.Now().UTC()) {
		return false, nil
	}
	today := time.Now().UTC().Format("2006-01-02")
	if card.FreeDate != today {
		card.DailyFreeUsed = 0
		card.FreeDate = today
	}
	cfg := model.CardConfigs[card.CardType]
	if card.DailyFreeUsed >= cfg.FreeDrawsDaily {
		return false, nil
	}
	card.DailyFreeUsed++
	s.userCards[userID] = card
	return true, nil
}

func (s *MemoryStore) GetFreeDrawRemaining(userID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	card, ok := s.userCards[userID]
	if !ok || card.ExpiresAt.Before(time.Now().UTC()) {
		return 0, nil
	}
	cfg := model.CardConfigs[card.CardType]
	today := time.Now().UTC().Format("2006-01-02")
	used := card.DailyFreeUsed
	if card.FreeDate != today {
		used = 0
	}
	remaining := cfg.FreeDrawsDaily - used
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

// ─────────────────────────────────────────────────────────────
// 🆕 月卡系统 MemoryStore 实现
// ─────────────────────────────────────────────────────────────

func (s *MemoryStore) GetMonthCard(userID string) (*model.MonthCard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if card, ok := s.monthCards[userID]; ok && card.ExpiresAt.After(time.Now().UTC()) {
		c := *card
		return &c, nil
	}
	return nil, nil
}

func (s *MemoryStore) BuyMonthCard(userID string, cardType model.MonthCardType, pointsCost int64) (*model.MonthCard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	var freeDraws int
	var discount float64
	var price int64
	var duration time.Duration
	switch cardType {
	case model.MonthCardWeekly:
		freeDraws, discount, price, duration = 1, 0.9, 990, 7*24*time.Hour
	case model.MonthCardMonthly:
		freeDraws, discount, price, duration = 2, 0.8, 2800, 30*24*time.Hour
	case model.MonthCardSeason:
		freeDraws, discount, price, duration = 3, 0.75, 6800, 90*24*time.Hour
	default:
		return nil, fmt.Errorf("invalid card type: %s", cardType)
	}
	s.monthCards[userID] = &model.MonthCard{
		ID: "mcard_" + randomSuffix(12), UserID: userID, CardType: cardType,
		Price: price, FreeDraws: freeDraws, DrawDiscount: discount,
		StartedAt: now, ExpiresAt: now.Add(duration), CreatedAt: now,
	}
	c := *s.monthCards[userID]
	return &c, nil
}

func (s *MemoryStore) UseFreeDraw(userID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := "free:" + userID + ":" + time.Now().UTC().Format("2006-01-02")
	s.freeDrawUsed[key]++
	return s.freeDrawUsed[key], nil
}

func (s *MemoryStore) GetTodayFreeDrawUsed(userID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := "free:" + userID + ":" + time.Now().UTC().Format("2006-01-02")
	return s.freeDrawUsed[key], nil
}
