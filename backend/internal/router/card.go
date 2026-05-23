package router

import (
	"net/http"
	"strconv"

	"campaign-lottery-platform/backend/internal/model"
	"campaign-lottery-platform/backend/internal/response"
	"campaign-lottery-platform/backend/internal/service"
)

// BuyCardHandler 购买月卡/周卡（旧系统）
func BuyCardHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.BuyCardRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := svc.BuyCard(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "card purchased", result)
	}
}

// GetUserCardHandler 查询我的月卡（旧系统）
func GetUserCardHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		card, err := svc.GetUserCard(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "my card", card)
	}
}

// MonthCardStatusHandler 查询月卡状态（新系统）
func MonthCardStatusHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		status, err := svc.MonthCardStatus(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "month card status", status)
	}
}

// BuyMonthCardHandler 购买月卡（新系统）
func BuyMonthCardHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.MonthCardPurchaseRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := svc.BuyMonthCard(token, input.CardType)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "month card purchased", result)
	}
}

// BattlePassInfoHandler 查询战令信息
func BattlePassInfoHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		info, err := svc.BattlePassInfo(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "battle pass info", info)
	}
}

// BuyBattlePassHandler 购买付费战令
func BuyBattlePassHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		result, err := svc.BuyBattlePass(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "battle pass purchased", result)
	}
}

// ClaimBattlePassRewardHandler 领取战令奖励
func ClaimBattlePassRewardHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		levelStr := r.PathValue("level")
		level, _ := strconv.Atoi(levelStr)
		claimed, err := svc.ClaimBattlePassReward(token, level)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "reward claimed", map[string]bool{"claimed": claimed})
	}
}
