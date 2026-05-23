package store

import (
	"fmt"
	"strconv"
	"time"

	"campaign-lottery-platform/backend/internal/model"
)

// ============================================================
// 🆕 社交裂变：邀请助力
// ============================================================

// CreateInviteRecord creates a new invite record
func (s *MemoryStore) CreateInviteRecord(inviterID, inviteeID string) *model.InviteRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.inviteIDSeq++
	record := model.InviteRecord{
		ID:        "inv_" + strconv.Itoa(s.inviteIDSeq),
		InviterID: inviterID,
		InviteeID: inviteeID,
		CreatedAt: time.Now().UTC(),
	}
	s.inviteRecords = append(s.inviteRecords, record)
	return &record
}

// GetInviteRecords returns all invite records where InviterID == userID
func (s *MemoryStore) GetInviteRecords(userID string) []model.InviteRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []model.InviteRecord
	for _, r := range s.inviteRecords {
		if r.InviterID == userID {
			result = append(result, r)
		}
	}
	return result
}

// GetInviteStats returns total invite count and total assist count for a user
func (s *MemoryStore) GetInviteStats(userID string) (invites int, assists int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, r := range s.inviteRecords {
		if r.InviterID == userID {
			invites++
		}
	}
	for _, a := range s.assistActions {
		if a.InviterID == userID {
			assists++
		}
	}
	return
}

// GetOrCreateAssistProgress returns existing assist progress or creates a new one.
func (s *MemoryStore) GetOrCreateAssistProgress(inviterID string, assistType model.AssistType) *model.AssistProgress {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s", inviterID, assistType)
	if p, ok := s.assistProgress[key]; ok {
		return &p
	}

	var targetCount int
	switch assistType {
	case model.AssistFreeDraw:
		targetCount = 3
	case model.AssistPityReduce:
		targetCount = 5
	case model.AssistCraftBoost:
		targetCount = 2
	default:
		targetCount = 3
	}

	created := time.Now().UTC()
	progress := model.AssistProgress{
		InviterID:   inviterID,
		AssistType:  assistType,
		TargetCount: targetCount,
		Current:     0,
		Claimed:     false,
		ExpiresAt:   created.Add(24 * time.Hour),
		CreatedAt:   created,
	}
	s.assistProgress[key] = progress
	return &progress
}

// IsAssistActionRecorded checks if the same helper already assisted today for the given type
func (s *MemoryStore) IsAssistActionRecorded(inviterID, helperID string, assistType model.AssistType) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	for _, a := range s.assistActions {
		if a.InviterID == inviterID && a.HelperID == helperID && a.AssistType == assistType {
			if a.CreatedAt.Year() == now.Year() && a.CreatedAt.YearDay() == now.YearDay() {
				return true
			}
		}
	}
	return false
}

// RecordAssistAction records a new assist action
func (s *MemoryStore) RecordAssistAction(inviterID, helperID string, assistType model.AssistType) *model.AssistAction {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.assistIDSeq++
	action := model.AssistAction{
		ID:         "ast_" + strconv.Itoa(s.assistIDSeq),
		InviterID:  inviterID,
		HelperID:   helperID,
		AssistType: assistType,
		CreatedAt:  time.Now().UTC(),
	}
	s.assistActions = append(s.assistActions, action)
	return &action
}

// IncrementAssistProgress increments the current count by 1
func (s *MemoryStore) IncrementAssistProgress(inviterID string, assistType model.AssistType) *model.AssistProgress {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s", inviterID, assistType)
	p := s.assistProgress[key]
	p.Current++
	s.assistProgress[key] = p
	return &p
}

// ClaimAssistReward marks the assist progress as claimed
func (s *MemoryStore) ClaimAssistReward(inviterID string, assistType model.AssistType) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s", inviterID, assistType)
	p := s.assistProgress[key]
	p.Claimed = true
	s.assistProgress[key] = p
}

// GetAssistProgress returns the assist progress for the given inviter and type
func (s *MemoryStore) GetAssistProgress(inviterID string, assistType model.AssistType) *model.AssistProgress {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", inviterID, assistType)
	p, ok := s.assistProgress[key]
	if !ok {
		return nil
	}
	return &p
}

// ============================================================
// 队伍社交 (Team Social)
// ============================================================

