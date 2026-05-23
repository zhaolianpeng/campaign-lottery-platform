package store

import (
	"fmt"
	"time"

	"campaign-lottery-platform/backend/internal/model"
)

// ============================================================
// 🆕 商店 + 道具 + 首充 MemoryStore 实现
// ============================================================

func (s *MemoryStore) GetShopItems() []model.ShopItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.ShopItem, 0, len(s.shopItems))
	for _, item := range s.shopItems {
		if !item.IsActive {
			continue
		}
		if item.ExpiresAt != nil && time.Now().UTC().After(*item.ExpiresAt) {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MemoryStore) BuyShopItem(userID string, itemID string, quantity int) (*model.BuyShopItemResult, error) {
	if quantity <= 0 {
		quantity = 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// 查找商品
	var item *model.ShopItem
	for i := range s.shopItems {
		if s.shopItems[i].ID == itemID {
			item = &s.shopItems[i]
			break
		}
	}
	if item == nil || !item.IsActive {
		return nil, ErrCampaignNotFound
	}

	// 检查过期
	if item.ExpiresAt != nil && time.Now().UTC().After(*item.ExpiresAt) {
		return nil, fmt.Errorf("item expired")
	}

	// 检查库存
	if item.Stock >= 0 && item.Stock < quantity {
		return nil, fmt.Errorf("insufficient stock: only %d left", item.Stock)
	}

	// 检查用户积分
	member, ok := s.members[userID]
	if !ok {
		return nil, ErrUnauthorized
	}
	totalCost := item.PricePoints * int64(quantity)
	if member.Points < totalCost {
		return nil, ErrInsufficientPoints
	}

	// 扣积分
	member.Points -= totalCost
	member.TotalSpent += totalCost
	s.members[userID] = member

	// 扣库存
	if item.Stock >= 0 {
		item.Stock -= quantity
	}

	// 添加道具
	s.addUserItemLocked(userID, item.ItemType, item.ItemQty*quantity)

	// 记录日志（简化）
	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID: s.nextTaskID, UserID: userID, Points: -totalCost,
		Balance: member.Points, Reason: "shop", Remark: fmt.Sprintf("购买 %s x%d", item.Name, quantity),
		CreatedAt: time.Now().UTC(),
	})
	s.nextTaskID++

	newQty, _ := s.getUserItemQtyLocked(userID, item.ItemType)

	return &model.BuyShopItemResult{
		ItemType:   item.ItemType,
		ItemName:   item.Name,
		Quantity:   quantity,
		PointsCost: totalCost,
		NewPoints:  member.Points,
		NewQty:     newQty,
	}, nil
}

func (s *MemoryStore) addUserItemLocked(userID string, itemType model.ItemType, qty int) {
	key := userID + ":" + string(itemType)
	existing, ok := s.userItems[key]
	if ok {
		existing.Quantity += qty
		s.userItems[key] = existing
	} else {
		s.userItems[key] = model.UserItem{UserID: userID, ItemType: itemType, Quantity: qty}
	}
}

func (s *MemoryStore) getUserItemQtyLocked(userID string, itemType model.ItemType) (int, bool) {
	key := userID + ":" + string(itemType)
	existing, ok := s.userItems[key]
	if !ok {
		return 0, false
	}
	return existing.Quantity, true
}

func (s *MemoryStore) GetUserItemQty(userID string, itemType model.ItemType) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	qty, _ := s.getUserItemQtyLocked(userID, itemType)
	return qty, nil
}

func (s *MemoryStore) AddUserItem(userID string, itemType model.ItemType, qty int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addUserItemLocked(userID, itemType, qty)
	return nil
}

func (s *MemoryStore) UseUserItem(userID string, itemType model.ItemType, qty int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userID + ":" + string(itemType)
	existing, ok := s.userItems[key]
	if !ok || existing.Quantity < qty {
		return false, nil
	}
	existing.Quantity -= qty
	if existing.Quantity <= 0 {
		delete(s.userItems, key)
	} else {
		s.userItems[key] = existing
	}
	return true, nil
}

