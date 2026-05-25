package store

import (
	"fmt"
	"strconv"
	"time"

	"campaign-lottery-platform/backend/internal/model"
)

// ============================================================
// 🆕 活动系统 MemoryStore 实现
// ============================================================

// GetActiveActivities 获取所有进行中的活动（时间范围内且状态为active）
func (s *MemoryStore) GetActiveActivities() ([]model.Activity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	var active []model.Activity
	for _, a := range s.activities {
		if a.Status == "active" && !now.Before(a.StartAt) && !now.After(a.EndAt) {
			active = append(active, a)
		}
	}
	return active, nil
}

// GetAllActivities 返回所有活动
func (s *MemoryStore) GetAllActivities() ([]model.Activity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.Activity, len(s.activities))
	copy(result, s.activities)
	return result, nil
}

// GetActivity 根据ID获取活动
func (s *MemoryStore) GetActivity(activityID string) (*model.Activity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, a := range s.activities {
		if a.ID == activityID {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("activity not found: %s", activityID)
}

// CreateActivity 创建活动
func (s *MemoryStore) CreateActivity(input model.ActivityCreateRequest) (*model.Activity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.activityIDSeq++
	now := time.Now().UTC()
	act := &model.Activity{
		ID:          "act_" + strconv.Itoa(s.activityIDSeq),
		Name:        input.Name,
		Description: input.Description,
		Type:        input.Type,
		BannerURL:   input.BannerURL,
		Rules:       input.Rules,
		SortOrder:   input.SortOrder,
		Status:      "draft",
		StartAt:     input.StartAt,
		EndAt:       input.EndAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.activities = append(s.activities, *act)
	return act, nil
}

// UpdateActivity 更新活动
func (s *MemoryStore) UpdateActivity(activityID string, input model.ActivityUpdateRequest) (*model.Activity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.activities {
		if s.activities[i].ID == activityID {
			if input.Name != "" {
				s.activities[i].Name = input.Name
			}
			if input.Description != "" {
				s.activities[i].Description = input.Description
			}
			if input.BannerURL != "" {
				s.activities[i].BannerURL = input.BannerURL
			}
			if input.Rules != nil {
				s.activities[i].Rules = *input.Rules
			}
			if input.SortOrder != nil {
				s.activities[i].SortOrder = *input.SortOrder
			}
			if input.Status != "" {
				s.activities[i].Status = input.Status
			}
			if input.StartAt != nil {
				s.activities[i].StartAt = *input.StartAt
			}
			if input.EndAt != nil {
				s.activities[i].EndAt = *input.EndAt
			}
			s.activities[i].UpdatedAt = time.Now().UTC()
			return &s.activities[i], nil
		}
	}
	return nil, fmt.Errorf("activity not found: %s", activityID)
}

// DeleteActivity 删除活动
func (s *MemoryStore) DeleteActivity(activityID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.activities {
		if s.activities[i].ID == activityID {
			s.activities = append(s.activities[:i], s.activities[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("activity not found: %s", activityID)
}

// GetActivityRewards 获取活动的奖励列表
func (s *MemoryStore) GetActivityRewards(activityID string) ([]model.ActivityReward, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rewards []model.ActivityReward
	for _, r := range s.activityRewards {
		if r.ActivityID == activityID {
			rewards = append(rewards, r)
		}
	}
	return rewards, nil
}

// JoinActivity 用户参与活动
func (s *MemoryStore) JoinActivity(userID, activityID string) (*model.ActivityParticipation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	part := &model.ActivityParticipation{
		ID:            s.GenerateID(),
		UserID:        userID,
		ActivityID:    activityID,
		RewardClaimed: false,
		JoinedAt:      now,
	}
	s.activityParticipations = append(s.activityParticipations, *part)
	return part, nil
}

// GetUserActivityParticipation 获取用户在特定活动的参与记录
func (s *MemoryStore) GetUserActivityParticipation(userID, activityID string) (*model.ActivityParticipation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.activityParticipations {
		if p.UserID == userID && p.ActivityID == activityID {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("activity participation not found: user=%s, activity=%s", userID, activityID)
}

// GetUserActivityParticipations 获取用户的所有活动参与记录
func (s *MemoryStore) GetUserActivityParticipations(userID string) ([]model.ActivityParticipation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var parts []model.ActivityParticipation
	for _, p := range s.activityParticipations {
		if p.UserID == userID {
			parts = append(parts, p)
		}
	}
	return parts, nil
}

// ClaimActivityReward 领取活动奖励
func (s *MemoryStore) ClaimActivityReward(userID, activityID, rewardID string) (*model.ActivityReward, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var part *model.ActivityParticipation
	for i := range s.activityParticipations {
		if s.activityParticipations[i].UserID == userID && s.activityParticipations[i].ActivityID == activityID {
			part = &s.activityParticipations[i]
			break
		}
	}
	if part == nil {
		return nil, fmt.Errorf("activity participation not found: user=%s, activity=%s", userID, activityID)
	}

	var reward *model.ActivityReward
	for i := range s.activityRewards {
		if s.activityRewards[i].ID == rewardID && s.activityRewards[i].ActivityID == activityID {
			reward = &s.activityRewards[i]
			break
		}
	}
	if reward == nil {
		return nil, fmt.Errorf("activity reward not found: %s", rewardID)
	}

	part.RewardClaimed = true

	if reward.RewardType == "points" {
		member, exists := s.members[userID]
		if !exists {
			return nil, fmt.Errorf("member not found: %s", userID)
		}
		member.Points += int64(reward.RewardQty)
		member.UpdatedAt = time.Now().UTC()
		s.members[userID] = member
	}

	return reward, nil
}
