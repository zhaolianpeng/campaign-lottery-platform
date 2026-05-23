package router

import (
	"net/http"

	"campaign-lottery-platform/backend/internal/model"
	"campaign-lottery-platform/backend/internal/response"
	"campaign-lottery-platform/backend/internal/service"
)

// BlindBoxCampaignsHandler 系列列表（带用户收集进度）
func BlindBoxCampaignsHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		result, err := svc.CampaignListWithProgress(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "campaign list with progress", result)
	}
}

// BlindBoxCampaignProbabilitiesHandler 系列概率公示详情
func BlindBoxCampaignProbabilitiesHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := svc.BlindBoxCampaignProbabilities(r.PathValue("campaignID"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "campaign probabilities", result)
	}
}

// BlindBoxDrawHandler 盲盒抽奖（支持单抽/十连）
func BlindBoxDrawHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.DrawConfig
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := svc.BlindBoxDraw(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "blind box draw completed", result)
	}
}

// BlindBoxPityStatusHandler 保底状态查询
func BlindBoxPityStatusHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		campaignID := r.URL.Query().Get("campaign_id")
		if campaignID == "" {
			response.JSON(w, http.StatusBadRequest, "bad_request", "campaign_id is required", nil)
			return
		}
		status, err := svc.PityStatus(token, campaignID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "pity status", status)
	}
}

// BlindBoxInventoryHandler 用户库存
func BlindBoxInventoryHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		items, err := svc.UserInventory(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "user inventory", items)
	}
}

// BlindBoxSeriesProgressHandler 系列收集进度
func BlindBoxSeriesProgressHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		campaignID := r.URL.Query().Get("campaign_id")
		if campaignID == "" {
			response.JSON(w, http.StatusBadRequest, "bad_request", "campaign_id is required", nil)
			return
		}
		progress, err := svc.SeriesProgress(token, campaignID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "series progress", progress)
	}
}

// BlindBoxExchangeOffersListHandler 交换市场 - 挂单列表
func BlindBoxExchangeOffersListHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		offers := svc.ExchangeOffers()
		response.JSON(w, http.StatusOK, "ok", "exchange offers", offers)
	}
}

// BlindBoxCreateExchangeOfferHandler 交换市场 - 创建挂单
func BlindBoxCreateExchangeOfferHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.ExchangeOfferMutation
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		offer, err := svc.CreateExchangeOffer(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusCreated, "ok", "exchange offer created", offer)
	}
}

// BlindBoxCancelExchangeOfferHandler 交换市场 - 取消挂单
func BlindBoxCancelExchangeOfferHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if err := svc.CancelExchangeOffer(token, r.PathValue("offerID")); err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "exchange offer cancelled", nil)
	}
}

// BlindBoxAcceptExchangeOfferHandler 交换市场 - 接受匹配
func BlindBoxAcceptExchangeOfferHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		offer, err := svc.AcceptExchangeOffer(token, r.PathValue("offerID"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "exchange offer accepted", offer)
	}
}

// BlindBoxBlendHandler 合成系统
func BlindBoxBlendHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.BlendRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := svc.BlendPrizes(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "blend success", result)
	}
}

// BlindBoxUPPoolInfoHandler UP池信息查询
func BlindBoxUPPoolInfoHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID := r.PathValue("campaignID")
		token := bearerToken(r)
		info, err := svc.UPPoolInfo(token, campaignID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "up pool info", info)
	}
}
