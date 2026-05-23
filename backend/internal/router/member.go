package router

import (
	"net/http"

	"campaign-lottery-platform/backend/internal/model"
	"campaign-lottery-platform/backend/internal/response"
	"campaign-lottery-platform/backend/internal/service"
)

// MemberInfoHandler 会员信息
func MemberInfoHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		member, err := svc.UserMember(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "member info", member)
	}
}

// PointsLogHandler 积分记录
func PointsLogHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		logs, err := svc.PointsLog(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "points log", logs)
	}
}

// RedeemHandler 积分兑换
func RedeemHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.RedeemRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := svc.RedeemPrize(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "redeem success", result)
	}
}

// CheckInHandler 每日签到
func CheckInHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		result, err := svc.DailyCheckIn(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "checkin success", result)
	}
}

// HintHandler 摇盒提示
func HintHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hint := svc.GetCampaignHint(r.PathValue("campaignID"))
		response.JSON(w, http.StatusOK, "ok", "hint", hint)
	}
}

// ShareRewardHandler 分享奖励
func ShareRewardHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		result, err := svc.ShareReward(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "share reward", result)
	}
}

// LeaderboardHandler 收集排行榜
func LeaderboardHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		entries, err := svc.GetLeaderboard(20)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "leaderboard", entries)
	}
}