func (s *MemoryStore) GetUserItems(userID string) ([]model.UserItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.UserItem, 0)
	for _, v := range s.userItems {
		if v.UserID == userID && v.Quantity > 0 {
			items = append(items, v)
		}
	}
	return items, nil
}

// 🆕 首充礼包

func (s *MemoryStore) GetFirstRechargeStatus(userID string) (*model.UserFirstRecharge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.firstRechargeRecords[userID]
	if !ok {
		return &model.UserFirstRecharge{UserID: userID, Claimed: []string{}}, nil
	}
	cp := rec
	return &cp, nil
}

func (s *MemoryStore) ClaimFirstRecharge(userID string, packID string) (*model.ClaimFirstRechargeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查礼包是否有效
	pack, ok := model.FirstRechargePacks[packID]
	if !ok {
		return nil, fmt.Errorf("invalid first recharge pack: %s", packID)
	}

	// 检查是否已经领取过
	rec, exists := s.firstRechargeRecords[userID]
	if exists {
		for _, claimed := range rec.Claimed {
			if claimed == packID {
				return nil, fmt.Errorf("already claimed pack: %s", packID)
			}
		}
	} else {
		rec = model.UserFirstRecharge{UserID: userID, Claimed: []string{}}
	}

	// 检查积分是否足够
	member, ok := s.members[userID]
	if !ok {
		return nil, ErrUnauthorized
	}
	if member.Points < pack.PricePoints {
		return nil, ErrInsufficientPoints
	}

	// 扣积分
	member.Points -= pack.PricePoints
	member.TotalSpent += pack.PricePoints
	s.members[userID] = member

	// 发放礼包内容
	for _, item := range pack.Items {
		switch item.Type {
		case "points":
			// 已包含在礼包积分中，直接加到余额
			member.Points += int64(item.Qty)
			s.members[userID] = member
		case "hint_card":
			s.addUserItemLocked(userID, model.ItemHintCard, item.Qty)
		case "see_through":
			s.addUserItemLocked(userID, model.ItemSeeThrough, item.Qty)
		case "ten_draw_ticket":
			s.addUserItemLocked(userID, model.ItemTenDrawTicket, item.Qty)
		case "free_draw":
			s.addUserItemLocked(userID, model.ItemFreeDraw, item.Qty)
		case "prize":
			// 随机找一个该系列的稀有款发放（简化：直接给积分等价物）
			pointsExtra := int64(item.Qty * 200) // 稀有款约200积分
			member.Points += pointsExtra
			s.members[userID] = member
		case "month_card":
			// 赠送月卡（一个月的monthly卡）
			card := &model.MonthCard{
				ID: userID + ":monthly:gift", UserID: userID,
				CardType: model.MonthCardMonthly, Price: 0,
				FreeDraws: 2, DrawDiscount: 0.8,
				StartedAt: time.Now().UTC(),
				ExpiresAt: time.Now().UTC().Add(30 * 24 * time.Hour),
				CreatedAt: time.Now().UTC(),
			}
			s.monthCards[userID] = card
		}
	}

	// 记录领取
	member.Points += pack.PricePoints // 退回礼包价值作为积分（首充双倍等效）
	s.members[userID] = member

	rec.Claimed = append(rec.Claimed, packID)
	s.firstRechargeRecords[userID] = rec

	// 记录日志
	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID: s.nextTaskID, UserID: userID, Points: 0,
		Balance: member.Points, Reason: "first_recharge", Remark: fmt.Sprintf("领取%s", pack.Name),
		CreatedAt: time.Now().UTC(),
	})
	s.nextTaskID++

	return &model.ClaimFirstRechargeResult{
		PackID: packID, PackName: pack.Name,
		Items: pack.Items, NewPoints: member.Points,
	}, nil
}
