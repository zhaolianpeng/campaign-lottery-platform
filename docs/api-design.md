# API 设计文档

## 通用约定

所有业务接口位于 `/api/v1` 下，响应统一为：

```json
{
  "code": "ok",
  "message": "message",
  "data": {}
}
```

用户端和管理端都使用 `Authorization: Bearer <token>`。游客登录返回用户 token，管理员登录返回后台 token。

## 用户与抽盒

- `POST /auth/guest-login`：游客登录。
- `GET /me`：当前用户。
- `GET /me/draw-records`：我的抽奖记录。
- `GET /campaigns`：兼容活动列表。
- `GET /blindbox/campaigns`：盲盒系列列表，登录后带收集进度。
- `GET /blindbox/campaigns/:id/probabilities`：概率公示。
- `POST /blindbox/draw`：单抽/十连。
- `GET /blindbox/pity-status?campaign_id=`：保底状态。
- `GET /blindbox/up-pool/:campaign_id`：UP 池信息。
- `GET /blindbox/hint/:campaign_id`：摇盒提示。

## 收藏、交换与积分

- `GET /blindbox/inventory`：用户库存。
- `GET /blindbox/series-progress?campaign_id=`：系列进度。
- `POST /blindbox/blend`：重复款合成。
- `GET /blindbox/exchange-offers`：交换市场。
- `POST /blindbox/exchange-offers`：发布交换。
- `POST /blindbox/exchange-offers/:id/accept`：接受交换。
- `DELETE /blindbox/exchange-offers/:id`：取消交换。
- `GET /blindbox/member`：会员积分。
- `GET /blindbox/points-log`：积分流水。
- `POST /blindbox/checkin`：签到。
- `POST /blindbox/share-reward`：分享奖励。
- `GET /blindbox/leaderboard`：收集排行榜。
- `POST /blindbox/redeem`：积分兑换。

## 商业化

- `GET /blindbox/my-card`：旧卡/月卡兼容状态。
- `POST /blindbox/buy-card`：购买周卡/月卡/季卡。
- `GET /month-card/status`：月卡状态。
- `POST /month-card/buy`：购买月卡。
- `GET /battle-pass/info`：战令信息。
- `POST /battle-pass/buy`：购买付费战令。
- `POST /battle-pass/claim/:level`：领取战令奖励。
- `GET /shop/items`：商品列表。
- `POST /shop/buy`：购买道具。
- `GET /shop/items/inventory`：用户道具库存。
- `POST /shop/items/use`：使用道具。
- `GET /first-recharge/packs`：首充礼包。
- `GET /first-recharge/status`：首充状态。
- `POST /first-recharge/claim`：领取首充礼包。

## 社交、拼图与活动

- `POST /share/card`：创建分享卡。
- `GET /share/cards`：我的分享卡。
- `POST /share/invite`：生成邀请链接。
- `GET /share/invitees`：邀请记录。
- `GET /share/invite-stats`：邀请统计。
- `GET /share/assist-progress`：助力进度。
- `POST /share/assist`：记录助力。
- `POST /share/assist-claim`：领取助力奖励。
- `POST /team/create`：创建队伍。
- `POST /team/join`：加入队伍。
- `GET /team/my`：我的队伍。
- `POST /team/leave`：离队。
- `POST /share/gift`：赠送礼物。
- `POST /share/gift/receive`：领取礼物。
- `GET /share/gifts/incoming`：收到的礼物。
- `GET /share/gifts/sent`：送出的礼物。
- `GET /puzzle/templates`：拼图模板。
- `GET /puzzle/progress/:template_id`：拼图进度。
- `GET /puzzle/my`：我的拼图。
- `POST /puzzle/compose`：拼合领奖。
- `POST /puzzle/team/create`：创建拼图小队。
- `POST /puzzle/team/join`：加入拼图小队。
- `GET /puzzle/team/my`：我的拼图小队。
- `GET /flash/list`：抢购列表。
- `POST /flash/:id/subscribe`：预约抢购。
- `POST /flash/:id/unsubscribe`：取消预约。
- `POST /flash/:id/purchase`：执行抢购。
- `GET /flash/my`：我的预约。
- `GET /activities`：运营活动。
- `GET /activities/:id`：活动详情。
- `POST /activities/:id/join`：参与活动。
- `POST /activities/claim`：领取活动奖励。

## 管理端

- `POST /admin/login`：管理员登录。
- `GET /admin/overview`：总览。
- `GET /admin/campaigns` / `POST /admin/campaigns`：活动列表和创建。
- `GET /admin/campaigns/:id` / `PUT /admin/campaigns/:id` / `DELETE /admin/campaigns/:id`：活动详情、更新、删除。
- `GET /admin/campaigns/:id/prizes` / `POST /admin/campaigns/:id/prizes`：礼品列表和创建。
- `PUT /admin/prizes/:id` / `DELETE /admin/prizes/:id`：礼品更新、删除。
- `GET /admin/campaigns/:id/pity-config` / `PUT /admin/campaigns/:id/pity-config`：概率配置。
- `GET /admin/fulfillment-tasks`：发奖任务。
- `PATCH /admin/fulfillment-tasks/:id`：更新发奖状态。
- `POST /admin/delivery/approve`：批量审核。
- `GET /admin/draw-records`：抽奖记录。
- `GET /admin/statistics`：统计。

兼容别名：`/admin/delivery/pending` 等价于发奖任务，`/admin/lottery-logs` 等价于抽奖记录。
