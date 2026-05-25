package service

import (
	"fmt"

	"campaign-lottery-platform/backend/internal/model"
)

// ============================================================
// 商店 + 道具 + 首充礼包
// ============================================================

// ShopItems 获取商店商品列表
func (s *Service) ShopItems(token string) ([]model.ShopItem, error) {
	if _, err := s.store.UserFromToken(token); err != nil {
		return nil, err
	}
	return s.store.GetShopItems(), nil
}

// BuyShopItem 购买商店商品
func (s *Service) BuyShopItem(token string, input model.BuyShopItemRequest) (*model.BuyShopItemResult, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	result, err := s.store.BuyShopItem(user.ID, input.ShopItemID, input.Quantity)
	if err != nil {
		return nil, err
	}
	// 记录积分变动
	member, _ := s.store.GetUserMember(user.ID)
	s.store.LogPoints(user.ID, -result.PointsCost, member.Points, "shop", fmt.Sprintf("购买 %s x%d", result.ItemName, result.Quantity))
	return result, nil
}

// UserItems 获取用户所有道具
func (s *Service) UserItems(token string) ([]model.UserItem, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetUserItems(user.ID)
}

// UseItem 使用道具
func (s *Service) UseItem(token string, input model.UseItemRequest) (map[string]any, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}

	switch input.ItemType {
	case model.ItemHintCard:
		// 提示卡：排除当前池1个不想要的款式
		ok, err := s.store.UseUserItem(user.ID, model.ItemHintCard, 1)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("insufficient hint cards")
		}
		return map[string]any{
			"success": true, "message": "使用提示卡成功，排除了一个款式",
			"excluded_prize_id": input.PrizeID,
		}, nil

	case model.ItemSeeThrough:
		// 透卡：预览下一抽（仅普通款）
		ok, err := s.store.UseUserItem(user.ID, model.ItemSeeThrough, 1)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("insufficient see-through cards")
		}
		// 随机返回一个普通款作为预览
		if input.CampaignID != "" {
			prizes := s.store.PrizeList(input.CampaignID)
			for _, p := range prizes {
				if p.Level == model.PrizeLevelCommon && p.Status == "active" {
					return map[string]any{
						"success": true, "message": "预览成功",
						"preview_prize": map[string]any{
							"id": p.ID, "name": p.Name, "level": p.Level,
						},
					}, nil
				}
			}
		}
		return map[string]any{"success": true, "message": "预览完成，仅显示普通款"}, nil

	case model.ItemTenDrawTicket:
		// 十连券：直接在抽奖中使用
		ok, err := s.store.UseUserItem(user.ID, model.ItemTenDrawTicket, 1)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("insufficient ten-draw tickets")
		}
		return map[string]any{
			"success": true, "message": "使用十连券成功，可进行一次免费十连抽",
		}, nil

	default:
		return nil, fmt.Errorf("unsupported item type: %s", input.ItemType)
	}
}

// ============================================================
// 首充礼包
// ============================================================

// FirstRechargeStatus 获取首充状态（可领取的礼包列表）
func (s *Service) FirstRechargeStatus(token string) (*model.UserFirstRecharge, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.GetFirstRechargeStatus(user.ID)
}

// FirstRechargePacks 获取所有首充礼包配置
func (s *Service) FirstRechargePacks() map[string]model.FirstRechargePack {
	return model.FirstRechargePacks
}

// ClaimFirstRecharge 领取首充礼包
func (s *Service) ClaimFirstRecharge(token string, input model.ClaimFirstRechargeRequest) (*model.ClaimFirstRechargeResult, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return nil, err
	}
	return s.store.ClaimFirstRecharge(user.ID, input.PackID)
}
