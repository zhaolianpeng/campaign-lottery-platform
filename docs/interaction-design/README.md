# BOX·MAGIC 交互设计文档索引

> 盲盒抽奖平台功能模块交互设计（IXD）  
> 编写规范见 [00-conventions.md](./00-conventions.md)

## 阅读顺序

1. [编写规范](./00-conventions.md)
2. 横切：[C 端壳层](./cross-cutting/navigation-shell.md)、[支付](./cross-cutting/payment-checkout.md)
3. C 端模块 `c-end/01` → `10`
4. 管理端模块 `admin/01` → `08`

## 能力标注图例

| 标注 | 含义 |
|------|------|
| **[已实现]** | 当前代码可复现 |
| **[部分实现]** | 主流程有，体验/风控不完整 |
| **[规划中]** | 产品文档有，代码未落地 |

## 文档地图

### 横切

| 文档 | 说明 |
|------|------|
| [cross-cutting/navigation-shell.md](./cross-cutting/navigation-shell.md) | C 端 Tab、详情栈、顶栏、功能开关 |
| [cross-cutting/payment-checkout.md](./cross-cutting/payment-checkout.md) | 积分充值、现金购、二维码/H5/JSAPI |

### C 端（`/` → `LotteryApp`）

| 文档 | Tab/入口 | 实现状态概览 |
|------|----------|----------------|
| [c-end/01-auth-login.md](./c-end/01-auth-login.md) | 登录门 | 游客/试玩/微信/手机 **[已实现]** |
| [c-end/02-series-activities-draw.md](./c-end/02-series-activities-draw.md) | `series` + 盲盒详情 | 抽盒/开盒弹窗 **[已实现]**；AR/摇盒 **[规划中]** |
| [c-end/03-inventory.md](./c-end/03-inventory.md) | `inventory` | 收藏网格 **[已实现]** |
| [c-end/04-exchange.md](./c-end/04-exchange.md) | `exchange` | 发布/接受 **[已实现]**；匹配推荐 **[规划中]** |
| [c-end/05-leaderboard.md](./c-end/05-leaderboard.md) | `rank` | 排行榜 **[已实现]** |
| [c-end/06-member-points.md](./c-end/06-member-points.md) | 顶栏 + `member` | 签到/流水/分享 **[已实现]** |
| [c-end/07-shop-first-recharge.md](./c-end/07-shop-first-recharge.md) | `shop` | 商店/首充 **[已实现]** |
| [c-end/08-monthcard-battlepass.md](./c-end/08-monthcard-battlepass.md) | `member` 内 | 月卡/战令 **[部分实现]** |
| [c-end/09-social.md](./c-end/09-social.md) | `social` | 邀请/组队/赠礼 **[部分实现]**（演示级） |
| [c-end/10-puzzle-flash.md](./c-end/10-puzzle-flash.md) | `puzzle` | 拼图/秒杀 **[部分实现]** |

### 管理端（`/admin` → `AdminApp`）

| 文档 | AdminTab | 实现状态概览 |
|------|----------|----------------|
| [admin/01-auth-shell.md](./admin/01-auth-shell.md) | 登录 + 壳层 | **[已实现]** |
| [admin/02-overview-records.md](./admin/02-overview-records.md) | `overview` / `records` | **[已实现]** |
| [admin/03-users.md](./admin/03-users.md) | `users` | **[已实现]** |
| [admin/04-campaign-prize-pity.md](./admin/04-campaign-prize-pity.md) | `campaigns` / `prizes` / `pity` | **[已实现]** |
| [admin/05-fulfillment.md](./admin/05-fulfillment.md) | `delivery` | **[已实现]** |
| [admin/06-shop-first-recharge.md](./admin/06-shop-first-recharge.md) | `shop` | **[已实现]** |
| [admin/07-feature-toggles.md](./admin/07-feature-toggles.md) | `features` | **[已实现]** |
| [admin/08-monthcard-battlepass-view.md](./admin/08-monthcard-battlepass-view.md) | `monthcard` | 只读 **[部分实现]** |

## 产品对照汇总

- [APPENDIX-product-gap-matrix.md](./APPENDIX-product-gap-matrix.md) — 产品文档第 4–9 章与当前实现对照表（排期/验收用）

## 关联设计文档

| 文档 | 用途 |
|------|------|
| [产品功能设计文档.md](../../产品功能设计文档.md) | 能力愿景与差异对照 |
| [modules.md](../modules.md) | 模块边界 |
| [api-design.md](../api-design.md) | 接口契约 |
| [user-management-user-design.md](../user-management-user-design.md) | 用户状态与登录规则 |
| [blind-box-management-design.md](../blind-box-management-design.md) | 盲盒/奖品/履约业务 |
| [payment-module-integration.md](../payment-module-integration.md) | 支付接入 |

## 验收清单

- [x] 18 篇 IXD + 本索引与规范
- [x] 每篇含信息架构图与主流程/状态说明
- [x] `TabKey` / `AdminTab` 与文档一一对应
- [x] 产品文档第 4–9 章能力在差异表中标注实现状态
- [x] 可凭文档复现：游客登录 → 单抽 → 管理端新建盲盒（见各篇流程编号）
