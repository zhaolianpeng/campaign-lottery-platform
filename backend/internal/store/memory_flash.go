package store

import (
	"fmt"
	"strconv"
	"time"

	"campaign-lottery-platform/backend/internal/model"
)

// ============================================================
// 🆕 v1.6 预约抢购 MemoryStore 实现
// ============================================================

// GetFlashSales 获取抢购活动列表
func (s *MemoryStore) GetFlashSales() []model.FlashSale {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.FlashSale, len(s.flashSales))
	copy(result, s.flashSales)
	return result
}

// GetFlashSale 获取抢购活动详情
func (s *MemoryStore) GetFlashSale(flashID string) (*model.FlashSale, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.flashSales {
		if s.flashSales[i].ID == flashID {
			return &s.flashSales[i], nil
		}
	}
	return nil, fmt.Errorf("flash sale not found: %s", flashID)
}

// SubscribeFlash 预约抢购活动
func (s *MemoryStore) SubscribeFlash(userID, flashID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	for _, f := range s.flashSales {
		if f.ID == flashID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("flash sale not found: %s", flashID)
	}

	for _, sub := range s.flashSubscriptions {
		if sub.UserID == userID && sub.FlashID == flashID {
			return nil
		}
	}

	s.flashSubscriptions = append(s.flashSubscriptions, model.FlashSubscription{
		UserID:    userID,
		FlashID:   flashID,
		CreatedAt: time.Now().UTC(),
	})
	return nil
}

// UnsubscribeFlash 取消预约
func (s *MemoryStore) UnsubscribeFlash(userID, flashID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, sub := range s.flashSubscriptions {
		if sub.UserID == userID && sub.FlashID == flashID {
			s.flashSubscriptions = append(s.flashSubscriptions[:i], s.flashSubscriptions[i+1:]...)
			return nil
		}
	}
	return nil
}

// IsFlashSubscribed 检查用户是否已预约
func (s *MemoryStore) IsFlashSubscribed(userID, flashID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, sub := range s.flashSubscriptions {
		if sub.UserID == userID && sub.FlashID == flashID {
			return true, nil
		}
	}
	return false, nil
}

// PurchaseFlash 执行抢购（扣积分减库存）
func (s *MemoryStore) PurchaseFlash(userID, flashID string) (*model.FlashPurchaseResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var flash *model.FlashSale
	for i := range s.flashSales {
		if s.flashSales[i].ID == flashID {
			flash = &s.flashSales[i]
			break
		}
	}
	if flash == nil {
		return &model.FlashPurchaseResult{
			FlashID: flashID, Success: false, Message: "抢购活动不存在",
		}, fmt.Errorf("flash sale not found: %s", flashID)
	}

	if flash.RemainingStock <= 0 {
		return &model.FlashPurchaseResult{
			FlashID: flashID, FlashName: flash.Name, Success: false, Message: "库存不足",
		}, fmt.Errorf("flash sale out of stock: %s", flashID)
	}

	member, ok := s.members[userID]
	if !ok {
		member = model.UserMember{
			UserID:    userID,
			Level:     model.MemberNormal,
			Points:    0,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
	}

	if member.Level < model.MemberLevel(flash.MinVipLevel) && flash.MinVipLevel != "" {
		return &model.FlashPurchaseResult{
			FlashID: flashID, FlashName: flash.Name, Success: false, Message: "会员等级不足",
		}, fmt.Errorf("insufficient vip level: need %s, have %s", flash.MinVipLevel, member.Level)
	}

	if member.TotalDraws < flash.MinTotalDraws {
		return &model.FlashPurchaseResult{
			FlashID: flashID, FlashName: flash.Name, Success: false, Message: "抽奖次数不足",
		}, fmt.Errorf("insufficient total draws: need %d, have %d", flash.MinTotalDraws, member.TotalDraws)
	}

	if member.Points < flash.PricePoints {
		return &model.FlashPurchaseResult{
			FlashID: flashID, FlashName: flash.Name, Success: false, Message: "积分不足",
		}, fmt.Errorf("insufficient points: need %d, have %d", flash.PricePoints, member.Points)
	}

	now := time.Now().UTC()
	member.Points -= flash.PricePoints
	member.UpdatedAt = now
	s.members[userID] = member

	flash.RemainingStock--

	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID:        int64(len(s.pointsLog) + 1),
		UserID:    userID,
		Points:    -flash.PricePoints,
		Balance:   member.Points,
		Reason:    "flash_purchase",
		Remark:    "抢购: " + flash.Name,
		CreatedAt: now,
	})

	s.inventory = append(s.inventory, model.UserInventory{
		ID:         s.GenerateID(),
		UserID:     userID,
		PrizeID:    "",
		PrizeName:  flash.Name,
		PrizeLevel: "",
		CampaignID: flash.CampaignID,
		Source:     "flash",
		CreatedAt:  now,
	})

	return &model.FlashPurchaseResult{
		FlashID:   flashID,
		FlashName: flash.Name,
		Success:   true,
		Message:   "抢购成功",
	}, nil
}

// GetUserFlashSubscriptions 获取用户预约列表
func (s *MemoryStore) GetUserFlashSubscriptions(userID string) ([]model.FlashSubscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var subs []model.FlashSubscription
	for _, sub := range s.flashSubscriptions {
		if sub.UserID == userID {
			subs = append(subs, sub)
		}
	}
	return subs, nil
}

// CreateFlashSale 创建抢购活动（管理端）
func (s *MemoryStore) CreateFlashSale(input model.FlashSale) (*model.FlashSale, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.flashIDSeq++
	now := time.Now().UTC()
	flash := &model.FlashSale{
		ID:             "flash_" + strconv.Itoa(s.flashIDSeq),
		CampaignID:     input.CampaignID,
		Name:           input.Name,
		Description:    input.Description,
		PricePoints:    input.PricePoints,
		TotalStock:     input.TotalStock,
		RemainingStock: input.TotalStock,
		MinVipLevel:    input.MinVipLevel,
		MinTotalDraws:  input.MinTotalDraws,
		StartAt:        input.StartAt,
		EndAt:          input.EndAt,
		Status:         "upcoming",
		CreatedAt:      now,
	}
	s.flashSales = append(s.flashSales, *flash)
	return flash, nil
}

// UpdateFlashSaleStatus 更新抢购状态
func (s *MemoryStore) UpdateFlashSaleStatus(flashID, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.flashSales {
		if s.flashSales[i].ID == flashID {
			s.flashSales[i].Status = status
			return nil
		}
	}
	return fmt.Errorf("flash sale not found: %s", flashID)
}
