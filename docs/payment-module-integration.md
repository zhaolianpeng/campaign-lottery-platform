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

业务履约（发积分、发货等）应在收到 `paid` 后由业务方实现，可：

- 轮询 `GET /api/v1/payments/orders/:orderNo`
- 或在后续迭代中订阅模块内订单状态（当前为内存仓储，重启会丢失，生产请落库，见 `docs/payment-module-design.md`）

## 5. 前端调用示例

使用现有 `apiRequest`（`front-page/src/client/api.ts`）：

```ts
import { apiRequest } from '@/client/api';

// 是否展示支付
const payConfig = await apiRequest<{
  enabled: boolean;
  channels: { wechat: boolean; alipay: boolean };
}>('/api/v1/payments/config/public', '');

// 创建收银台
const checkout = await apiRequest('/api/v1/payments/orders', token, {
  method: 'POST',
  body: JSON.stringify({
    client_request_id: `req_${Date.now()}`,
    channel: 'wechat',
    amount_cents: 990,
    subject: '100积分包',
    business_type: 'points_pack',
    business_id: 'pack_100',
  }),
});

if (checkout.presentation === 'qrcode') {
  // 展示 checkout.qr_code_content 二维码
} else if (checkout.presentation === 'redirect_h5') {
  window.location.href = checkout.redirect_url;
} else if (checkout.presentation === 'wechat_jsapi') {
  // WeixinJSBridge.invoke('getBrandWCPayRequest', checkout.jsapi_params, ...)
}
```

**不要** 在前端安装或 import `@campaign-lottery/payment-module`。

## 6. 呈现方式对照

| 访问端 | 微信 | 支付宝 |
| --- | --- | --- |
| 手机 + 微信内置浏览器 | JSAPI 调起 | WAP 跳转 |
| 手机 + 其他浏览器 | H5 跳转 | WAP 跳转 |
| 电脑 | Native 二维码 | precreate 二维码 |

## 7. 与业务履约的衔接（建议）

当前接入层只负责 **收款与订单状态**，不包含积分/盲盒发货。建议业务侧：

1. 创建订单前校验商品与金额。
2. 调用 `POST /api/v1/payments/orders` 获取收银台参数。
3. 支付成功后查询订单 `status === 'paid'`。
4. 在自有服务中执行履约（MySQL 事务 + 幂等），失败则走退款（`payment-module` 的 `requestRefund`，后续可再暴露为管理端 API）。

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

## 9. 生产注意事项

- 将 `mock` 设为 `false`，配置真实商户证书。
- `notifyBaseUrl` 必须为公网 HTTPS，并在微信/支付宝商户平台登记回调 URL。
- 订单当前存于 `payment-module` 内存；生产环境需按设计文档落库 MySQL。
- 密钥文件勿提交 Git；`payment.config.json` 仅保留 example。

## 10. 相关文档

- 模块 API 与配置：`payment-module/README.md`
- 领域与一致性设计：`docs/payment-module-design.md`
