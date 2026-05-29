# payment-module

独立封装的微信支付 / 支付宝模块，**不依赖主项目代码**，通过配置文件管理商户凭证与密钥。

## 能力

- 根据 `User-Agent` 自动选择支付方式：
  - **手机端**：返回跳转 URL（微信 H5 / 支付宝 WAP），或在微信内置浏览器返回 JSAPI 调起参数。
  - **电脑端**：返回扫码支付内容（微信 Native `code_url` / 支付宝 `precreate` 二维码原文）。
- 支付回调验签与订单状态推进（内置内存订单仓储，便于单独测试）。
- 查单、退款接口。
- `mock: true` 时无需真实商户号即可联调前端。

## 安装

```bash
cd payment-module
npm install
npm run build
npm test
```

## 配置

1. 复制示例配置：

```bash
cp config/payment.config.example.json config/payment.config.json
```

2. 将微信、支付宝证书放到 `payment-module/certs/`（路径在配置中引用）。
3. 填写 `notifyBaseUrl` 及各自 `appId`、商户号、密钥路径等。

配置项说明见 `config/payment.config.example.json`。

> 真实密钥与 `payment.config.json` 已加入 `payment-module/.gitignore`，请勿提交仓库。

## 快速使用

```ts
import { createPaymentModule } from '@campaign-lottery/payment-module';

const payment = createPaymentModule({
  configPath: './config/payment.config.json',
});

// 创建收银台（自动识别手机 / 电脑）
const checkout = await payment.createCheckout({
  userId: 'user_001',
  clientRequestId: 'req_202605260001',
  channel: 'wechat',
  amountCents: 990,
  subject: '100 积分包',
  businessType: 'points_pack',
  businessId: 'pack_100',
  userAgent: req.headers['user-agent'] ?? '',
  clientIp: '203.0.113.10',
  wechatOpenid: 'oXXXX', // 仅微信 JSAPI 需要
});

if (checkout.presentation === 'qrcode') {
  // 电脑端：将 checkout.qrCodeContent 生成二维码展示
}

if (checkout.presentation === 'redirect_h5') {
  // 手机端：location.href = checkout.redirectUrl
}

if (checkout.presentation === 'wechat_jsapi') {
  // 微信内：WeixinJSBridge.invoke('getBrandWCPayRequest', checkout.jsapiParams, ...)
}

// 支付回调（在业务项目自己的 HTTP 路由中调用）
const result = await payment.handlePaymentNotify('wechat', headers, rawBody);
// 返回 result.channelResponseBody 给微信

// 查单
const { order, channelPaid } = await payment.queryOrder(checkout.orderNo, {
  syncChannel: true,
});

// 退款
await payment.requestRefund({
  orderNo: checkout.orderNo,
  refundNo: 'ref_001',
  amountCents: 990,
  reason: '履约失败',
});
```

## 对外 API

| 函数 | 说明 |
| --- | --- |
| `createPaymentModule(options?)` | 加载配置并创建模块实例 |
| `detectClientPlatform(userAgent)` | 返回 `mobile` / `desktop` |
| `detectPlatformForChannel(userAgent, channel)` | 返回推荐 `presentation` |
| `createCheckout(input)` | 创建订单并返回收银台参数 |
| `handlePaymentNotify(channel, headers, body)` | 验签回调并标记已支付 |
| `queryOrder(orderNo, { syncChannel })` | 查询订单，可选渠道查单 |
| `requestRefund(input)` | 发起退款 |
| `getOrder(orderNo)` | 读取内置订单快照 |

## 呈现方式对照

| 端 | 微信 | 支付宝 |
| --- | --- | --- |
| 手机 + 微信内置浏览器 | `wechat_jsapi` | — |
| 手机 + 其他浏览器 | `redirect_h5`（H5） | `redirect_h5`（WAP） |
| 电脑 | `qrcode`（Native） | `qrcode`（precreate） |

可通过 `presentationOverride` 强制指定呈现方式。

## 与主项目关系

本目录为**独立 npm 包**，未修改 `backend-server`、`front-page` 等现有代码。接入时由业务项目自行：

1. 安装或引用 `payment-module`；
2. 在自有 API 路由中调用上述接口；
3. 在支付成功回调后执行履约逻辑（发积分、抽奖次数等）。

详细领域设计见仓库根目录 `docs/payment-module-design.md`。
