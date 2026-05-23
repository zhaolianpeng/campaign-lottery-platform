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
	// 原有路由（全部保留）
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
	// 盲盒专用路由（新增）
	// ============================================================

	// 系列列表（带用户收集进度，可选token）
	mux.HandleFunc("GET /api/v1/blindbox/campaigns", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		result, err := services.CampaignListWithProgress(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "campaign list with progress", result)
	})

	// 系列概率公示详情
	mux.HandleFunc("GET /api/v1/blindbox/campaigns/{campaignID}/probabilities", func(w http.ResponseWriter, r *http.Request) {
		result, err := services.BlindBoxCampaignProbabilities(r.PathValue("campaignID"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "campaign probabilities", result)
	})

	// 盲盒抽奖（支持单抽/十连）
	mux.HandleFunc("POST /api/v1/blindbox/draw", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.DrawConfig
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := services.BlindBoxDraw(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "blind box draw completed", result)
	})

	// 购买月卡/周卡
	mux.HandleFunc("POST /api/v1/blindbox/buy-card", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.BuyCardRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := services.BuyCard(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "card purchased", result)
	})

	// 查询我的月卡
	mux.HandleFunc("GET /api/v1/blindbox/my-card", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		card, err := services.GetUserCard(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "my card", card)
	})

	// 保底状态查询
	mux.HandleFunc("GET /api/v1/blindbox/pity-status", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		campaignID := r.URL.Query().Get("campaign_id")
		if campaignID == "" {
			response.JSON(w, http.StatusBadRequest, "bad_request", "campaign_id is required", nil)
			return
		}
		status, err := services.PityStatus(token, campaignID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "pity status", status)
	})

	// 用户库存
	mux.HandleFunc("GET /api/v1/blindbox/inventory", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		items, err := services.UserInventory(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "user inventory", items)
	})

	// 系列收集进度
	mux.HandleFunc("GET /api/v1/blindbox/series-progress", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		campaignID := r.URL.Query().Get("campaign_id")
		if campaignID == "" {
			response.JSON(w, http.StatusBadRequest, "bad_request", "campaign_id is required", nil)
			return
		}
		progress, err := services.SeriesProgress(token, campaignID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "series progress", progress)
	})

	// 交换市场 - 挂单列表
	mux.HandleFunc("GET /api/v1/blindbox/exchange-offers", func(w http.ResponseWriter, _ *http.Request) {
		offers := services.ExchangeOffers()
		response.JSON(w, http.StatusOK, "ok", "exchange offers", offers)
	})

	// 交换市场 - 创建挂单
	mux.HandleFunc("POST /api/v1/blindbox/exchange-offers", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.ExchangeOfferMutation
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		offer, err := services.CreateExchangeOffer(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusCreated, "ok", "exchange offer created", offer)
	})

	// 交换市场 - 取消挂单
	mux.HandleFunc("DELETE /api/v1/blindbox/exchange-offers/{offerID}", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if err := services.CancelExchangeOffer(token, r.PathValue("offerID")); err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "exchange offer cancelled", nil)
	})

	// 交换市场 - 接受匹配
	mux.HandleFunc("POST /api/v1/blindbox/exchange-offers/{offerID}/accept", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		offer, err := services.AcceptExchangeOffer(token, r.PathValue("offerID"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "exchange offer accepted", offer)
	})

	// 会员信息
	mux.HandleFunc("GET /api/v1/blindbox/member", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		member, err := services.UserMember(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "member info", member)
	})

	// 积分记录
	mux.HandleFunc("GET /api/v1/blindbox/points-log", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		logs, err := services.PointsLog(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "points log", logs)
	})

	// 积分兑换
	mux.HandleFunc("POST /api/v1/blindbox/redeem", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.RedeemRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := services.RedeemPrize(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "redeem success", result)
	})

	// 合成系统
	mux.HandleFunc("POST /api/v1/blindbox/blend", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.BlendRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := services.BlendPrizes(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "blend success", result)
	})

	// 🆕 UP池信息查询
	mux.HandleFunc("GET /api/v1/blindbox/up-pool/{campaignID}", func(w http.ResponseWriter, r *http.Request) {
		campaignID := r.PathValue("campaignID")
		token := bearerToken(r)
		info, err := services.UPPoolInfo(token, campaignID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "up pool info", info)
	})

	// ============================================================
	// 管理端路由（保留原有）
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

	// 🆕 管理端 - 获取活动保底配置
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
	// 集卡系统新路由
	// ============================================================

	// 每日签到
	mux.HandleFunc("POST /api/v1/blindbox/checkin", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		result, err := services.DailyCheckIn(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "checkin success", result)
	})

	// 摇盒提示
	mux.HandleFunc("GET /api/v1/blindbox/hint/{campaignID}", func(w http.ResponseWriter, r *http.Request) {
		hint := services.GetCampaignHint(r.PathValue("campaignID"))
		response.JSON(w, http.StatusOK, "ok", "hint", hint)
	})

	// 分享奖励
	mux.HandleFunc("POST /api/v1/blindbox/share-reward", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		result, err := services.ShareReward(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "share reward", result)
	})

	// 收集排行榜
	mux.HandleFunc("GET /api/v1/blindbox/leaderboard", func(w http.ResponseWriter, _ *http.Request) {
		entries, err := services.GetLeaderboard(20)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "leaderboard", entries)
	})

	// 🆕 月卡系统路由
	mux.Handle("GET /api/v1/month-card/status", monthCardStatusHandler(services))
	mux.Handle("POST /api/v1/month-card/buy", buyMonthCardHandler(services))

	// 🆕 战令系统路由
	mux.Handle("GET /api/v1/battle-pass/info", battlePassInfoHandler(services))
	mux.Handle("POST /api/v1/battle-pass/buy", buyBattlePassHandler(services))
	mux.Handle("POST /api/v1/battle-pass/claim/{level}", claimBattlePassRewardHandler(services))

	// ============================================================
	// 🆕 限时商店 + 付费道具 + 首充礼包 路由
	// ============================================================

	// 商店商品列表
	mux.HandleFunc("GET /api/v1/shop/items", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		items, err := services.ShopItems(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "shop items", items)
	})

	// 购买商店商品
	mux.HandleFunc("POST /api/v1/shop/buy", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.BuyShopItemRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := services.BuyShopItem(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "purchase success", result)
	})

	// 用户道具列表
	mux.HandleFunc("GET /api/v1/shop/items/inventory", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		items, err := services.UserItems(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "user items", items)
	})

	// 使用道具
	mux.HandleFunc("POST /api/v1/shop/items/use", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.UseItemRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := services.UseItem(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "item used", result)
	})

	// 首充礼包列表（获取所有可领取的首充礼包配置）
	mux.HandleFunc("GET /api/v1/first-recharge/packs", func(w http.ResponseWriter, _ *http.Request) {
		packs := services.FirstRechargePacks()
		result := make([]map[string]any, 0, len(packs))
		for _, p := range packs {
			result = append(result, map[string]any{
				"id": p.ID, "name": p.Name, "price_points": p.PricePoints,
				"cash_price": p.CashPrice, "items": p.Items,
				"description": p.Description, "sort_order": p.SortOrder,
			})
		}
		response.JSON(w, http.StatusOK, "ok", "first recharge packs", result)
	})

	// 用户首充状态
	mux.HandleFunc("GET /api/v1/first-recharge/status", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		status, err := services.FirstRechargeStatus(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "first recharge status", status)
	})

	// 领取首充礼包
	mux.HandleFunc("POST /api/v1/first-recharge/claim", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.ClaimFirstRechargeRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := services.ClaimFirstRecharge(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "first recharge claimed", result)
	})

	// ============================================================
	// 🆕 v1.5 社交裂变路由
	// ============================================================

	// ---- 分享卡片 ----

	// 生成分享卡片
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

	// 获取我的分享卡片
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

	// 生成邀请链接
	mux.HandleFunc("POST /api/v1/share/invite", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		card, err := services.GenerateInviteLink(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "invite link", card)
	})

	// 邀请记录（查询我邀请的人）
	mux.HandleFunc("GET /api/v1/share/invitees", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		records, err := services.GetInviteRecords(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "invite records", records)
	})

	// 邀请统计
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

	// 查询助力进度
	mux.HandleFunc("GET /api/v1/share/assist-progress", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		progress, err := services.GetAssistAllProgress(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "assist progress", progress)
	})

	// 好友助力
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

	// 领取助力奖励
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

	// 创建队伍
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

	// 加入队伍
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

	// 我的队伍信息
	mux.HandleFunc("GET /api/v1/team/my", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		info, err := services.GetMyTeam(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "my team", info)
	})

	// 离开队伍
	mux.HandleFunc("POST /api/v1/team/leave", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if err := services.LeaveTeam(token); err != nil {
			response.JSON(w, http.StatusConflict, "leave_error", err.Error(), nil)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "team left", nil)
	})

	// ---- 礼物赠送 ----

	// 赠送盲盒
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

	// 接收礼物
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

	// 我的待收礼物
	mux.HandleFunc("GET /api/v1/share/gifts/incoming", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		gifts, err := services.GetMyGifts(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "incoming gifts", gifts)
	})

	// 我送出的礼物
	mux.HandleFunc("GET /api/v1/share/gifts/sent", func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		gifts, err := services.GetSentGifts(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "sent gifts", gifts)
	})

	return mux, nil
}

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

// 🆕 ---- 月卡系统路由 ----

// 查询月卡状态
func monthCardStatusHandler(svc *service.Service) http.HandlerFunc {
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

// 购买月卡
func buyMonthCardHandler(svc *service.Service) http.HandlerFunc {
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

// 🆕 ---- 战令系统路由 ----

// 查询战令信息
func battlePassInfoHandler(svc *service.Service) http.HandlerFunc {
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

// 购买付费战令
func buyBattlePassHandler(svc *service.Service) http.HandlerFunc {
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

// 领取战令奖励
func claimBattlePassRewardHandler(svc *service.Service) http.HandlerFunc {
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