func (s *MemoryStore) CreateTeam(captainID string, input model.CreateTeamRequest) (*model.Team, error) {
	if input.MaxMembers < 2 || input.MaxMembers > 5 {
		return nil, fmt.Errorf("max_members must be between 2 and 5")
	}
	if input.GoalDraws <= 0 {
		return nil, fmt.Errorf("goal_draws must be greater than 0")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	activeTeam := s.getUserActiveTeamLocked(captainID)
	if activeTeam != nil {
		return nil, fmt.Errorf("user already has an active team")
	}

	now := time.Now().UTC()
	team := model.Team{
		ID:           s.GenerateID(),
		CaptainID:    captainID,
		Name:         input.Name,
		MaxMembers:   input.MaxMembers,
		GoalDraws:    input.GoalDraws,
		CurrentDraws: 0,
		StartsAt:     now,
		ExpiresAt:    now.Add(48 * time.Hour),
		Status:       "recruiting",
		CreatedAt:    now,
	}

	s.teams[team.ID] = team

	member := model.TeamMember{
		TeamID:   team.ID,
		UserID:   captainID,
		Draws:    0,
		JoinedAt: now,
	}
	s.teamMembers[team.ID] = []model.TeamMember{member}
	return &team, nil
}

func (s *MemoryStore) JoinTeam(userID, teamID string) (*model.TeamMember, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.teams[teamID]
	if !ok {
		return nil, fmt.Errorf("team not found")
	}
	if team.Status != "recruiting" {
		return nil, fmt.Errorf("team is not recruiting")
	}

	members := s.teamMembers[teamID]
	if len(members) >= team.MaxMembers {
		return nil, fmt.Errorf("team is full")
	}
	for _, m := range members {
		if m.UserID == userID {
			return nil, fmt.Errorf("user is already in this team")
		}
	}

	now := time.Now().UTC()
	member := model.TeamMember{
		TeamID:   teamID,
		UserID:   userID,
		Draws:    0,
		JoinedAt: now,
	}
	s.teamMembers[teamID] = append(members, member)

	if len(s.teamMembers[teamID]) >= team.MaxMembers {
		team.Status = "active"
		s.teams[teamID] = team
	}

	return &member, nil
}

func (s *MemoryStore) LeaveTeam(userID, teamID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.teams[teamID]
	if !ok {
		return fmt.Errorf("team not found")
	}

	members := s.teamMembers[teamID]
	found := false
	var idx int
	for i, m := range members {
		if m.UserID == userID {
			found = true
			idx = i
			break
		}
	}
	if !found {
		return fmt.Errorf("user is not a member of this team")
	}

	members = append(members[:idx], members[idx+1:]...)
	if len(members) == 0 {
		delete(s.teams, teamID)
		delete(s.teamMembers, teamID)
		return nil
	}

	s.teamMembers[teamID] = members
	if team.CaptainID == userID {
		team.CaptainID = members[0].UserID
		s.teams[teamID] = team
	}

	return nil
}

func (s *MemoryStore) GetTeam(teamID string) (*model.Team, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	team, ok := s.teams[teamID]
	if !ok {
		return nil, fmt.Errorf("team not found")
	}
	return &team, nil
}

func (s *MemoryStore) GetTeamMembers(teamID string) ([]model.TeamMember, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	members, ok := s.teamMembers[teamID]
	if !ok {
		return nil, fmt.Errorf("team not found")
	}
	return members, nil
}

func (s *MemoryStore) GetUserActiveTeam(userID string) (*model.Team, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t := s.getUserActiveTeamLocked(userID)
	if t == nil {
		return nil, nil
	}
	return t, nil
}

func (s *MemoryStore) AddTeamDraw(userID, teamID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.teams[teamID]
	if !ok {
		return 0, fmt.Errorf("team not found")
	}

	members := s.teamMembers[teamID]
	found := false
	for i, m := range members {
		if m.UserID == userID {
			m.Draws++
			members[i] = m
			found = true
			break
		}
	}
	if !found {
		return 0, fmt.Errorf("user is not a member of this team")
	}

	s.teamMembers[teamID] = members
	team.CurrentDraws++
	s.teams[teamID] = team

	return team.CurrentDraws, nil
}

func (s *MemoryStore) CompleteTeam(teamID string) (*model.TeamReward, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.teams[teamID]
	if !ok {
		return nil, fmt.Errorf("team not found")
	}

	now := time.Now().UTC()
	team.Status = "completed"
	team.CompletedAt = &now
	s.teams[teamID] = team

	basePoints := team.GoalDraws * 10
	captainPoints := int(float64(basePoints) * 1.05)

	members := s.teamMembers[teamID]
	for _, m := range members {
		pts := basePoints
		if m.UserID == team.CaptainID {
			pts = captainPoints
		}
		if member, ok := s.members[m.UserID]; ok {
			member.Points += int64(pts)
			s.members[m.UserID] = member
		}
		s.pointsLog = append(s.pointsLog, model.UserPointsLog{
			ID:       s.nextTaskID,
			UserID:   m.UserID,
			Points:   int64(pts),
			Balance:  s.members[m.UserID].Points,
			Reason:   "team_reward",
			Remark:   fmt.Sprintf("组队奖励：完成 %d 次开盒", team.GoalDraws),
			CreatedAt: now,
		})
		s.nextTaskID++
	}

	reward := model.TeamReward{
		TeamID:      teamID,
		CaptainID:   team.CaptainID,
		RewardType:  "points",
		RewardQty:   captainPoints + basePoints*(len(members)-1),
		Description: fmt.Sprintf("组队完成 %d 次开盒，每位成员获得 %d 积分，队长获得 %d 积分", team.GoalDraws, basePoints, captainPoints),
	}
	s.teamRewards[teamID] = reward

	return &reward, nil
}

func (s *MemoryStore) ExpireTeam(teamID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.teams[teamID]
	if !ok {
		return fmt.Errorf("team not found")
	}
	team.Status = "expired"
	s.teams[teamID] = team
	return nil
}

func (s *MemoryStore) GetExpiredTeams() []model.Team {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	var expired []model.Team
	for _, team := range s.teams {
		if (team.Status == "recruiting" || team.Status == "active") && team.ExpiresAt.Before(now) {
			expired = append(expired, team)
		}
	}
	return expired
}
