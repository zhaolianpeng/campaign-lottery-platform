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
	default:
		response.JSON(w, http.StatusInternalServerError, "internal_error", err.Error(), nil)
	}
}
