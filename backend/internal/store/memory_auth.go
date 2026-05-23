package store

import (
	"sort"
	"time"

	"campaign-lottery-platform/backend/internal/model"
)

// ============================================================
// 基础 CRUD
// ============================================================

func (s *MemoryStore) CreateGuestSession(nickname string) (model.User, model.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if nickname == "" {
		nickname = "Guest" + randomSuffix(4)
	}

	user := model.User{
		ID:        "usr_" + randomSuffix(12),
		Nickname:  nickname,
		CreatedAt: time.Now().UTC(),
	}
	s.users[user.ID] = user

	session := model.Session{
		Token:     "utk_" + randomSuffix(24),
		UserID:    user.ID,
		ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
	}
	s.sessions[session.Token] = session

	// 新用户赠送初始积分
	now := time.Now().UTC()
	s.members[user.ID] = model.UserMember{
		UserID:    user.ID,
		Level:     model.MemberNormal,
		Points:    100,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.pointsLog = append(s.pointsLog, model.UserPointsLog{
		ID:        int64(len(s.pointsLog) + 1),
		UserID:    user.ID,
		Points:    100,
		Balance:   100,
		Reason:    "welcome",
		Remark:    "新用户注册赠送",
		CreatedAt: now,
	})

	return user, session, nil
}

func (s *MemoryStore) Campaigns() []model.Campaign {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.Campaign, 0, len(s.campaigns))
	for _, campaign := range s.campaigns {
		items = append(items, campaign)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].StartsAt.Before(items[j].StartsAt)
	})
	return items
}

func (s *MemoryStore) GetCampaign(campaignID string) (model.Campaign, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.campaigns[campaignID]
	if !ok {
		return model.Campaign{}, ErrCampaignNotFound
	}
	return c, nil
}

func (s *MemoryStore) PrizeList(campaignID string) []model.Prize {
	s.mu.RLock()
	defer s.mu.RUnlock()
	prizes := s.prizes[campaignID]
	items := make([]model.Prize, len(prizes))
	copy(items, prizes)
	return items
}

func (s *MemoryStore) UserFromToken(token string) (model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[token]
	if !ok || session.ExpiresAt.Before(time.Now().UTC()) {
		return model.User{}, ErrUnauthorized
	}
	user, ok := s.users[session.UserID]
	if !ok {
		return model.User{}, ErrUnauthorized
	}
	return user, nil
}

// ============================================================
// 管理员
// ============================================================

func (s *MemoryStore) AdminLogin(username string, password string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if username != s.adminUser || password != s.adminPassword {
		return "", ErrBadAdminAuth
	}
	token := "atk_" + randomSuffix(24)
	s.adminSessions[token] = time.Now().UTC().Add(12 * time.Hour)
	return token, nil
}

func (s *MemoryStore) AdminOverview(token string) (model.AdminOverview, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.ensureAdmin(token); err != nil {
		return model.AdminOverview{}, err
	}

	prizeSummaries := make([]model.PrizeSummary, 0, 8)
	for _, prizes := range s.prizes {
		for _, prize := range prizes {
			prizeSummaries = append(prizeSummaries, model.PrizeSummary{
				PrizeID: prize.ID, PrizeName: prize.Name, PrizeLevel: prize.Level, Stock: prize.Stock,
			})
		}
	}

	campaigns := make([]model.Campaign, 0, len(s.campaigns))
	for _, campaign := range s.campaigns {
		campaigns = append(campaigns, campaign)
	}

	recentDraws := make([]model.DrawRecord, 0, minInt(10, len(s.drawRecords)))
	for i, record := range s.drawRecords {
		if i >= 10 {
			break
		}
		recentDraws = append(recentDraws, record)
	}

	totalWins := 0
	for _, record := range s.drawRecords {
		if record.Result == "win" {
			totalWins++
		}
	}

	balance := make(map[string]int, len(s.userDrawCounts))
	for key, used := range s.userDrawCounts {
		balance[key] = used
	}

	return model.AdminOverview{
		TotalUsers:      len(s.users),
		TotalDraws:      len(s.drawRecords),
		TotalWins:       totalWins,
		Campaigns:       campaigns,
		PrizeSummaries:  prizeSummaries,
		RecentDraws:     recentDraws,
		UserDrawBalance: balance,
	}, nil
}

func (s *MemoryStore) AdminDrawRecords(token string) ([]model.DrawRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureAdmin(token); err != nil {
		return nil, err
	}
	items := make([]model.DrawRecord, len(s.drawRecords))
	copy(items, s.drawRecords)
	return items, nil
}

func (s *MemoryStore) AdminCampaigns(token string) ([]model.Campaign, error) {
	if err := s.ensureAdmin(token); err != nil {
		return nil, err
	}
	return s.Campaigns(), nil
}

func (s *MemoryStore) CreateCampaign(token string, input model.CampaignMutation) (model.Campaign, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureAdmin(token); err != nil {
		return model.Campaign{}, err
	}
	campaign := model.Campaign{
		ID:              "camp_" + randomSuffix(10),
		Name:            input.Name,
		Slug:            input.Slug,
		Status:          input.Status,
		StartsAt:        input.StartsAt,
		EndsAt:          input.EndsAt,
		DailyDrawLimit:  input.DailyDrawLimit,
		MissWeight:      input.MissWeight,
		BannerImageURL:  input.BannerImageURL,
		CampaignSummary: input.CampaignSummary,
		PityConfig:      input.PityConfig,
	}
	s.campaigns[campaign.ID] = campaign
	return campaign, nil
}

