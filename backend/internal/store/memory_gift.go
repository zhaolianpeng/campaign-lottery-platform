package store

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"campaign-lottery-platform/backend/internal/model"
)

// ============================================================
// 🆕 礼物赠送系统
// ============================================================

// CreateGift 创建礼物（赠送道具）
func (s *MemoryStore) CreateGift(giverID, receiverID, prizeID, campaignID string) (*model.GiftRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 查找奖品信息
	prizes, ok := s.prizes[campaignID]
	if !ok {
		return nil, fmt.Errorf("campaign not found: %s", campaignID)
	}
	var prizeName, prizeLevel string
	for _, p := range prizes {
		if p.ID == prizeID {
			prizeName = p.Name
			prizeLevel = p.Level
			break
		}
	}
	if prizeName == "" {
		return nil, fmt.Errorf("prize not found: %s", prizeID)
	}

	// 查找赠送者的库存
	invIdx := -1
	for i, inv := range s.inventory {
		if inv.UserID == giverID && inv.PrizeID == prizeID && inv.CampaignID == campaignID {
			invIdx = i
			break
		}
	}
	if invIdx == -1 {
		return nil, fmt.Errorf("giver does not own this prize in inventory")
	}

	// 稀有/隐藏/限定款收取包装费
	feePoints := int64(0)
	if prizeLevel == model.PrizeLevelRare || prizeLevel == model.PrizeLevelSecret || prizeLevel == model.PrizeLevelLimited {
		feePoints = 500
		member, ok := s.members[giverID]
		if !ok {
			member = model.UserMember{UserID: giverID, Level: model.MemberNormal, Points: 0}
		}
		if member.Points < feePoints {
			return nil, ErrInsufficientPoints
		}
		member.Points -= feePoints
		s.members[giverID] = member

		s.pointsLog = append(s.pointsLog, model.UserPointsLog{
			ID:        int64(len(s.pointsLog) + 1),
			UserID:    giverID,
			Points:    -feePoints,
			Balance:   member.Points,
			Reason:    "gift_fee",
			Remark:    fmt.Sprintf("赠送%s包装费", prizeName),
			CreatedAt: time.Now().UTC(),
		})
	}

	// 从赠送者库存中删除该道具
	s.inventory = append(s.inventory[:invIdx], s.inventory[invIdx+1:]...)

	// 创建礼物记录
	s.giftIDSeq++
	now := time.Now().UTC()
	gift := model.GiftRecord{
		ID:         "gift_" + strconv.Itoa(s.giftIDSeq),
		GiverID:    giverID,
		ReceiverID: receiverID,
		PrizeID:    prizeID,
		PrizeName:  prizeName,
		PrizeLevel: prizeLevel,
		FeePoints:  feePoints,
		Status:     "sent",
		CreatedAt:  now,
		ExpiresAt:  now.Add(24 * time.Hour),
	}
	s.giftRecords = append(s.giftRecords, gift)
	s.giftPrizeInfo[gift.ID] = prizeName + ":" + prizeLevel

	return &gift, nil
}

// GetGift 查询礼物
func (s *MemoryStore) GetGift(giftID string) (*model.GiftRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.giftRecords {
		if s.giftRecords[i].ID == giftID {
			return &s.giftRecords[i], nil
		}
	}
	return nil, fmt.Errorf("gift not found: %s", giftID)
}

// ReceiveGift 接收礼物（将道具添加到接收者库存）
func (s *MemoryStore) ReceiveGift(giftID string) (*model.ReceiveGiftResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i := range s.giftRecords {
		if s.giftRecords[i].ID == giftID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil, fmt.Errorf("gift not found: %s", giftID)
	}

	gift := &s.giftRecords[idx]
	if gift.Status != "sent" {
		return nil, fmt.Errorf("gift status is %s, cannot receive", gift.Status)
	}

	now := time.Now().UTC()
	gift.Status = "received"
	gift.ReceivedAt = &now

	newItem := model.UserInventory{
		ID:         "inv_" + randomSuffix(12),
		UserID:     gift.ReceiverID,
		PrizeID:    gift.PrizeID,
		PrizeName:  gift.PrizeName,
		PrizeLevel: gift.PrizeLevel,
		CampaignID: "",
		Source:     "gift",
		CreatedAt:  now,
	}
	s.inventory = append(s.inventory, newItem)

	info := s.giftPrizeInfo[giftID]
	parts := strings.SplitN(info, ":", 2)
	result := &model.ReceiveGiftResult{
		GiftID:    giftID,
		NewItemID: newItem.ID,
	}
	if len(parts) == 2 {
		result.PrizeName = parts[0]
		result.PrizeLevel = parts[1]
	} else {
		result.PrizeName = gift.PrizeName
		result.PrizeLevel = gift.PrizeLevel
	}

	return result, nil
}

// GetUserGifts 获取用户收到的礼物列表
func (s *MemoryStore) GetUserGifts(userID string) ([]model.GiftRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.GiftRecord, 0)
	for _, g := range s.giftRecords {
		if g.ReceiverID == userID && g.Status == "sent" {
			result = append(result, g)
		}
	}
	return result, nil
}

// GetUserSentGifts 获取用户送出的礼物列表
func (s *MemoryStore) GetUserSentGifts(userID string) ([]model.GiftRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.GiftRecord, 0)
	for _, g := range s.giftRecords {
		if g.GiverID == userID {
			result = append(result, g)
		}
	}
	return result, nil
}

// ExpireGift 过期礼物（将道具退回赠送者库存）
func (s *MemoryStore) ExpireGift(giftID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i := range s.giftRecords {
		if s.giftRecords[i].ID == giftID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("gift not found: %s", giftID)
	}

	gift := &s.giftRecords[idx]
	if gift.Status != "sent" {
		return fmt.Errorf("gift status is %s, cannot expire", gift.Status)
	}

	gift.Status = "expired"

	s.inventory = append(s.inventory, model.UserInventory{
		ID:         "inv_" + randomSuffix(12),
		UserID:     gift.GiverID,
		PrizeID:    gift.PrizeID,
		PrizeName:  gift.PrizeName,
		PrizeLevel: gift.PrizeLevel,
		CampaignID: "",
		Source:     "gift_return",
		CreatedAt:  time.Now().UTC(),
	})

	return nil
}

// ============================================================
// 🆕 分享卡片系统
// ============================================================

// CreateShareCard 创建分享卡片
func (s *MemoryStore) CreateShareCard(userID, cardType, title, description, prizeName, prizeLevel, inviteLink string) (*model.ShareCard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.shareCardIDSeq++
	now := time.Now().UTC()

	card := model.ShareCard{
		ID:          "card_" + strconv.Itoa(s.shareCardIDSeq),
		UserID:      userID,
		CardType:    cardType,
		Title:       title,
		Description: description,
		PrizeName:   prizeName,
		PrizeLevel:  prizeLevel,
		InviteLink:  inviteLink,
		CreatedAt:   now,
	}
	s.shareCards[userID] = append(s.shareCards[userID], card)

	return &card, nil
}

// GetShareCards 获取用户的分享卡片列表
func (s *MemoryStore) GetShareCards(userID string) ([]model.ShareCard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cards := s.shareCards[userID]
	if cards == nil {
		return []model.ShareCard{}, nil
	}
	return cards, nil
}
