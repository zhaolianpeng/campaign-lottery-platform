package store

import (
	"fmt"
	"sort"
	"time"

	mathrand "math/rand/v2"

	"campaign-lottery-platform/backend/internal/model"
)

// ============================================================
// 集卡系统扩展
// ============================================================

// DailyCheckIn 每日签到
func (s *MemoryStore) DailyCheckIn(userID string, points int64) (*model.CheckInResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	member, ok := s.members[userID]
	if !ok {
		member = model.UserMember{UserID: userID, Level: model.MemberNormal, Points: 0}
	}

	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)

	// 计算连续签到天数
	lastDate, hasLast := s.checkInDates[userID]
	streak := s.checkInStreaks[userID]
	if hasLast {
		lastDay := lastDate.Truncate(24 * time.Hour)
		diff := today.Sub(lastDay)
		if diff == 24*time.Hour {
			// 连续签到
			streak++
		} else if diff > 24*time.Hour {
			// 中断，重置
			streak = 1
		}
		// diff == 0 表示今天已签过到，保持 streak 不变
	} else {
		streak = 1
	}

	// 更新签到日期
	s.checkInDates[userID] = now
	s.checkInStreaks[userID] = streak

	// 加积分
	totalPoints := points
	isBonus := false
	if streak == 7 {
		totalPoints += 20
		isBonus = true
		// 重置连续天数，重新计数
		s.checkInStreaks[userID] = 0
	}

	member.Points += totalPoints
	s.members[userID] = member

	// 记录积分日志
	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID:        int64(len(s.pointsLog) + 1),
		UserID:    userID,
		Points:    totalPoints,
		Balance:   member.Points,
		Reason:    "daily",
		Remark:    "每日签到",
		CreatedAt: now,
	})

	return &model.CheckInResult{
		PointsAwarded: totalPoints,
		StreakDays:    streak,
		IsBonus:       isBonus,
		NewBalance:    member.Points,
	}, nil
}

// GetCheckInStreak 获取连续签到天数
func (s *MemoryStore) GetCheckInStreak(userID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.checkInStreaks[userID], nil
}

// CheckCollectionCompletion 检查用户是否集齐系列所有普通+稀有款式
func (s *MemoryStore) CheckCollectionCompletion(userID, campaignID string) (*model.CollectionReward, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prizes := s.prizes[campaignID]
	if len(prizes) == 0 {
		return nil, nil
	}

	// 筛选需要集齐的款式（排除 secret 和 limited）
	required := make([]model.Prize, 0, len(prizes))
	campaignName := ""
	for _, p := range prizes {
		if p.Level == "secret" || p.Level == "limited" {
			continue
		}
		if p.Status != "active" {
			continue
		}
		required = append(required, p)
		campaignName = p.CampaignID // fallback name source
	}

	if len(required) == 0 {
		return nil, nil
	}

	// 查找 campaign name
	if c, ok := s.campaigns[campaignID]; ok {
		campaignName = c.Name
	}

	// 构建用户已拥有的奖品集合
	owned := make(map[string]bool)
	for _, inv := range s.inventory {
		if inv.UserID == userID && inv.CampaignID == campaignID {
			owned[inv.PrizeID] = true
		}
	}

	// 检查是否拥有所有必需的款式
	for _, p := range required {
		if !owned[p.ID] {
			return nil, nil
		}
	}

	// 集齐了
	return &model.CollectionReward{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		RewardType:   "title",
		RewardName:   "收集大师",
		Description:  "恭喜集齐",
	}, nil
}

// GrantCollectionReward 发放集齐奖励
func (s *MemoryStore) GrantCollectionReward(userID string, reward *model.CollectionReward) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	// 添加库存记录
	s.inventory = append(s.inventory, model.UserInventory{
		ID:         "inv_" + randomSuffix(12),
		UserID:     userID,
		PrizeID:    reward.CampaignID,
		PrizeName:  reward.RewardName,
		CampaignID: reward.CampaignID,
		Source:     "collection_reward",
		CreatedAt:  now,
	})

	// 加积分
	member, ok := s.members[userID]
	if !ok {
		member = model.UserMember{UserID: userID, Level: model.MemberNormal, Points: 0}
	}
	member.Points += 500
	s.members[userID] = member

	// 记录积分日志
	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID:        int64(len(s.pointsLog) + 1),
		UserID:    userID,
		Points:    500,
		Balance:   member.Points,
		Reason:    "collection_reward",
		Remark:    "集齐系列奖励: " + reward.RewardName,
		CreatedAt: now,
	})

	return nil
}

