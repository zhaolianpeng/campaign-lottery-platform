package router

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"campaign-lottery-platform/backend/internal/config"
	"campaign-lottery-platform/backend/internal/model"
	"campaign-lottery-platform/backend/internal/response"
	"campaign-lottery-platform/backend/internal/service"
	"campaign-lottery-platform/backend/internal/store"
)

type healthResponse struct {
	Service   string `json:"service"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type guestLoginRequest struct {
	Nickname string `json:"nickname"`
}

type drawRequest struct {
	CampaignID string `json:"campaign_id"`
}

type adminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// New creates the HTTP handler with all routes.
func New(cfg config.Config) (http.Handler, error) {
	dataStore, err := store.NewMySQLStore(cfg)
	if err != nil {
		// MySQL 不可用时使用内存存储（适合本地开发/演示）
		dataStore = store.NewMemoryStore(cfg.AdminUser, cfg.AdminPassword)
	}
	if err := dataStore.Seed(); err != nil {
		return nil, err
	}
	services := service.New(dataStore)
	mux := http.NewServeMux()

	// ============================================================
	// 基础路由
	// ============================================================

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(healthResponse{
			Service:   "campaign-lottery-api",
			Status:    "ok",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("POST /api/v1/auth/guest-login", func(w http.ResponseWriter, r *http.Request) {
		var input guestLoginRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		user, session, err := services.GuestLogin(input.Nickname)
		if err != nil {
			response.JSON(w, http.StatusInternalServerError, "login_failed", err.Error(), nil)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "guest login succeeded", map[string]any{
			"user":    user,
			"session": session,
		})
	})

	mux.HandleFunc("GET /api/v1/campaigns", func(w http.ResponseWriter, _ *http.Request) {
		campaigns := services.CampaignList()
		payload := make([]map[string]any, 0, len(campaigns))
		for _, campaign := range campaigns {
			payload = append(payload, map[string]any{
				"campaign": campaign,
				"prizes":   services.CampaignPrizeList(campaign.ID),
			})
		}
		response.JSON(w, http.StatusOK, "ok", "campaign list", payload)
	})

	mux.HandleFunc("GET /api/v1/me", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		user, err := services.UserFromToken(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "current user", user)
	})

	mux.HandleFunc("GET /api/v1/me/draw-records", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		records, err := services.UserDrawRecords(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "user draw records", records)
	})

	mux.HandleFunc("POST /api/v1/lottery/draw", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input drawRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := services.Draw(token, input.CampaignID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "draw completed", result)
	})

	// ============================================================
	// 盲盒专用路由
	// ============================================================

	mux.Handle("GET /api/v1/blindbox/campaigns", BlindBoxCampaignsHandler(services))
	mux.Handle("GET /api/v1/blindbox/campaigns/{campaignID}/probabilities", BlindBoxCampaignProbabilitiesHandler(services))
	mux.Handle("POST /api/v1/blindbox/draw", BlindBoxDrawHandler(services))
	mux.Handle("GET /api/v1/blindbox/pity-status", BlindBoxPityStatusHandler(services))
	mux.Handle("GET /api/v1/blindbox/inventory", BlindBoxInventoryHandler(services))
	mux.Handle("GET /api/v1/blindbox/series-progress", BlindBoxSeriesProgressHandler(services))
	mux.Handle("GET /api/v1/blindbox/exchange-offers", BlindBoxExchangeOffersListHandler(services))
	mux.Handle("POST /api/v1/blindbox/exchange-offers", BlindBoxCreateExchangeOfferHandler(services))
	mux.Handle("DELETE /api/v1/blindbox/exchange-offers/{offerID}", BlindBoxCancelExchangeOfferHandler(services))
	mux.Handle("POST /api/v1/blindbox/exchange-offers/{offerID}/accept", BlindBoxAcceptExchangeOfferHandler(services))
	mux.Handle("POST /api/v1/blindbox/blend", BlindBoxBlendHandler(services))
	mux.Handle("GET /api/v1/blindbox/up-pool/{campaignID}", BlindBoxUPPoolInfoHandler(services))

	// 会员 / 收集相关
	mux.Handle("GET /api/v1/blindbox/member", MemberInfoHandler(services))
	mux.Handle("GET /api/v1/blindbox/points-log", PointsLogHandler(services))
	mux.Handle("POST /api/v1/blindbox/redeem", RedeemHandler(services))
	mux.Handle("POST /api/v1/blindbox/checkin", CheckInHandler(services))
	mux.Handle("GET /api/v1/blindbox/hint/{campaignID}", HintHandler(services))
	mux.Handle("POST /api/v1/blindbox/share-reward", ShareRewardHandler(services))
	mux.Handle("GET /api/v1/blindbox/leaderboard", LeaderboardHandler(services))

	// 旧卡系统
	mux.Handle("POST /api/v1/blindbox/buy-card", BuyCardHandler(services))
	mux.Handle("GET /api/v1/blindbox/my-card", GetUserCardHandler(services))

	// ============================================================
	// 管理端路由
	// ============================================================

	mux.HandleFunc("POST /api/v1/admin/login", func(w http.ResponseWriter, r *http.Request) {
		var input adminLoginRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		token, err := services.AdminLogin(input.Username, input.Password)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "admin login succeeded", map[string]string{"token": token})
	})

	mux.HandleFunc("GET /api/v1/admin/overview", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		overview, err := services.AdminOverview(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "admin overview", overview)
	})

	mux.HandleFunc("GET /api/v1/admin/draw-records", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		records, err := services.AdminDrawRecords(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "admin draw records", records)
	})

	mux.HandleFunc("GET /api/v1/admin/campaigns", func(w http.ResponseWriter, r *http.Request) {
		items, err := services.AdminCampaigns(bearerToken(r))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "admin campaigns", items)
	})

	mux.HandleFunc("POST /api/v1/admin/campaigns", func(w http.ResponseWriter, r *http.Request) {
		var input model.CampaignMutation
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		item, err := services.CreateCampaign(bearerToken(r), input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusCreated, "ok", "campaign created", item)
	})

	mux.HandleFunc("PUT /api/v1/admin/campaigns/{campaignID}", func(w http.ResponseWriter, r *http.Request) {
		var input model.CampaignMutation
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		item, err := services.UpdateCampaign(bearerToken(r), r.PathValue("campaignID"), input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "campaign updated", item)
	})

	mux.HandleFunc("DELETE /api/v1/admin/campaigns/{campaignID}", func(w http.ResponseWriter, r *http.Request) {
		if err := services.DeleteCampaign(bearerToken(r), r.PathValue("campaignID")); err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "campaign deleted", nil)
	})

	mux.HandleFunc("GET /api/v1/admin/campaigns/{campaignID}/prizes", func(w http.ResponseWriter, r *http.Request) {
		items, err := services.AdminPrizes(bearerToken(r), r.PathValue("campaignID"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "campaign prizes", items)
	})

	mux.HandleFunc("POST /api/v1/admin/campaigns/{campaignID}/prizes", func(w http.ResponseWriter, r *http.Request) {
		var input model.PrizeMutation
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		item, err := services.CreatePrize(bearerToken(r), r.PathValue("campaignID"), input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusCreated, "ok", "prize created", item)
	})

	mux.HandleFunc("PUT /api/v1/admin/prizes/{prizeID}", func(w http.ResponseWriter, r *http.Request) {
		var input model.PrizeMutation
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		item, err := services.UpdatePrize(bearerToken(r), r.PathValue("prizeID"), input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "prize updated", item)
	})

	mux.HandleFunc("DELETE /api/v1/admin/prizes/{prizeID}", func(w http.ResponseWriter, r *http.Request) {
		if err := services.DeletePrize(bearerToken(r), r.PathValue("prizeID")); err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "prize deleted", nil)
	})

	mux.HandleFunc("GET /api/v1/admin/fulfillment-tasks", func(w http.ResponseWriter, r *http.Request) {
		items, err := services.FulfillmentTasks(bearerToken(r))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "fulfillment tasks", items)
	})

	mux.HandleFunc("PATCH /api/v1/admin/fulfillment-tasks/{taskID}", func(w http.ResponseWriter, r *http.Request) {
		var input model.FulfillmentTaskMutation
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		taskID, err := strconv.ParseInt(r.PathValue("taskID"), 10, 64)
		if err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid task id", nil)
			return
		}
		item, err := services.UpdateFulfillmentTask(bearerToken(r), taskID, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "fulfillment task updated", item)
	})

	// 管理端 - 数据统计
	mux.HandleFunc("GET /api/v1/admin/statistics", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		campaignID := r.URL.Query().Get("campaign_id")
		stats, err := services.DrawStatistics(token, campaignID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "draw statistics", stats)
	})

	// 管理端 - 更新活动保底配置
	mux.HandleFunc("PUT /api/v1/admin/campaigns/{campaignID}/pity-config", func(w http.ResponseWriter, r *http.Request) {
		var input model.PityConfig
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		campaign, err := services.AdminUpdatePityConfig(bearerToken(r), r.PathValue("campaignID"), input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "pity config updated", campaign)
	})

	// 管理端 - 获取活动保底配置
	mux.HandleFunc("GET /api/v1/admin/campaigns/{campaignID}/pity-config", func(w http.ResponseWriter, r *http.Request) {
		campaignID := r.PathValue("campaignID")
		campaign, err := services.AdminGetCampaign(bearerToken(r), campaignID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "pity config", campaign.PityConfig)
	})

	// ============================================================
	// 月卡系统路由
	// ============================================================

	mux.Handle("GET /api/v1/month-card/status", MonthCardStatusHandler(services))
	mux.Handle("POST /api/v1/month-card/buy", BuyMonthCardHandler(services))

	// ============================================================
	// 战令系统路由
	// ============================================================

	mux.Handle("GET /api/v1/battle-pass/info", BattlePassInfoHandler(services))
	mux.Handle("POST /api/v1/battle-pass/buy", BuyBattlePassHandler(services))
	mux.Handle("POST /api/v1/battle-pass/claim/{level}", ClaimBattlePassRewardHandler(services))

	// ============================================================
	// 限时商店 + 付费道具 + 首充礼包 路由
	// ============================================================

	mux.Handle("GET /api/v1/shop/items", ShopItemsHandler(services))
	mux.Handle("POST /api/v1/shop/buy", ShopBuyHandler(services))
	mux.Handle("GET /api/v1/shop/items/inventory", UserItemsHandler(services))
	mux.Handle("POST /api/v1/shop/items/use", UseItemHandler(services))

	mux.Handle("GET /api/v1/first-recharge/packs", FirstRechargePacksHandler(services))
	mux.Handle("GET /api/v1/first-recharge/status", FirstRechargeStatusHandler(services))
	mux.Handle("POST /api/v1/first-recharge/claim", ClaimFirstRechargeHandler(services))

	// ============================================================
	// v1.5 社交裂变路由
	// ============================================================

	// ---- 分享卡片 ----

	mux.HandleFunc("POST /api/v1/share/card", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input struct {
			CardType   string `json:"card_type"`
			PrizeName  string `json:"prize_name,omitempty"`
			PrizeLevel string `json:"prize_level,omitempty"`
		}
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		card, err := services.CreateShareCard(token, input.CardType, input.PrizeName, input.PrizeLevel)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "share card", card)
	})

	mux.HandleFunc("GET /api/v1/share/cards", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		cards, err := services.GetShareCards(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "my share cards", cards)
	})

	// ---- 邀请 ----

	mux.HandleFunc("POST /api/v1/share/invite", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		card, err := services.GenerateInviteLink(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "invite link", card)
	})

	mux.HandleFunc("GET /api/v1/share/invitees", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		records, err := services.GetInviteRecords(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "invite records", records)
	})

	mux.HandleFunc("GET /api/v1/share/invite-stats", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		stats, err := services.GetInviteStats(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "invite stats", stats)
	})

	// ---- 好友助力 ----

	mux.HandleFunc("GET /api/v1/share/assist-progress", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		progress, err := services.GetAssistAllProgress(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "assist progress", progress)
	})

	mux.HandleFunc("POST /api/v1/share/assist", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input struct {
			AssistType string `json:"assist_type"`
			HelperID   string `json:"helper_id"`
		}
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		progress, err := services.AssistAction(token, model.AssistType(input.AssistType), input.HelperID)
		if err != nil {
			response.JSON(w, http.StatusConflict, "assist_error", err.Error(), nil)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "assist recorded", progress)
	})

	mux.HandleFunc("POST /api/v1/share/assist-claim", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input struct {
			AssistType string `json:"assist_type"`
		}
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := services.ClaimAssistReward(token, model.AssistType(input.AssistType))
		if err != nil {
			response.JSON(w, http.StatusConflict, "claim_error", err.Error(), nil)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "assist reward claimed", result)
	})

	// ---- 组队开盒 ----

	mux.HandleFunc("POST /api/v1/team/create", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.CreateTeamRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		info, err := services.CreateTeam(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "team created", info)
	})

	mux.HandleFunc("POST /api/v1/team/join", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input struct {
			TeamID string `json:"team_id"`
		}
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		info, err := services.JoinTeam(token, input.TeamID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "team joined", info)
	})

	mux.HandleFunc("GET /api/v1/team/my", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		info, err := services.GetMyTeam(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "my team", info)
	})

	mux.HandleFunc("POST /api/v1/team/leave", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if err := services.LeaveTeam(token); err != nil {
			response.JSON(w, http.StatusConflict, "leave_error", err.Error(), nil)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "team left", nil)
	})

	// ---- 礼物赠送 ----

	mux.HandleFunc("POST /api/v1/share/gift", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.SendGiftRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		gift, err := services.SendGift(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "gift sent", gift)
	})

	mux.HandleFunc("POST /api/v1/share/gift/receive", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input struct {
			GiftID string `json:"gift_id"`
		}
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := services.ReceiveGift(token, input.GiftID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "gift received", result)
	})

	mux.HandleFunc("GET /api/v1/share/gifts/incoming", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		gifts, err := services.GetMyGifts(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "incoming gifts", gifts)
	})

	mux.HandleFunc("GET /api/v1/share/gifts/sent", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		gifts, err := services.GetSentGifts(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "sent gifts", gifts)
	})

	// ============================================================
	// 碎片拼图路由
	// ============================================================

	mux.HandleFunc("GET /api/v1/puzzle/templates", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		templates, err := services.GetPuzzleTemplates(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "puzzle templates", templates)
	})

	mux.HandleFunc("GET /api/v1/puzzle/progress/{template_id}", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		progress, err := services.GetPuzzleProgress(token, r.PathValue("template_id"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "puzzle progress", progress)
	})

	mux.HandleFunc("GET /api/v1/puzzle/my", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		progress, err := services.GetAllPuzzleProgress(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "all puzzle progress", progress)
	})

	mux.HandleFunc("POST /api/v1/puzzle/compose", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input struct {
			TemplateID string `json:"template_id"`
		}
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := services.ComposePuzzle(token, input.TemplateID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "puzzle composed", result)
	})

	mux.HandleFunc("POST /api/v1/puzzle/team/create", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input struct {
			TemplateID string `json:"template_id"`
		}
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		team, err := services.CreatePuzzleTeam(token, input.TemplateID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "puzzle team created", team)
	})

	mux.HandleFunc("POST /api/v1/puzzle/team/join", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input struct {
			TeamID string `json:"team_id"`
		}
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := services.JoinPuzzleTeam(token, input.TeamID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "joined puzzle team", result)
	})

	mux.HandleFunc("GET /api/v1/puzzle/team/my", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		teams, err := services.GetMyPuzzleTeams(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "my puzzle teams", teams)
	})

	// ============================================================
	// 预约抢购路由
	// ============================================================

	mux.HandleFunc("GET /api/v1/flash/list", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		list, err := services.GetFlashList(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "flash list", list)
	})

	mux.HandleFunc("POST /api/v1/flash/{id}/subscribe", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		result, err := services.SubscribeFlash(token, r.PathValue("id"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "subscribed", result)
	})

	mux.HandleFunc("POST /api/v1/flash/{id}/unsubscribe", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		result, err := services.UnsubscribeFlash(token, r.PathValue("id"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "unsubscribed", result)
	})

	mux.HandleFunc("POST /api/v1/flash/{id}/purchase", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		result, err := services.PurchaseFlash(token, r.PathValue("id"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "purchased", result)
	})

	mux.HandleFunc("GET /api/v1/flash/my", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		subscriptions, err := services.GetMyFlashSubscriptions(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "my flash subscriptions", subscriptions)
	})

	// ============================================================
	// Activity 路由
	// ============================================================

	mux.Handle("GET /api/v1/activities", getActivityListHandler(services))
	mux.Handle("GET /api/v1/activities/{id}", getActivityDetailHandler(services))
	mux.Handle("POST /api/v1/activities/{id}/join", joinActivityHandler(services))
	mux.Handle("POST /api/v1/activities/claim", claimActivityRewardHandler(services))

	return mux, nil
}

// ---- Activity handlers ----

func getActivityListHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		list, err := svc.GetActivityList(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "activity list", list)
	}
}

func getActivityDetailHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		id := r.PathValue("id")
		detail, err := svc.GetActivityDetail(token, id)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "activity detail", detail)
	}
}

func joinActivityHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		id := r.PathValue("id")
		participation, err := svc.JoinActivity(token, id)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "joined", participation)
	}
}

func claimActivityRewardHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var req model.ClaimActivityRewardRequest
		if err := decodeJSON(r, &req); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		reward, err := svc.ClaimActivityReward(token, req.ActivityID, req.RewardID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "reward claimed", reward)
	}
}

// ---- Shared helpers ----

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	reader := io.LimitReader(r.Body, 1<<20)
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func bearerToken(r *http.Request) string {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return ""
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrUnauthorized):
		response.JSON(w, http.StatusUnauthorized, "unauthorized", err.Error(), nil)
	case errors.Is(err, store.ErrAdminUnauthorized):
		response.JSON(w, http.StatusUnauthorized, "admin_unauthorized", err.Error(), nil)
	case errors.Is(err, store.ErrBadAdminAuth):
		response.JSON(w, http.StatusUnauthorized, "bad_admin_credentials", err.Error(), nil)
	case errors.Is(err, store.ErrCampaignNotFound):
		response.JSON(w, http.StatusNotFound, "campaign_not_found", err.Error(), nil)
	case errors.Is(err, store.ErrCampaignInactive):
		response.JSON(w, http.StatusConflict, "campaign_inactive", err.Error(), nil)
	case errors.Is(err, store.ErrNoDrawChances):
		response.JSON(w, http.StatusConflict, "no_draw_chances", err.Error(), nil)
	case errors.Is(err, store.ErrInsufficientPoints):
		response.JSON(w, http.StatusConflict, "insufficient_points", err.Error(), nil)
	case errors.Is(err, store.ErrShareLimitReached):
		response.JSON(w, http.StatusConflict, "share_limit_reached", err.Error(), nil)
	case errors.Is(err, store.ErrNoActiveSeason):
		response.JSON(w, http.StatusNotFound, "no_active_season", err.Error(), nil)
	case errors.Is(err, store.ErrAlreadyPurchased):
		response.JSON(w, http.StatusConflict, "already_purchased", err.Error(), nil)
	case errors.Is(err, store.ErrNotEligible):
		response.JSON(w, http.StatusConflict, "not_eligible", err.Error(), nil)
	default:
		response.JSON(w, http.StatusInternalServerError, "internal_error", err.Error(), nil)
	}
}
