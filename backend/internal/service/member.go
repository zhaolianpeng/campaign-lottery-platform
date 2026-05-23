package service

import (
	"campaign-lottery-platform/backend/internal/model"
	"campaign-lottery-platform/backend/internal/store"
)

// ============================================================
// 用户库存 / 收集进度
// ============================================================

// UserInventory 返回用户的库存列表
func (s *Service) UserInventory(token string) ([]model.UserInventory, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetUserInventory(user.ID)
}

// SeriesProgress 返回用户在某个系列的收集进度
func (s *Service) SeriesProgress(token string, campaignID string) (*model.SeriesProgress, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	campaign, err := s.store.GetCampaign(campaignID)
	if err != nil {
		return nil, err
	}
	return s.store.GetSeriesProgress(user.ID, campaignID, campaign.Name)
}

// ============================================================
// 积分/会员
// ============================================================

func (s *Service) UserMember(token string) (*model.UserMember, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetUserMember(user.ID)
}

func (s *Service) PointsLog(token string) ([]model.UserPointsLog, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetPointsLog(user.ID)
}

func (s *Service) RedeemPrize(token string, input model.RedeemRequest) (*model.RedeemResult, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.RedeemPrize(user.ID, input)
}

// BlendPrizes 合成：消耗多个重复款式，合成更高级款式
func (s *Service) BlendPrizes(token string, input model.BlendRequest) (*model.BlendResult, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.BlendPrizes(user.ID, input.SourcePrizeID, input.CampaignID)
}

// ============================================================
// 每日签到 / 分享 / 排行榜
// ============================================================

// DailyCheckIn 每日签到
func (s *Service) DailyCheckIn(token string) (*model.CheckInResult, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	result, err := s.store.DailyCheckIn(user.ID, 5)
	if err != nil {
		return nil, err
	}
	if result.StreakDays >= 7 {
		bonusResult, err := s.store.DailyCheckIn(user.ID, 20)
		if err != nil {
			return nil, err
		}
		bonusResult.PointsAwarded = result.PointsAwarded + 20
		bonusResult.IsBonus = true
		member, _ := s.store.GetUserMember(user.ID)
		s.store.LogPoints(user.ID, bonusResult.PointsAwarded, member.Points, "daily", "每日签到")
		return bonusResult, nil
	}
	member, _ := s.store.GetUserMember(user.ID)
	s.store.LogPoints(user.ID, result.PointsAwarded, member.Points, "daily", "每日签到")
	return result, nil
}

// ShareReward 分享奖励
func (s *Service) ShareReward(token string) (*model.ShareRewardResult, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	count, err := s.store.GetShareDailyCount(user.ID)
	if err != nil {
		return nil, err
	}
	if count >= 10 {
		return nil, store.ErrShareLimitReached
	}
	result, err := s.store.ShareReward(user.ID, 2)
	if err != nil {
		return nil, err
	}
	member, _ := s.store.GetUserMember(user.ID)
	s.store.LogPoints(user.ID, 2, member.Points, "share", "分享奖励")
	return result, nil
}

// GetLeaderboard 获取收集排行榜
func (s *Service) GetLeaderboard(limit int) ([]model.LeaderboardEntry, error) {
	return s.store.GetLeaderboard(limit)
}

// GetCampaignHint 获取盲盒摇盒提示文案
func (s *Service) GetCampaignHint(campaignID string) *model.HintMessage {
	return s.store.GetCampaignHint(campaignID)
}
