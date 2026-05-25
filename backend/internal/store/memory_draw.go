package store

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"campaign-lottery-platform/backend/internal/model"
)

// ============================================================
// 盲盒抽奖
// ============================================================

func (s *MemoryStore) CreateDrawRecord(userID, campaignID, prizeID string, _ bool) (model.DrawRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 找到奖品并扣库存
	prizeName := "未知奖品"
	prizes := s.prizes[campaignID]
	found := false
	for i := range prizes {
		if prizes[i].ID == prizeID && prizes[i].Stock > 0 {
			prizes[i].Stock--
			prizeName = prizes[i].Name
			found = true
			break
		}
	}
	s.prizes[campaignID] = prizes
	if !found {
		// 库存不足，记录未中奖
		return s.createMissRecordLocked(userID, campaignID)
	}

	now := time.Now().UTC()
	record := model.DrawRecord{
		ID:         "draw_" + randomSuffix(12),
		CampaignID: campaignID,
		UserID:     userID,
		PrizeID:    &prizeID,
		PrizeName:  prizeName,
		Result:     "win",
		DrawnAt:    now,
	}
	s.drawRecords = append([]model.DrawRecord{record}, s.drawRecords...)

	// 添加到用户库存
	s.inventory = append(s.inventory, model.UserInventory{
		ID:         "inv_" + randomSuffix(12),
		UserID:     userID,
		PrizeID:    prizeID,
		PrizeName:  prizeName,
		Source:     "draw",
		CreatedAt:  now,
	})

	return record, nil
}

func (s *MemoryStore) createMissRecordLocked(userID, campaignID string) (model.DrawRecord, error) {
	now := time.Now().UTC()
	record := model.DrawRecord{
		ID:         "draw_" + randomSuffix(12),
		CampaignID: campaignID,
		UserID:     userID,
		PrizeName:  "未中奖",
		Result:     "miss",
		DrawnAt:    now,
	}
	s.drawRecords = append([]model.DrawRecord{record}, s.drawRecords...)
	return record, nil
}

func (s *MemoryStore) CreateMissRecord(userID, campaignID string, _ bool) (model.DrawRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createMissRecordLocked(userID, campaignID)
}

