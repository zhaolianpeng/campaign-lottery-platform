package model

import "time"

// ============================================================
// 🆕 限时商店 + 付费道具 + 首充礼包 系统
// ============================================================

// ItemType 道具类型
type ItemType string

const (
	ItemHintCard       ItemType = "hint_card"        // 提示卡：排除当前池1个款式
	ItemSeeThrough     ItemType = "see_through"      // 透卡：预览下一抽（仅普通款）
	ItemPityInherit    ItemType = "pity_inherit"     // 保底继承券：跨池保底转移
	ItemSpecifyVoucher ItemType = "specify_voucher"  // 指定款券：必得指定普通款
	ItemTenDrawTicket  ItemType = "ten_draw_ticket"  // 十连券
	ItemFreeDraw       ItemType = "free_draw"        // 免费抽券
)

// ShopItem 商店商品
type ShopItem struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	PricePoints int64      `json:"price_points"`    // 积分价格（0表示仅现金购买）
	PriceCash   int64      `json:"price_cash"`      // 现金价格（分，0表示仅积分购买）
	ItemType    ItemType   `json:"item_type"`        // 道具类型
	ItemQty     int        `json:"item_qty"`         // 数量
	Stock       int        `json:"stock"`            // 库存(-1=无限)
	DailyLimit  int        `json:"daily_limit"`      // 每日限购(0=不限)
	Category    string     `json:"category"`         // daily/weekly/festival/item
	IsActive    bool       `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	SortOrder   int        `json:"sort_order"`
}

// UserItem 用户道具库存
type UserItem struct {
	UserID   string   `json:"user_id"`
	ItemType ItemType `json:"item_type"`
	Quantity int      `json:"quantity"`
}

// UseItemRequest 使用道具请求
type UseItemRequest struct {
	ItemType   ItemType `json:"item_type"`
	CampaignID string   `json:"campaign_id,omitempty"` // 部分道具需要
	PrizeID    string   `json:"prize_id,omitempty"`    // 指定款券需要
}

// BuyShopItemRequest 购买商品请求
type BuyShopItemRequest struct {
	ShopItemID string `json:"shop_item_id"`
	Quantity   int    `json:"quantity"` // 购买数量（默认1）
}

// BuyShopItemResult 购买结果
type BuyShopItemResult struct {
	ItemType   ItemType `json:"item_type"`
	ItemName   string   `json:"item_name"`
	Quantity   int      `json:"quantity"`
	PointsCost int64    `json:"points_cost"`
	NewPoints  int64    `json:"new_points"`
	NewQty     int      `json:"new_qty"` // 购买后该道具数量
}

// ============================================================
// 首充礼包系统
// ============================================================

// FirstRechargePack 首充礼包定义
type FirstRechargePack struct {
	ID          string     `json:"id"`          // "tier_1"/"tier_2"/"tier_3"
	Name        string     `json:"name"`        // "首充6元"/"首充30元"/"首充98元"
	PricePoints int64      `json:"price_points"` // 积分价格
	CashPrice   int64      `json:"cash_price"`  // 现金价格（分）600/3000/9800
	Items       []PackItem `json:"items"`
	Description string     `json:"description"`
	SortOrder   int        `json:"sort_order"`
}

// PackItem 礼包内容项
type PackItem struct {
	Type    string `json:"type"`     // "points"/"draw_ticket"/"prize"/"month_card"/"hint_card"/"see_through"
	Name    string `json:"name"`
	Qty     int    `json:"qty"`
	PrizeID string `json:"prize_id,omitempty"` // 如果是奖品
}

// 首充礼包配置表
var FirstRechargePacks = map[string]FirstRechargePack{
	"tier_1": {
		ID: "tier_1", Name: "首充6元", PricePoints: 600, CashPrice: 600,
		Items: []PackItem{
			{Type: "points", Name: "积分", Qty: 60},
			{Type: "prize", Name: "稀有款盲盒", Qty: 1},
			{Type: "hint_card", Name: "提示卡", Qty: 3},
		},
		Description: "60积分+稀有盲盒×1+提示卡×3", SortOrder: 1,
	},
	"tier_2": {
		ID: "tier_2", Name: "首充30元", PricePoints: 3000, CashPrice: 3000,
		Items: []PackItem{
			{Type: "points", Name: "积分", Qty: 300},
			{Type: "ten_draw_ticket", Name: "十连券", Qty: 1},
			{Type: "see_through", Name: "透卡", Qty: 5},
		},
		Description: "300积分+十连券×1+透卡×5+限定头像框", SortOrder: 2,
	},
	"tier_3": {
		ID: "tier_3", Name: "首充98元", PricePoints: 9800, CashPrice: 9800,
		Items: []PackItem{
			{Type: "points", Name: "积分", Qty: 980},
			{Type: "month_card", Name: "月卡(30天)", Qty: 1},
			{Type: "ten_draw_ticket", Name: "十连券", Qty: 2},
		},
		Description: "980积分+月卡×1+十连券×2+稀有款自选券", SortOrder: 3,
	},
}

// UserFirstRecharge 用户首充记录
type UserFirstRecharge struct {
	UserID  string   `json:"user_id"`
	Claimed []string `json:"claimed"` // 已领取的礼包ID列表
}

// ClaimFirstRechargeRequest 领取首充礼包请求
type ClaimFirstRechargeRequest struct {
	PackID string `json:"pack_id"`
}

// ClaimFirstRechargeResult 领取结果
type ClaimFirstRechargeResult struct {
	PackID    string     `json:"pack_id"`
	PackName  string     `json:"pack_name"`
	Items     []PackItem `json:"items"`
	NewPoints int64      `json:"new_points"`
}

// ============================================================
// 🆕 v1.6 预约抢购 系统
// ============================================================

// FlashSale 抢购活动
type FlashSale struct {
	ID             string    `json:"id"`
	CampaignID     string    `json:"campaign_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	PricePoints    int64     `json:"price_points"`     // 抢购价格（积分）
	TotalStock     int       `json:"total_stock"`      // 总库存
	RemainingStock int       `json:"remaining_stock"`  // 剩余库存
	MinVipLevel    string    `json:"min_vip_level"`    // 最低会员等级要求
	MinTotalDraws  int64     `json:"min_total_draws"`  // 最少抽奖次数要求
	StartAt        time.Time `json:"start_at"`
	EndAt          time.Time `json:"end_at"`
	Status         string    `json:"status"`           // upcoming / active / ended
	CreatedAt      time.Time `json:"created_at"`
}

// FlashSubscription 用户抢购预约
type FlashSubscription struct {
	UserID    string    `json:"user_id"`
	FlashID   string    `json:"flash_id"`
	CreatedAt time.Time `json:"created_at"`
}

// FlashPurchaseResult 抢购结果
type FlashPurchaseResult struct {
	FlashID   string `json:"flash_id"`
	FlashName string `json:"flash_name"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
}

// FlashListInfo 抢购列表项（前端展示）
type FlashListInfo struct {
	Flash       *FlashSale `json:"flash"`
	Subscribed  bool       `json:"subscribed"`
	Purchasable bool       `json:"purchasable"` // 是否符合资格
}
