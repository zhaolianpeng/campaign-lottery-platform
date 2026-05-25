package store

import (
	"strconv"
	"time"

	"campaign-lottery-platform/backend/internal/model"
)

// ─────────────────────────────────────────────────────────────
// 🆕 战令系统 MemoryStore 实现
// ─────────────────────────────────────────────────────────────

func (s *MemoryStore) GetActiveSeason() (*model.BattlePassSeason, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, season := range s.battlePassSeasons {
		if season.Status == "active" {
			s := season
			return &s, nil
		}
	}
	return nil, ErrNoActiveSeason
}

func (s *MemoryStore) GetUserBattlePass(userID string, seasonID int) (*model.BattlePass, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := userID + ":" + strconv.Itoa(seasonID)
	if bp, ok := s.battlePasses[key]; ok {
		b := *bp
		return &b, nil
	}
	// 创建免费版战令
	now := time.Now().UTC()
	_, err := s.GetActiveSeason()
	if err != nil {
		return nil, err
	}
	bp := &model.BattlePass{
		UserID: userID, SeasonID: seasonID, PassType: "free",
		Level: 1, XP: 0, TotalXP: 0, ClaimedLevels: []int{}, UpdatedAt: now,
	}
	s.battlePasses[key] = bp
	b := *bp
	return &b, nil
}

func (s *MemoryStore) BuyBattlePass(userID string, seasonID int, pointsCost int64) (*model.BattlePass, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userID + ":" + strconv.Itoa(seasonID)
	bp, ok := s.battlePasses[key]
	if !ok {
		return nil, ErrCampaignNotFound
	}
	if bp.PassType == "paid" {
		return nil, ErrAlreadyPurchased
	}
	now := time.Now().UTC()
	bp.PassType = "paid"
	bp.BoughtAt = now
	bp.UpdatedAt = now
	b := *bp
	return &b, nil
}

func (s *MemoryStore) AddBattlePassXP(userID string, seasonID int, xp int) (*model.BattlePass, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userID + ":" + strconv.Itoa(seasonID)
	bp, ok := s.battlePasses[key]
	if !ok {
		return nil, ErrCampaignNotFound
	}
	season, err := s.GetActiveSeason()
	if err != nil {
		return nil, err
	}
	bp.XP += xp
	bp.TotalXP += xp
	for bp.Level < season.MaxLevel && bp.XP >= season.XPPerLevel {
		bp.XP -= season.XPPerLevel
		bp.Level++
	}
	bp.UpdatedAt = time.Now().UTC()
	b := *bp
	return &b, nil
}

func (s *MemoryStore) ClaimBattlePassReward(userID string, seasonID int, level int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userID + ":" + strconv.Itoa(seasonID)
	bp, ok := s.battlePasses[key]
	if !ok {
		return false, ErrCampaignNotFound
	}
	if bp.Level < level {
		return false, ErrNotEligible
	}
	for _, l := range bp.ClaimedLevels {
		if l == level {
			return false, nil // already claimed
		}
	}
	bp.ClaimedLevels = append(bp.ClaimedLevels, level)
	return true, nil
}

func (s *MemoryStore) GetBattlePassTasks(seasonID int) ([]model.BattlePassTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tasks := make([]model.BattlePassTask, 0)
	for _, t := range s.battlePassTasks {
		if t.SeasonID == seasonID {
			tasks = append(tasks, t)
		}
	}
	return tasks, nil
}

func (s *MemoryStore) GetBattlePassTaskProgress(userID string, seasonID int) ([]model.BattlePassTaskProgress, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	progs := make([]model.BattlePassTaskProgress, 0)
	for _, p := range s.battlePassTaskProgress {
		if p.UserID == userID {
			progs = append(progs, p)
		}
	}
	return progs, nil
}

func (s *MemoryStore) UpdateTaskProgress(userID string, taskID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userID + ":" + strconv.Itoa(taskID)
	prog, ok := s.battlePassTaskProgress[key]
	if !ok {
		s.battlePassTaskProgress[key] = model.BattlePassTaskProgress{
			UserID: userID, TaskID: taskID, Progress: 1, Completed: false,
		}
		return nil
	}
	prog.Progress++
	s.battlePassTaskProgress[key] = prog
	return nil
}

func (s *MemoryStore) GetBattlePassRewards(seasonID int) ([]model.BattlePassReward, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rewards := make([]model.BattlePassReward, 0)
	for _, r := range s.battlePassRewards {
		if r.Level <= seasonID/1000*1000+50 { // simplified filter
			rewards = append(rewards, r)
		}
	}
	return rewards, nil
}