func (s *MemoryStore) CheckDrawQuota(userID, campaignID string, dailyLimit int) (int, error) {
	// 0 = unlimited
	if dailyLimit <= 0 {
		return 999999, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	used := s.userDrawCounts[userID+":"+campaignID]
	remaining := dailyLimit - used
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

func (s *MemoryStore) DeductDrawQuota(userID, campaignID string, count int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userID + ":" + campaignID
	s.userDrawCounts[key] += count
	return 99 - s.userDrawCounts[key], nil
}

// ============================================================
// 原 Draw 兼容（由 service 接管逻辑后，这里保留桩件）
// ============================================================

func (s *MemoryStore) Draw(token string, campaignID string) (model.DrawResult, error) {
	return model.DrawResult{}, errors.New("use BlindBoxDraw via Service instead")
}

func (s *MemoryStore) UserDrawRecords(token string) ([]model.DrawRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[token]
	if !ok || session.ExpiresAt.Before(time.Now().UTC()) {
		return nil, ErrUnauthorized
	}

	items := make([]model.DrawRecord, 0, 8)
	for _, record := range s.drawRecords {
		if record.UserID == session.UserID {
			items = append(items, record)
		}
	}
	return items, nil
}

// ============================================================
// 库存 / 收集进度
// ============================================================

func (s *MemoryStore) GetUserInventory(userID string) ([]model.UserInventory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.UserInventory, 0, 8)
	for _, inv := range s.inventory {
		if inv.UserID == userID {
			items = append(items, inv)
		}
	}
	return items, nil
}

func (s *MemoryStore) GetSeriesProgress(userID, campaignID, campaignName string) (*model.SeriesProgress, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prizes := s.prizes[campaignID]
	totalItems := len(prizes)
	collectedMap := make(map[string]int)

	for _, inv := range s.inventory {
		if inv.UserID == userID && inv.CampaignID == campaignID {
			collectedMap[inv.PrizeID]++
		}
	}

	collected := make([]model.CollectedPrize, 0, len(collectedMap))
	missing := make([]model.PrizeSummary, 0)
	duplicates := 0

	for _, p := range prizes {
		if count, ok := collectedMap[p.ID]; ok {
			collected = append(collected, model.CollectedPrize{Prize: p, Count: count})
			if count > 1 {
				duplicates += count - 1
			}
		} else {
			missing = append(missing, model.PrizeSummary{
				PrizeID: p.ID, PrizeName: p.Name, PrizeLevel: p.Level,
			})
		}
	}

	pct := 0.0
	if totalItems > 0 {
		pct = float64(len(collected)) / float64(totalItems) * 100
	}
	return &model.SeriesProgress{
		CampaignID:      campaignID,
		CampaignName:    campaignName,
		TotalItems:      totalItems,
		CollectedItems:  len(collected),
		ProgressPercent: pct,
		Duplicates:      duplicates,
		CollectedPrizes: collected,
		MissingPrizes:   missing,
	}, nil
}

func (s *MemoryStore) UserHasPrize(userID, prizeID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, inv := range s.inventory {
		if inv.UserID == userID && inv.PrizeID == prizeID {
			return true, nil
		}
	}
	return false, nil
}

// GetPrizeCount 获取用户某系列某款式的数量
func (s *MemoryStore) GetPrizeCount(userID, prizeID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, inv := range s.inventory {
		if inv.UserID == userID && inv.PrizeID == prizeID {
			count++
		}
	}
	return count, nil
}

func (s *MemoryStore) BlendPrizes(userID string, sourcePrizeID string, campaignID string) (*model.BlendResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 查找源款式信息
	var sourcePrize model.Prize
	var found bool
	for _, prizes := range s.prizes {
		for _, p := range prizes {
			if p.ID == sourcePrizeID {
				sourcePrize = p
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return nil, errors.New("prize not found")
	}

	recipe, ok := model.BlendRecipes[sourcePrize.Level]
	if !ok {
		return nil, errors.New("no recipe for level: " + sourcePrize.Level)
	}

	// 检查用户拥有数量
	count := 0
	for _, inv := range s.inventory {
		if inv.UserID == userID && inv.PrizeID == sourcePrizeID {
			count++
		}
	}
	if count < recipe.NeedCount {
		return nil, fmt.Errorf("need %d of %s, have %d", recipe.NeedCount, sourcePrize.Name, count)
	}

	// 找到目标级别的奖品（同级随机选一个 active 且 stock > 0 的）
	var resultPrize model.Prize
	found = false
	for _, prizes := range s.prizes {
		for _, p := range prizes {
			if p.Level == recipe.ResultLevel && p.Status == "active" && p.Stock > 0 {
				resultPrize = p
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return nil, errors.New("no available prize of level " + recipe.ResultLevel + " to blend into")
	}

	now := time.Now().UTC()

	// 删除 N 条库存记录
	deleted := 0
	newInventory := make([]model.UserInventory, 0, len(s.inventory))
	for _, inv := range s.inventory {
		if inv.UserID == userID && inv.PrizeID == sourcePrizeID && deleted < recipe.NeedCount {
			deleted++
			continue
		}
		newInventory = append(newInventory, inv)
	}
	s.inventory = newInventory

	// 扣库存
	for i := range s.prizes[campaignID] {
		if s.prizes[campaignID][i].ID == resultPrize.ID && s.prizes[campaignID][i].Stock > 0 {
			s.prizes[campaignID][i].Stock--
			break
		}
	}

	// 添加结果到用户库存
	s.inventory = append(s.inventory, model.UserInventory{
		ID:         "inv_" + randomSuffix(12),
		UserID:     userID,
		PrizeID:    resultPrize.ID,
		PrizeName:  resultPrize.Name,
		PrizeLevel: resultPrize.Level,
		CampaignID: campaignID,
		Source:     "blend",
		CreatedAt:  now,
	})

	return &model.BlendResult{
		SourcePrizeID:   sourcePrizeID,
		SourcePrizeName: sourcePrize.Name,
		SourceLevel:     sourcePrize.Level,
		ResultPrizeID:   resultPrize.ID,
		ResultPrizeName: resultPrize.Name,
		ResultLevel:     resultPrize.Level,
		RemainingSrc:    count - recipe.NeedCount,
	}, nil
}
