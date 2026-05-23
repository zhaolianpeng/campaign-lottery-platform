package store

import (
	"fmt"
	"time"

	"campaign-lottery-platform/backend/internal/model"
)

// ============================================================
// 积分/会员
// ============================================================

func (s *MemoryStore) GetUserMember(userID string) (*model.UserMember, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if m, ok := s.members[userID]; ok {
		return &m, nil
	}
	return &model.UserMember{UserID: userID, Level: model.MemberNormal}, nil
}

func (s *MemoryStore) LogPoints(userID string, points int64, balance int64, reason, remark string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID:        int64(len(s.pointsLog) + 1),
		UserID:    userID,
		Points:    points,
		Balance:   balance,
		Reason:    reason,
		Remark:    remark,
		CreatedAt: time.Now().UTC(),
	})
	return nil
}

func (s *MemoryStore) GetPointsLog(userID string) ([]model.UserPointsLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.UserPointsLog, 0, 8)
	for _, log := range s.pointsLog {
		if log.UserID == userID {
			items = append(items, log)
		}
	}
	return items, nil
}

func (s *MemoryStore) UpdateUserMember(member *model.UserMember) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.members[member.UserID] = *member
	return nil
}

func (s *MemoryStore) RedeemPrize(userID string, input model.RedeemRequest) (*model.RedeemResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 找奖品
	var prizeName, prizeLevel string
	var found bool
	for _, prizes := range s.prizes {
		for _, p := range prizes {
			if p.ID == input.PrizeID && p.Status == "active" {
				prizeName = p.Name
				prizeLevel = p.Level
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("prize not found")
	}

	pointsCost := map[string]int64{
		"common": 100, "rare": 500, "secret": 2000, "limited": 5000,
	}[prizeLevel]
	if pointsCost == 0 {
		pointsCost = 100
	}

	member, ok := s.members[userID]
	if !ok {
		member = model.UserMember{UserID: userID, Level: model.MemberNormal, Points: 0}
	}
	if member.Points < pointsCost {
		return nil, fmt.Errorf("insufficient points: have %d, need %d", member.Points, pointsCost)
	}

	member.Points -= pointsCost
	s.members[userID] = member

	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID:        int64(len(s.pointsLog) + 1),
		UserID:    userID,
		Points:    -pointsCost,
		Balance:   member.Points,
		Reason:    "redeem",
		Remark:    "兑换: " + prizeName,
		CreatedAt: time.Now().UTC(),
	})

	// 加库存
	prizeCampaignID := ""
	for cid, prizes := range s.prizes {
		for _, p := range prizes {
			if p.ID == input.PrizeID {
				prizeCampaignID = cid
				break
			}
		}
		if prizeCampaignID != "" {
			break
		}
	}
	now := time.Now().UTC()
	s.inventory = append(s.inventory, model.UserInventory{
		ID:         "inv_" + randomSuffix(12),
		UserID:     userID,
		PrizeID:    input.PrizeID,
		PrizeName:  prizeName,
		PrizeLevel: prizeLevel,
		CampaignID: prizeCampaignID,
		Source:     "redeem",
		CreatedAt:  now,
	})

	return &model.RedeemResult{
		RecordID:   "rdm_" + randomSuffix(12),
		PrizeID:    input.PrizeID,
		PrizeName:  prizeName,
		PointsCost: pointsCost,
		Remaining:  member.Points,
	}, nil
}
