# 支付模块接入文档

日期：2026-05-26

本文说明如何将独立包 `payment-module` 通过 **后端 API** 接入当前项目。前端只调用后端，不直接引用 `payment-module`。

## 1. 架构说明

```text
浏览器 / H5
    |  Authorization: Bearer <session>
    v
backend-server  /api/v1/payments/*
    |  createPaymentModule()
    v
payment-module（npm 本地包 file:../payment-module）
    |  HTTPS
    v
微信支付 / 支付宝
```

设计约束：

- **未修改** 现有 `app/api/v1/[...path]/route.ts` 聚合路由及抽奖等业务逻辑。
- 支付能力通过 **新增** `app/api/v1/payments/**` 路由提供。
- 默认 `PAYMENT_ENABLED=false`，不启用时现有功能完全不受影响。

## 2. 目录与新增文件

| 路径 | 说明 |
| --- | --- |
| `payment-module/` | 独立支付 SDK（微信/支付宝、扫码/H5/JSAPI） |
| `backend-server/src/server/payment-gateway.ts` | 加载配置、单例 `PaymentModule` |
| `backend-server/src/server/payment-http.ts` | 鉴权、错误映射、回调处理辅助 |
| `backend-server/app/api/v1/payments/**` | HTTP 接口 |
| `backend-server/config/payment.config.example.json` | 后端侧配置模板 |

## 3. 启用步骤

### 3.1 构建支付模块

```bash
cd payment-module
npm install
npm run build
```

### 3.2 安装后端依赖

```bash
cd backend-server
npm install
```

`package.json` 已增加：

```json
"@campaign-lottery/payment-module": "file:../payment-module"
```

### 3.3 配置文件

```bash
cd backend-server
cp config/payment.config.example.json config/payment.config.json
```

按环境填写微信/支付宝商户号、密钥路径、`notifyBaseUrl` 等。  
`notifyPath` 必须与下文 API 路径一致：

- 微信：`/api/v1/payments/wechat/notify`
- 支付宝：`/api/v1/payments/alipay/notify`

完整回调 URL = `notifyBaseUrl` + `notifyPath`，例如：

`https://api.example.com/api/v1/payments/wechat/notify`

本地 mock 联调可将 `mock` 设为 `true`（无需真实证书，适配器使用进程内临时密钥）。

### 3.4 环境变量

在 `backend-server/.env.local` 中增加：

```env
PAYMENT_ENABLED=true
PAYMENT_CONFIG_PATH=config/payment.config.json
```

未设置或 `PAYMENT_ENABLED=false` 时，除「公开探测」类接口外，支付写操作返回 `503 payment_disabled`。

## 4. HTTP API

统一响应格式与现有接口一致：

```json
{
  "code": "ok",
  "message": "...",
  "data": { }
}
```

### 4.1 公开配置

`GET /api/v1/payments/config/public`

无需登录。用于前端判断是否展示支付入口。

响应示例：

```json
{
  "code": "ok",
  "message": "payment public config",
  "data": {
    "enabled": true,
    "channels": {
      "wechat": true,
      "alipay": true
    }
  }
}
```

### 4.2 客户端平台探测

`GET /api/v1/payments/platform`

无需登录。根据 `User-Agent` 判断手机/电脑及推荐支付方式。

查询参数：

- `channel`（可选）：`wechat` | `alipay`

响应示例（无 `channel`）：

```json
{
  "code": "ok",
  "data": {
    "platform": "mobile",
    "wechat": {
      "platform": "mobile",
      "isWechatBrowser": true,
      "recommendedPresentation": "wechat_jsapi",
      "channel": "wechat"
    },
    "alipay": {
      "platform": "mobile",
      "isWechatBrowser": true,
      "recommendedPresentation": "redirect_h5",
      "channel": "alipay"
    }
  }
}
```

### 4.3 创建收银台

`POST /api/v1/payments/orders`

需要登录：`Authorization: Bearer <token>`

请求体：

```json
{
  "client_request_id": "req_202605260001",
  "channel": "wechat",
  "amount_cents": 990,
  "subject": "100积分包",
  "business_type": "points_pack",
  "business_id": "pack_100",
  "product_snapshot": { "name": "100积分包" }
}
```