// GetLeaderboard 获取收集排行榜
func (s *MemoryStore) GetLeaderboard(limit int) ([]model.LeaderboardEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.users) == 0 {
		return []model.LeaderboardEntry{}, nil
	}

	// 收集每个系列的总款式数（排除 secret 和 limited）
	seriesTotal := make(map[string]int)
	for cid, prizes := range s.prizes {
		count := 0
		for _, p := range prizes {
			if p.Level != "secret" && p.Level != "limited" && p.Status == "active" {
				count++
			}
		}
		if count > 0 {
			seriesTotal[cid] = count
		}
	}

	// 统计每个用户在各系列的收集进度
	type userProgress struct {
		collectedCount int
		seriesCount    int // 已集齐的系列数
		seriesTotal    int // 总系列款式数（最大）
	}
	userMap := make(map[string]*userProgress)

	for _, inv := range s.inventory {
		if inv.CampaignID == "" {
			continue
		}
		// 只统计非 secret/limited 的奖品
		total, ok := seriesTotal[inv.CampaignID]
		if !ok {
			continue
		}
		if _, exists := userMap[inv.UserID]; !exists {
			userMap[inv.UserID] = &userProgress{}
		}
		userMap[inv.UserID].seriesTotal = total
	}

	// 按用户+系列去重统计 collectedCount
	type userCampaignKey struct {
		userID     string
		campaignID string
	}
	seen := make(map[userCampaignKey]map[string]bool)

	for _, inv := range s.inventory {
		if _, ok := seriesTotal[inv.CampaignID]; !ok {
			continue
		}
		key := userCampaignKey{userID: inv.UserID, campaignID: inv.CampaignID}
		if seen[key] == nil {
			seen[key] = make(map[string]bool)
		}
		seen[key][inv.PrizeID] = true
	}

	for key, prizes := range seen {
		p := userMap[key.userID]
		if p == nil {
			continue
		}
		p.collectedCount += len(prizes)
		if len(prizes) >= seriesTotal[key.campaignID] {
			p.seriesCount++
		}
	}

	// 构建排行榜条目
	entries := make([]model.LeaderboardEntry, 0, len(userMap))
	for uid, progress := range userMap {
		nickname := ""
		if u, ok := s.users[uid]; ok {
			nickname = u.Nickname
		}
		totalCount := progress.seriesTotal
		pct := 0.0
		if totalCount > 0 {
			pct = float64(progress.collectedCount) / float64(totalCount) * 100
		}
		entries = append(entries, model.LeaderboardEntry{
			UserID:          uid,
			Nickname:        nickname,
			CollectedCount:  progress.collectedCount,
			TotalCount:      totalCount,
			ProgressPercent: pct,
			SeriesCompleted: progress.seriesCount,
		})
	}

	// 按 collectedCount 降序排序
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].CollectedCount != entries[j].CollectedCount {
			return entries[i].CollectedCount > entries[j].CollectedCount
		}
		return entries[i].UserID < entries[j].UserID
	})

	// 取前 limit 名并填充排名
	if limit > len(entries) {
		limit = len(entries)
	}
	result := entries[:limit]
	for i := range result {
		result[i].Rank = i + 1
	}

	return result, nil
}

// GetCampaignHint 获取系列摇盒提示文案
func (s *MemoryStore) GetCampaignHint(campaignID string) *model.HintMessage {
	hints := []model.HintMessage{
		{Type: "hot", Content: "据大数据分析，本系列当前隐藏款热度较高"},
		{Type: "social", Content: "已有 xx 位用户抽到此系列的隐藏款"},
		{Type: "luck", Content: "刚刚有用户十连抽中了稀有款！"},
	}
	idx := mathrand.N(len(hints))
	return &hints[idx]
}

// ShareReward 分享奖励
func (s *MemoryStore) ShareReward(userID string, points int64) (*model.ShareRewardResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查当日分享次数
	count := s.shareCounts[userID]
	if count >= 10 {
		return nil, fmt.Errorf("今日分享次数已达上限")
	}

	// 增加分享次数
	s.shareCounts[userID] = count + 1

	// 加积分
	member, ok := s.members[userID]
	if !ok {
		member = model.UserMember{UserID: userID, Level: model.MemberNormal, Points: 0}
	}
	member.Points += points
	s.members[userID] = member

	// 记录积分日志
	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID:        int64(len(s.pointsLog) + 1),
		UserID:    userID,
		Points:    points,
		Balance:   member.Points,
		Reason:    "share",
		Remark:    "分享奖励",
		CreatedAt: time.Now().UTC(),
	})

	return &model.ShareRewardResult{
		PointsAwarded: points,
		DailyLeft:     10 - s.shareCounts[userID],
		NewBalance:    member.Points,
	}, nil
}

// GetShareDailyCount 获取今日已分享次数
func (s *MemoryStore) GetShareDailyCount(userID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.shareCounts[userID], nil
}