func (s *MemoryStore) UpdateCampaign(token string, campaignID string, input model.CampaignMutation) (model.Campaign, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureAdmin(token); err != nil {
		return model.Campaign{}, err
	}
	_, ok := s.campaigns[campaignID]
	if !ok {
		return model.Campaign{}, ErrCampaignNotFound
	}
	campaign := model.Campaign{
		ID: campaignID, Name: input.Name, Slug: input.Slug, Status: input.Status,
		StartsAt: input.StartsAt, EndsAt: input.EndsAt, DailyDrawLimit: input.DailyDrawLimit,
		MissWeight: input.MissWeight, BannerImageURL: input.BannerImageURL, CampaignSummary: input.CampaignSummary,
		PityConfig: input.PityConfig,
	}
	s.campaigns[campaignID] = campaign
	return campaign, nil
}

func (s *MemoryStore) DeleteCampaign(token string, campaignID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureAdmin(token); err != nil {
		return err
	}
	delete(s.campaigns, campaignID)
	delete(s.prizes, campaignID)
	return nil
}

func (s *MemoryStore) AdminPrizes(token string, campaignID string) ([]model.Prize, error) {
	if err := s.ensureAdmin(token); err != nil {
		return nil, err
	}
	return s.PrizeList(campaignID), nil
}

func (s *MemoryStore) CreatePrize(token string, campaignID string, input model.PrizeMutation) (model.Prize, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureAdmin(token); err != nil {
		return model.Prize{}, err
	}
	prize := model.Prize{
		ID: "prize_" + randomSuffix(10), CampaignID: campaignID, Name: input.Name,
		Level: input.Level, Stock: input.Stock, ProbabilityWeight: input.ProbabilityWeight, Status: input.Status,
	}
	s.prizes[campaignID] = append(s.prizes[campaignID], prize)
	return prize, nil
}

func (s *MemoryStore) UpdatePrize(token string, prizeID string, input model.PrizeMutation) (model.Prize, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureAdmin(token); err != nil {
		return model.Prize{}, err
	}
	for campaignID, prizes := range s.prizes {
		for i := range prizes {
			if prizes[i].ID == prizeID {
				prizes[i].Name = input.Name
				prizes[i].Level = input.Level
				prizes[i].Stock = input.Stock
				prizes[i].ProbabilityWeight = input.ProbabilityWeight
				prizes[i].Status = input.Status
				s.prizes[campaignID] = prizes
				return prizes[i], nil
			}
		}
	}
	return model.Prize{}, ErrCampaignNotFound
}

func (s *MemoryStore) DeletePrize(token string, prizeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureAdmin(token); err != nil {
		return err
	}
	for campaignID, prizes := range s.prizes {
		filtered := prizes[:0]
		for _, prize := range prizes {
			if prize.ID != prizeID {
				filtered = append(filtered, prize)
			}
		}
		s.prizes[campaignID] = filtered
	}
	return nil
}

func (s *MemoryStore) FulfillmentTasks(token string) ([]model.FulfillmentTask, error) {
	if err := s.ensureAdmin(token); err != nil {
		return nil, err
	}
	return []model.FulfillmentTask{}, nil
}

func (s *MemoryStore) UpdateFulfillmentTask(token string, taskID int64, input model.FulfillmentTaskMutation) (model.FulfillmentTask, error) {
	if err := s.ensureAdmin(token); err != nil {
		return model.FulfillmentTask{}, err
	}
	return model.FulfillmentTask{ID: taskID, Status: input.Status, OperatorNote: input.OperatorNote}, nil
}

func (s *MemoryStore) GetDrawStatistics(token, campaignID string) (*model.DrawStatistics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureAdmin(token); err != nil {
		return nil, err
	}

	totalDraws := int64(len(s.drawRecords))
	totalUsers := int64(len(s.users))
	totalWins := int64(0)
	prizeCounts := make(map[string]int64)

	for _, r := range s.drawRecords {
		if r.CampaignID == campaignID || campaignID == "" {
			if r.Result == "win" && r.PrizeID != nil {
				prizeCounts[*r.PrizeID]++
				totalWins++
			}
		}
	}

	breakdown := make([]model.PrizeStatItem, 0, len(prizeCounts))
	for _, prizes := range s.prizes {
		for _, p := range prizes {
			if count, ok := prizeCounts[p.ID]; ok {
				percent := 0.0
				if totalWins > 0 {
					percent = float64(count) / float64(totalWins) * 100
				}
				breakdown = append(breakdown, model.PrizeStatItem{
					PrizeID: p.ID, PrizeName: p.Name, Level: p.Level, Count: count, Percent: percent,
				})
			}
		}
	}

	winRate := 0.0
	if totalDraws > 0 {
		winRate = float64(totalWins) / float64(totalDraws) * 100
	}

	return &model.DrawStatistics{
		TotalDraws:     totalDraws,
		TotalUsers:     totalUsers,
		TotalWins:      totalWins,
		WinRate:        winRate,
		PrizeBreakdown: breakdown,
	}, nil
}