| 字段 | 说明 |
| --- | --- |
| `client_request_id` | 幂等键，同一用户重复提交返回同一订单 |
| `channel` | `wechat` / `alipay` |
| `amount_cents` | 金额（分），服务端以此为准 |
| `business_type` / `business_id` | 业务方自定义，用于后续履约关联 |

服务端自动读取：

- `User-Agent` → 手机拉起 App / 电脑二维码
- `X-Forwarded-For` / `X-Real-IP` → 客户端 IP
- 当前用户微信绑定 → `openid`（JSAPI 场景）

响应按 `presentation` 区分：

**电脑扫码 `qrcode`**

```json
{
  "presentation": "qrcode",
  "channel": "wechat",
  "order_no": "pay_xxx",
  "amount_cents": 990,
  "currency": "CNY",
  "expire_at": "2026-05-26T18:00:00.000Z",
  "platform": "desktop",
  "qr_code_content": "weixin://wxpay/bizpayurl?pr=..."
}
```

前端将 `qr_code_content` 渲染为二维码即可。

**手机 H5 `redirect_h5`**

```json
{
  "presentation": "redirect_h5",
  "redirect_url": "https://..."
}
```

前端：`window.location.href = data.redirect_url`

**微信内 JSAPI `wechat_jsapi`**

```json
{
  "presentation": "wechat_jsapi",
  "jsapi_params": {
    "appId": "...",
    "timeStamp": "...",
    "nonceStr": "...",
    "package": "prepay_id=...",
    "signType": "RSA",
    "paySign": "..."
  }
}
```

前端在微信内调用 `WeixinJSBridge.invoke('getBrandWCPayRequest', jsapi_params, callback)`。

### 4.4 查询订单

`GET /api/v1/payments/orders/:orderNo?sync_channel=true`

需要登录，且只能查询本人订单。

- `sync_channel=true` 时向微信/支付宝主动查单并同步本地状态。

### 4.5 支付回调（渠道服务器调用）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/v1/payments/wechat/notify` | 微信支付异步通知 |
| POST | `/api/v1/payments/alipay/notify` | 支付宝异步通知 |

无需 Bearer Token。由 `payment-module` 验签后更新订单为 `paid`。

渠道回调仅将订单更新为 `paid`，**不会**自动发放游戏内权益（回调无用户 Bearer）。C 端应在用户完成支付后：

1. 轮询 `GET /api/v1/payments/orders/:orderNo?sync_channel=true` 直至 `status === 'paid'`
2. 调用 `POST /api/v1/payments/orders/:orderNo/fulfill` 触发业务履约（幂等）

### 4.6 业务履约

`POST /api/v1/payments/orders/:orderNo/fulfill`

需要登录，且只能履约本人订单。请求体可为空 `{}`。

| 条件 | 行为 |
| --- | --- |
| 订单已为 `fulfilled` | 返回 `already_fulfilled: true`，不重复发奖 |
| 订单为 `paid` | 按 `business_type` / `business_id` 发放权益，订单标记为 `fulfilled` |
| 订单未支付 | `402 payment_not_paid` |

当前后端已支持的 `business_type`：

| `business_type` | `business_id` 示例 | 说明 |
| --- | --- | --- |
| `first_recharge_pack` | 礼包 ID | 首充领取（不扣积分） |
| `membership` | `monthly` | 月卡（金额须为 2800 分） |
| `shop_item` | 商店道具 ID | 发放道具（不扣积分） |
| `battle_pass` | 赛季 ID（字符串） | 付费战令（金额须为 6800 分） |
| `points_pack` | 档位 ID | 按金额充值积分 |

响应示例：

```json
{
  "code": "ok",
  "data": {
    "order_no": "pay_xxx",
    "status": "fulfilled",
    "business_type": "membership",
    "already_fulfilled": false
  }
}
```

## 5. 前端调用示例

推荐使用已封装的客户端（`front-page/src/client/payment-api.ts`、`payment-checkout.ts`），**不要**在前端安装或 import `@campaign-lottery/payment-module`。

