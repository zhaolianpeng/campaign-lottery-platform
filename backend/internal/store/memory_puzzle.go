package store

import (
	"time"
)

import (
	"fmt"

	"campaign-lottery-platform/backend/internal/model"
)

// ============================================================
// 🆕 v1.6 碎片拼图 MemoryStore 实现
// ============================================================

// GetActivePuzzleTemplates 获取当前活跃的拼图模板列表
func (s *MemoryStore) GetActivePuzzleTemplates() []model.PuzzleTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []model.PuzzleTemplate
	for _, t := range s.puzzleTemplates {
		if t.IsActive {
			active = append(active, t)
		}
	}
	return active
}

// GetPuzzleTemplate 获取拼图模板详情
func (s *MemoryStore) GetPuzzleTemplate(templateID string) (*model.PuzzleTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.puzzleTemplates {
		if t.ID == templateID {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("puzzle template not found: %s", templateID)
}

// GetOrCreatePuzzleProgress 获取或创建用户拼图进度
func (s *MemoryStore) GetOrCreatePuzzleProgress(userID, templateID string) (*model.PuzzleProgress, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := userID + ":" + templateID
	progress, ok := s.puzzleProgresses[key]
	if ok {
		return progress, nil
	}

	var template *model.PuzzleTemplate
	for i := range s.puzzleTemplates {
		if s.puzzleTemplates[i].ID == templateID {
			template = &s.puzzleTemplates[i]
			break
		}
	}
	if template == nil {
		return nil, fmt.Errorf("puzzle template not found: %s", templateID)
	}

	progress = &model.PuzzleProgress{
		UserID:      userID,
		TemplateID:  templateID,
		Collected:   []int{},
		TotalPieces: template.TotalPieces,
		IsCompleted: false,
	}
	s.puzzleProgresses[key] = progress
	return progress, nil
}

// AddPuzzlePiece 收集碎片，返回是否是新收集
func (s *MemoryStore) AddPuzzlePiece(userID, templateID string, pieceIndex int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := userID + ":" + templateID
	progress, ok := s.puzzleProgresses[key]
	if !ok {
		return false, fmt.Errorf("puzzle progress not found for user=%s template=%s", userID, templateID)
	}

	for _, idx := range progress.Collected {
		if idx == pieceIndex {
			return false, nil
		}
	}

	progress.Collected = append(progress.Collected, pieceIndex)
	return true, nil
}

// ComposePuzzle 合成拼图（集齐所有碎片后兑换奖励）
func (s *MemoryStore) ComposePuzzle(userID, templateID string) (*model.ComposePuzzleResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := userID + ":" + templateID
	progress, ok := s.puzzleProgresses[key]
	if !ok {
		return nil, fmt.Errorf("puzzle progress not found for user=%s template=%s", userID, templateID)
	}

	if progress.IsCompleted {
		return nil, fmt.Errorf("puzzle already completed")
	}

	var template *model.PuzzleTemplate
	for i := range s.puzzleTemplates {
		if s.puzzleTemplates[i].ID == templateID {
			template = &s.puzzleTemplates[i]
			break
		}
	}
	if template == nil {
		return nil, fmt.Errorf("puzzle template not found: %s", templateID)
	}

	if len(progress.Collected) < template.TotalPieces {
		return nil, fmt.Errorf("not all pieces collected: have %d, need %d", len(progress.Collected), template.TotalPieces)
	}

	now := time.Now().UTC()
	progress.IsCompleted = true
	progress.CompletedAt = &now

	result := &model.ComposePuzzleResult{
		TemplateID:   templateID,
		TemplateName: template.Name,
		RewardType:   template.RewardType,
		RewardName:   template.RewardName,
		RewardQty:    template.RewardQty,
	}

	switch template.RewardType {
	case "points":
		member, ok := s.members[userID]
		if !ok {
			member = model.UserMember{UserID: userID, Level: model.MemberNormal, Points: 0, CreatedAt: now, UpdatedAt: now}
		}
		member.Points += int64(template.RewardQty)
		member.UpdatedAt = now
		s.members[userID] = member
		s.pointsLog = append(s.pointsLog, model.UserPointsLog{
			ID:        int64(len(s.pointsLog) + 1),
			UserID:    userID,
			Points:    int64(template.RewardQty),
			Balance:   member.Points,
			Reason:    "puzzle_compose",
			Remark:    "拼图合成奖励: " + template.Name,
			CreatedAt: now,
		})
	case "draw_ticket":
		s.addUserItemLocked(userID, model.ItemFreeDraw, template.RewardQty)
	case "prize":
		s.inventory = append(s.inventory, model.UserInventory{
			ID:         s.GenerateID(),
			UserID:     userID,
			PrizeID:    template.RewardID,
			PrizeName:  template.RewardName,
			PrizeLevel: "",
			CampaignID: template.CampaignID,
			Source:     "puzzle_compose",
			CreatedAt:  now,
		})
	}

	return result, nil
}

// GetPuzzleInfo 获取拼图详细信息（进度+模板）
func (s *MemoryStore) GetPuzzleInfo(userID, templateID string) (*model.PuzzleInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := userID + ":" + templateID
	progress, ok := s.puzzleProgresses[key]
	if !ok {
		return nil, fmt.Errorf("puzzle progress not found for user=%s template=%s", userID, templateID)
	}

	var template *model.PuzzleTemplate
	for i := range s.puzzleTemplates {
		if s.puzzleTemplates[i].ID == templateID {
			template = &s.puzzleTemplates[i]
			break
		}
	}
	if template == nil {
		return nil, fmt.Errorf("puzzle template not found: %s", templateID)
	}

	collectedSet := make(map[int]bool)
	for _, idx := range progress.Collected {
		collectedSet[idx] = true
	}

	var collectedNames []string
	var missingNames []string
	for i, name := range template.PieceNames {
		if collectedSet[i] {
			collectedNames = append(collectedNames, name)
		} else {
			missingNames = append(missingNames, name)
		}
	}

	progressPercent := 0.0
	if template.TotalPieces > 0 {
		progressPercent = float64(len(progress.Collected)) / float64(template.TotalPieces) * 100
	}

	return &model.PuzzleInfo{
		Template:        template,
		Progress:        progress,
		CollectedNames:  collectedNames,
		MissingNames:    missingNames,
		ProgressPercent: progressPercent,
	}, nil
}

// CreatePuzzleTeam 创建拼图小队
func (s *MemoryStore) CreatePuzzleTeam(captainID, templateID string) (*model.PuzzleTeam, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var totalPieces int
	found := false
	for _, t := range s.puzzleTemplates {
		if t.ID == templateID {
			totalPieces = t.TotalPieces
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("puzzle template not found: %s", templateID)
	}

	s.puzzleTeamIDSeq++
	now := time.Now().UTC()
	team := &model.PuzzleTeam{
		ID:          s.GenerateID(),
		TemplateID:  templateID,
		CaptainID:   captainID,
		Members:     []string{captainID},
		Shared:      []int{},
		TotalPieces: totalPieces,
		IsCompleted: false,
		CreatedAt:   now,
	}
	s.puzzleTeams[team.ID] = team
	return team, nil
}

// JoinPuzzleTeam 加入拼图小队
func (s *MemoryStore) JoinPuzzleTeam(userID, teamID string) (*model.PuzzleTeam, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.puzzleTeams[teamID]
	if !ok {
		return nil, fmt.Errorf("puzzle team not found: %s", teamID)
	}

	for _, m := range team.Members {
		if m == userID {
			return team, nil
		}
	}

	team.Members = append(team.Members, userID)
	return team, nil
}

// GetPuzzleTeam 获取拼图小队信息
func (s *MemoryStore) GetPuzzleTeam(teamID string) (*model.PuzzleTeam, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	team, ok := s.puzzleTeams[teamID]
	if !ok {
		return nil, fmt.Errorf("puzzle team not found: %s", teamID)
	}
	return team, nil
}

// GetUserPuzzleTeams 获取用户加入的拼图小队
func (s *MemoryStore) GetUserPuzzleTeams(userID string) ([]model.PuzzleTeam, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var teams []model.PuzzleTeam
	for _, team := range s.puzzleTeams {
		for _, m := range team.Members {
			if m == userID {
				teams = append(teams, *team)
				break
			}
		}
	}
	return teams, nil
}

// GetUserPuzzleProgresses 获取用户所有拼图进度
func (s *MemoryStore) GetUserPuzzleProgresses(userID string) ([]model.PuzzleInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prefix := userID + ":"
	var infos []model.PuzzleInfo
	for key, progress := range s.puzzleProgresses {
		if len(key) < len(prefix) || key[:len(prefix)] != prefix {
			continue
		}
		for _, t := range s.puzzleTemplates {
			if t.ID == progress.TemplateID {
				collectedSet := make(map[int]bool)
				for _, idx := range progress.Collected {
					collectedSet[idx] = true
				}
				var collectedNames, missingNames []string
				for i, name := range t.PieceNames {
					if collectedSet[i] {
						collectedNames = append(collectedNames, name)
					} else {
						missingNames = append(missingNames, name)
					}
				}
				progressPercent := 0.0
				if t.TotalPieces > 0 {
					progressPercent = float64(len(progress.Collected)) / float64(t.TotalPieces) * 100
				}
				infos = append(infos, model.PuzzleInfo{
					Template:        &t,
					Progress:        progress,
					CollectedNames:  collectedNames,
					MissingNames:    missingNames,
					ProgressPercent: progressPercent,
				})
				break
			}
		}
	}
	return infos, nil
}

// SharePuzzlePiece 在拼图小队中共享碎片
func (s *MemoryStore) SharePuzzlePiece(userID, teamID string, pieceIndex int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.puzzleTeams[teamID]
	if !ok {
		return false, fmt.Errorf("puzzle team not found: %s", teamID)
	}

	isMember := false
	for _, m := range team.Members {
		if m == userID {
			isMember = true
			break
		}
	}
	if !isMember {
		return false, fmt.Errorf("user %s is not a member of team %s", userID, teamID)
	}

	for _, idx := range team.Shared {
		if idx == pieceIndex {
			return false, nil
		}
	}

	team.Shared = append(team.Shared, pieceIndex)
	return true, nil
}
