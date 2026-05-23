package router

import (
	"net/http"

	"campaign-lottery-platform/backend/internal/model"
	"campaign-lottery-platform/backend/internal/response"
	"campaign-lottery-platform/backend/internal/service"
)

// ShopItemsHandler 商店商品列表
func ShopItemsHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		items, err := svc.ShopItems(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "shop items", items)
	}
}

// ShopBuyHandler 购买商店商品
func ShopBuyHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.BuyShopItemRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := svc.BuyShopItem(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "purchase success", result)
	}
}

// UserItemsHandler 用户道具列表
func UserItemsHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		items, err := svc.UserItems(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "user items", items)
	}
}

// UseItemHandler 使用道具
func UseItemHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.UseItemRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := svc.UseItem(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "item used", result)
	}
}

// FirstRechargePacksHandler 首充礼包列表
func FirstRechargePacksHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		packs := svc.FirstRechargePacks()
		result := make([]map[string]any, 0, len(packs))
		for _, p := range packs {
			result = append(result, map[string]any{
				"id": p.ID, "name": p.Name, "price_points": p.PricePoints,
				"cash_price": p.CashPrice, "items": p.Items,
				"description": p.Description, "sort_order": p.SortOrder,
			})
		}
		response.JSON(w, http.StatusOK, "ok", "first recharge packs", result)
	}
}

// FirstRechargeStatusHandler 用户首充状态
func FirstRechargeStatusHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		status, err := svc.FirstRechargeStatus(token)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "first recharge status", status)
	}
}

// ClaimFirstRechargeHandler 领取首充礼包
func ClaimFirstRechargeHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		var input model.ClaimFirstRechargeRequest
		if err := decodeJSON(r, &input); err != nil {
			response.JSON(w, http.StatusBadRequest, "bad_request", "invalid request body", nil)
			return
		}
		result, err := svc.ClaimFirstRecharge(token, input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", "first recharge claimed", result)
	}
}