```ts
import { fetchPaymentPublicConfig, pickDefaultChannel } from '@/client/payment-api';
import { runPaymentCheckout, resumePendingPayment } from '@/client/payment-checkout';

// 是否展示现金支付入口
const config = await fetchPaymentPublicConfig();
if (!config.enabled) {
  /* 回退积分购买等旧逻辑 */
}

// 一键：建单 → 扫码/跳转/JSAPI → 轮询 paid → fulfill
await runPaymentCheckout({
  token,
  channel: pickDefaultChannel(config, 'wechat'),
  input: {
    client_request_id: `monthly_${Date.now()}`,
    channel: 'wechat',
    amount_cents: 2800,
    subject: '月卡',
    business_type: 'membership',
    business_id: 'monthly',
  },
  onQrcode: (checkout) => {
    /* 在弹层中展示 checkout.qr_code_content */
  },
});

// 从 H5 支付页返回后可恢复轮询
await resumePendingPayment(token);
```

底层 API 对应关系：

| 封装函数 | HTTP |
| --- | --- |
| `fetchPaymentPublicConfig` | `GET /payments/config/public` |
| `createPaymentCheckout` | `POST /payments/orders` |
| `queryPaymentOrder` | `GET /payments/orders/:orderNo` |
| `pollPaymentUntilPaid` | 轮询上者 + `sync_channel=true` |
| `fulfillPaymentOrder` | `POST /payments/orders/:orderNo/fulfill` |

C 端 `lottery-app.tsx` 已在 `PAYMENT_ENABLED=true` 时对首充、月卡、现金商店、战令、积分充值走上述流程；支付未启用时仍使用原积分接口。

## 6. 呈现方式对照

| 访问端 | 微信 | 支付宝 |
| --- | --- | --- |
| 手机 + 微信内置浏览器 | JSAPI 调起 | WAP 跳转 |
| 手机 + 其他浏览器 | H5 跳转 | WAP 跳转 |
| 电脑 | Native 二维码 | precreate 二维码 |

## 7. 与业务履约的衔接

推荐流程：

1. 创建订单前校验商品与金额（前端展示价须与 `amount_cents` 一致）。
2. `POST /api/v1/payments/orders` 获取收银台参数并拉起支付。
3. 轮询至 `status === 'paid'`（mock 环境可手动 POST 渠道 notify）。
4. `POST /api/v1/payments/orders/:orderNo/fulfill` 发放权益（幂等，重复调用安全）。

履约实现见 `backend-server/src/server/payment-fulfillment.ts`。抽盒仍默认扣积分；现金路径可通过 `points_pack` 充值后再抽。限时抢购仍为积分逻辑，未接支付。

领域模型与表结构见 `docs/payment-module-design.md`。

## 8. 本地联调清单

1. `payment-module` 执行 `npm run build`
2. `backend-server` 复制 `config/payment.config.json`，`mock: true`
3. `.env.local` 设置 `PAYMENT_ENABLED=true`
4. 启动后端：`npm run dev`（端口 18100）
5. 登录获取 token，调用 `POST /api/v1/payments/orders`
6. mock 回调：`POST /api/v1/payments/wechat/notify`，body：

```json
{
  "out_trade_no": "<order_no>",
  "transaction_id": "wx_mock_1",
  "amount": 990
}
```

7. `GET /api/v1/payments/orders/<order_no>` 确认 `status` 为 `paid`
8. `POST /api/v1/payments/orders/<order_no>/fulfill`（Bearer）确认权益到账、`status` 为 `fulfilled`

## 9. 生产注意事项

- 将 `mock` 设为 `false`，配置真实商户证书。
- `notifyBaseUrl` 必须为公网 HTTPS，并在微信/支付宝商户平台登记回调 URL。
- 订单当前存于 `payment-module` 内存；生产环境需按设计文档落库 MySQL。
- 密钥文件勿提交 Git；`payment.config.json` 仅保留 example。

## 10. 相关文档

- 模块 API 与配置：`payment-module/README.md`
- 领域与一致性设计：`docs/payment-module-design.md`
