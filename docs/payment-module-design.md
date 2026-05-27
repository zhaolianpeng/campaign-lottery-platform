# 支付模块设计文档

日期：2026-05-26

## 1. 背景与目标

当前平台包含活动抽奖、盲盒、商店商品、首充礼包、积分与会员等业务能力。后续接入微信支付和支付宝时，支付模块必须保证资金状态与业务权益状态一致，避免出现以下高风险问题：

- 用户已被微信或支付宝扣费，但平台业务履约失败，用户没有获得权益。
- 平台业务已经成功发放积分、抽奖次数、礼包或会员权益，但实际支付未成功。
- 微信、支付宝重复回调、用户重复点击、网络超时或服务重启导致重复发货、重复退款或订单状态错乱。

本设计目标：

- 统一接入微信支付和支付宝，屏蔽渠道差异。
- 以 MySQL 作为支付订单、交易流水、退款流水、业务履约和账本的权威数据源。
- 通过订单状态机、数据库事务、幂等约束、异步补偿和对账任务保证最终一致性。
- 保障回调验签、金额校验、密钥管理、日志脱敏、权限审计等安全要求。

## 2. 设计原则

### 2.1 支付与业务履约解耦

支付模块只负责资金相关状态：

- 创建支付订单。
- 调用微信或支付宝统一下单。
- 接收并验签支付回调。
- 查询渠道订单状态。
- 发起退款。
- 下载账单并对账。

业务履约模块负责发放权益：

- 发放积分。
- 增加抽奖次数。
- 发放首充礼包。
- 开通会员。
- 发放盲盒、道具或库存类资产。

支付模块不直接修改用户资产，业务模块也不能绕过支付成功状态直接发放付费权益。

### 2.2 支付成功以三方渠道为准

客户端支付结果只能作为展示参考，不能作为业务履约依据。

平台必须以下列信息作为支付成功依据：

- 微信支付异步通知验签通过，且交易状态为成功。
- 支付宝异步通知验签通过，且交易状态为成功。
- 主动查单确认三方交易成功。

### 2.3 业务权益以本地事务为准

发放权益必须在 MySQL 事务内完成：

- 锁定支付订单。
- 确认订单已支付且未履约。
- 写入业务履约记录。
- 写入权益流水或资金账本。
- 修改用户资产。
- 推进订单状态。

事务提交后才认为业务成功。

### 2.4 外部扣款不可回滚，只能补偿

微信和支付宝扣款发生后，本地数据库事务无法回滚外部资金动作。因此不能把“退款”理解成普通数据库 rollback。

正确机制是：

- 本地业务失败时，订单进入补偿状态。
- 补偿任务优先重试业务履约。
- 如果业务无法履约，则原路退款。
- 退款失败时进入告警和人工处理流程。

### 2.5 所有写操作必须幂等

以下行为都可能重复发生：

- 用户重复点击支付按钮。
- 客户端超时后重试创建订单。
- 微信或支付宝重复发送支付回调。
- 后台 worker 重试履约任务。
- 退款接口超时后重试。

因此所有关键写入都必须带幂等键和唯一约束。

## 3. 总体架构

```text
客户端
  |
  | 创建支付单 / 查询订单
  v
API Gateway
  |
  v
Payment Service
  |-- WeChatPay Adapter
  |-- AliPay Adapter
  |-- Payment Order Repository
  |-- Refund Repository
  |-- Reconciliation Job
  |
  | 支付成功事件
  v
Fulfillment Service
  |-- Points Fulfillment
  |-- Draw Chance Fulfillment
  |-- First Recharge Fulfillment
  |-- Membership Fulfillment
  |
  v
MySQL 权威数据源
  |
  v
Outbox / Worker / Alert
```

模块职责：

| 模块 | 职责 |
| --- | --- |
| Payment Service | 支付订单创建、渠道下单、支付回调、查单、退款、对账 |
| Channel Adapter | 封装微信支付、支付宝的签名、验签、下单、退款、查单差异 |
| Fulfillment Service | 付费业务权益发放 |
| Ledger Service | 记录资金、积分、权益流水 |
| Outbox Worker | 执行履约重试、退款补偿、对账修复 |
| Admin Console | 人工处理异常订单、退款失败、补发权益 |

## 4. 支付渠道抽象

建议定义统一支付渠道接口：

```ts
interface PaymentChannelAdapter {
  readonly channel: 'wechat' | 'alipay';

  createPayment(input: CreatePaymentInput): Promise<CreatePaymentResult>;

  verifyNotify(input: VerifyNotifyInput): Promise<VerifiedPaymentNotify>;

  queryPayment(input: QueryPaymentInput): Promise<QueryPaymentResult>;

  closePayment(input: ClosePaymentInput): Promise<ClosePaymentResult>;

  createRefund(input: CreateRefundInput): Promise<CreateRefundResult>;

  queryRefund(input: QueryRefundInput): Promise<QueryRefundResult>;
}
```

统一入参要包含：

- 平台订单号。
- 用户 ID。
- 商品快照。
- 支付金额，单位为分。
- 支付渠道。
- 支付场景，例如 H5、小程序、App、扫码。
- 回调地址。
- 客户端 IP。

渠道适配器内部处理：

- 微信支付 V3 签名、验签、证书序列号、平台证书。
- 支付宝 RSA2 签名、验签、应用 ID、卖家账号。
- 渠道订单号和平台订单号映射。
- 渠道错误码归一化。

## 5. 数据库设计

### 5.1 payment_orders

支付订单主表。

```sql
CREATE TABLE IF NOT EXISTS payment_orders (
  id BIGINT NOT NULL AUTO_INCREMENT,
  order_no VARCHAR(64) NOT NULL,
  user_id VARCHAR(32) NOT NULL,
  client_request_id VARCHAR(64) NOT NULL,
  channel VARCHAR(16) NOT NULL,
  scene VARCHAR(32) NOT NULL DEFAULT '',
  subject VARCHAR(128) NOT NULL,
  body VARCHAR(255) NOT NULL DEFAULT '',
  business_type VARCHAR(32) NOT NULL,
  business_id VARCHAR(64) NOT NULL DEFAULT '',
  product_snapshot_json JSON NOT NULL,
  amount_cents BIGINT NOT NULL,
  currency VARCHAR(8) NOT NULL DEFAULT 'CNY',
  status VARCHAR(32) NOT NULL,
  paid_at DATETIME NULL,
  fulfilled_at DATETIME NULL,
  closed_at DATETIME NULL,
  expire_at DATETIME NOT NULL,
  version BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_payment_orders_order_no (order_no),
  UNIQUE KEY uk_payment_orders_user_request (user_id, client_request_id),
  KEY idx_payment_orders_user_id (user_id),
  KEY idx_payment_orders_status_expire (status, expire_at),
  KEY idx_payment_orders_business (business_type, business_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

说明：

- `order_no` 是平台支付订单号，对外暴露。
- `client_request_id` 由客户端生成或服务端下发，用于创建订单幂等。
- `product_snapshot_json` 保存商品名称、价格、权益、版本等快照，避免商品改价影响历史订单。
- `amount_cents` 必须由服务端根据商品配置计算，不能信任客户端金额。
- `status` 使用订单状态机定义的值。

### 5.2 payment_transactions

渠道交易流水表。

```sql
CREATE TABLE IF NOT EXISTS payment_transactions (
  id BIGINT NOT NULL AUTO_INCREMENT,
  payment_order_id BIGINT NOT NULL,
  order_no VARCHAR(64) NOT NULL,
  channel VARCHAR(16) NOT NULL,
  channel_trade_no VARCHAR(128) NOT NULL,
  channel_notify_id VARCHAR(128) NOT NULL DEFAULT '',
  trade_status VARCHAR(32) NOT NULL,
  amount_cents BIGINT NOT NULL,
  currency VARCHAR(8) NOT NULL DEFAULT 'CNY',
  payer_id VARCHAR(128) NOT NULL DEFAULT '',
  raw_notify_hash VARCHAR(128) NOT NULL DEFAULT '',
  verified TINYINT(1) NOT NULL DEFAULT 0,
  paid_at DATETIME NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_payment_transactions_channel_trade (channel, channel_trade_no),
  KEY idx_payment_transactions_order_no (order_no),
  KEY idx_payment_transactions_payment_order_id (payment_order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

说明：

- 微信的 `transaction_id`、支付宝的 `trade_no` 存入 `channel_trade_no`。
- 回调原文不建议长期明文保存，可保存脱敏内容或哈希。
- `verified` 标记验签结果，验签失败的回调不得推进订单。

### 5.3 business_fulfillments

业务履约记录表。

```sql
CREATE TABLE IF NOT EXISTS business_fulfillments (
  id BIGINT NOT NULL AUTO_INCREMENT,
  payment_order_id BIGINT NOT NULL,
  order_no VARCHAR(64) NOT NULL,
  user_id VARCHAR(32) NOT NULL,
  business_type VARCHAR(32) NOT NULL,
  business_id VARCHAR(64) NOT NULL DEFAULT '',
  idempotency_key VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL,
  result_json JSON NULL,
  failure_reason VARCHAR(255) NOT NULL DEFAULT '',
  retry_count INT NOT NULL DEFAULT 0,
  next_retry_at DATETIME NULL,
  fulfilled_at DATETIME NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_business_fulfillments_order (payment_order_id),
  UNIQUE KEY uk_business_fulfillments_idempotency (idempotency_key),
  KEY idx_business_fulfillments_status_retry (status, next_retry_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

说明：

- 同一支付订单只能履约一次。
- `idempotency_key` 用于业务资产写入幂等。
- 履约失败不删除记录，必须保留失败原因和重试次数。

### 5.4 payment_refunds

退款流水表。

```sql
CREATE TABLE IF NOT EXISTS payment_refunds (
  id BIGINT NOT NULL AUTO_INCREMENT,
  refund_no VARCHAR(64) NOT NULL,
  payment_order_id BIGINT NOT NULL,
  order_no VARCHAR(64) NOT NULL,
  channel VARCHAR(16) NOT NULL,
  channel_refund_no VARCHAR(128) NOT NULL DEFAULT '',
  amount_cents BIGINT NOT NULL,
  reason VARCHAR(255) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL,
  requested_by VARCHAR(64) NOT NULL DEFAULT '',
  requested_at DATETIME NOT NULL,
  refunded_at DATETIME NULL,
  failure_reason VARCHAR(255) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_payment_refunds_refund_no (refund_no),
  KEY idx_payment_refunds_order_no (order_no),
  KEY idx_payment_refunds_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

说明：

- 退款金额累计不能超过订单实付金额。
- 自动补偿退款和人工退款都必须写入该表。
- 退款成功以渠道退款通知或主动查退款结果为准。

### 5.5 account_ledger

账本流水表。

```sql
CREATE TABLE IF NOT EXISTS account_ledger (
  id BIGINT NOT NULL AUTO_INCREMENT,
  user_id VARCHAR(32) NOT NULL,
  ledger_type VARCHAR(32) NOT NULL,
  direction VARCHAR(16) NOT NULL,
  amount BIGINT NOT NULL,
  balance_after BIGINT NULL,
  business_type VARCHAR(32) NOT NULL,
  business_id VARCHAR(64) NOT NULL,
  idempotency_key VARCHAR(128) NOT NULL,
  remark VARCHAR(255) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_account_ledger_idempotency (idempotency_key),
  KEY idx_account_ledger_user_id (user_id),
  KEY idx_account_ledger_business (business_type, business_id),
  KEY idx_account_ledger_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

说明：

- 账本只追加，不覆盖。
- `ledger_type` 可表示 `cash_payment`、`points`、`draw_chance`、`membership`、`refund` 等。
- 用户资产表中的余额或权益数量应能由账本追溯。

### 5.6 payment_outbox

异步任务表。

```sql
CREATE TABLE IF NOT EXISTS payment_outbox (
  id BIGINT NOT NULL AUTO_INCREMENT,
  topic VARCHAR(64) NOT NULL,
  aggregate_id VARCHAR(64) NOT NULL,
  payload_json JSON NOT NULL,
  status VARCHAR(32) NOT NULL,
  retry_count INT NOT NULL DEFAULT 0,
  next_retry_at DATETIME NOT NULL,
  locked_by VARCHAR(64) NOT NULL DEFAULT '',
  locked_until DATETIME NULL,
  last_error VARCHAR(255) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_payment_outbox_status_retry (status, next_retry_at),
  KEY idx_payment_outbox_topic (topic)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

说明：

- 用于履约重试、退款补偿、关单、查单、对账修复。
- outbox 记录应与订单状态变更放在同一个数据库事务内。

## 6. 订单状态机

### 6.1 支付订单状态

| 状态 | 含义 | 是否终态 |
| --- | --- | --- |
| `created` | 本地订单已创建，尚未完成渠道下单 | 否 |
| `pending` | 渠道预支付单已创建，等待用户支付 | 否 |
| `paid` | 渠道确认支付成功，尚未履约 | 否 |
| `fulfilling` | 本地正在发放业务权益 | 否 |
| `fulfilled` | 支付成功且业务履约成功 | 是 |
| `compensate_required` | 支付成功但业务履约失败，需要补偿 | 否 |
| `refund_requested` | 已发起退款 | 否 |
| `refunded` | 退款成功 | 是 |
| `refund_failed` | 退款失败，需要重试或人工处理 | 否 |
| `expired` | 用户未支付超时 | 是 |
| `closed` | 订单关闭且未扣费 | 是 |

### 6.2 状态转移

```text
created
  -> pending
  -> paid
  -> fulfilling
  -> fulfilled

pending
  -> expired
  -> closed

fulfilling
  -> compensate_required

compensate_required
  -> fulfilling
  -> refund_requested

refund_requested
  -> refunded
  -> refund_failed

refund_failed
  -> refund_requested
```

非法状态转移必须拒绝，例如：

- `pending` 不能直接变成 `fulfilled`。
- `created` 不能直接发放权益。
- `refunded` 不能再次履约。
- `fulfilled` 不能因为重复回调再次发放权益。

## 7. 核心流程

### 7.1 创建支付订单

流程：

1. 客户端选择商品和支付渠道，请求创建支付单。
2. 客户端提交 `client_request_id`。
3. 服务端读取商品配置，重新计算金额。
4. 服务端写入 `payment_orders(created)`。
5. 服务端调用微信或支付宝统一下单。
6. 渠道下单成功后，订单状态更新为 `pending`。
7. 返回客户端支付参数。

关键要求：

- 如果 `user_id + client_request_id` 已存在，直接返回原订单，不创建新订单。
- 商品名称、权益、金额、版本必须保存快照。
- 客户端金额只做展示，不能参与最终计价。
- 订单必须设置过期时间。

### 7.2 支付回调

流程：

1. 微信或支付宝向平台回调地址发送支付通知。
2. 服务端读取原始回调内容。
3. 渠道适配器验签。
4. 校验商户号、应用号、订单号、金额、币种、交易状态。
5. 在 MySQL 事务内锁定 `payment_orders`。
6. 写入 `payment_transactions`。
7. 如果订单还未处理，则推进到 `paid`。
8. 写入 `payment_outbox`，触发业务履约。
9. 返回渠道要求的成功响应。

关键要求：

- 验签失败不得更新订单。
- 金额不一致不得更新订单，必须告警。
- 重复回调只能幂等返回成功，不能重复写交易或重复履约。
- 回调处理要尽快返回，复杂业务通过 outbox worker 执行。

### 7.3 业务履约

流程：

1. Worker 读取 `payment_outbox` 的履约任务。
2. 开启 MySQL 事务。
3. `SELECT ... FOR UPDATE` 锁定支付订单。
4. 校验订单状态为 `paid` 或可重试的 `compensate_required`。
5. 校验 `business_fulfillments` 不存在成功记录。
6. 按业务类型发放权益。
7. 写入 `business_fulfillments(fulfilled)`。
8. 写入 `account_ledger`。
9. 更新 `payment_orders(fulfilled)`。
10. 提交事务。

业务类型示例：

| business_type | 履约动作 |
| --- | --- |
| `points_pack` | 增加用户积分 |
| `draw_chance_pack` | 增加抽奖次数 |
| `first_recharge_pack` | 发放首充礼包 |
| `membership` | 开通或续费会员 |
| `shop_item` | 发放商店商品或道具 |

### 7.4 主动查单

查单用于处理以下情况：

- 用户支付后前端未收到结果。
- 支付回调延迟或丢失。
- 本地订单长期停留在 `pending`。
- 对账发现本地状态与渠道状态不一致。

查单结果处理：

- 渠道成功，本地未成功：补写交易流水，推进到 `paid`，触发履约。
- 渠道未支付且已超时：关闭订单。
- 渠道订单不存在：标记异常，进入人工排查。

## 8. 失败回滚与补偿机制

### 8.1 失败场景处理

| 场景 | 资金状态 | 业务状态 | 处理方式 |
| --- | --- | --- | --- |
| 渠道下单失败 | 未扣费 | 未发放 | 订单关闭或保持 created，可重试 |
| 用户未支付超时 | 未扣费 | 未发放 | 关闭本地订单并调用渠道关单 |
| 回调验签失败 | 未确认 | 未发放 | 拒绝处理，记录安全日志 |
| 回调金额不一致 | 未确认 | 未发放 | 拒绝处理，告警 |
| 回调成功但本地写交易失败 | 可能已扣费 | 未发放 | 查单修复，补写交易 |
| 已扣费但履约失败 | 已扣费 | 未发放 | 进入 `compensate_required` |
| 履约成功但响应超时 | 已扣费 | 已发放 | 客户端查单返回成功，禁止重复发放 |
| 退款接口失败 | 已扣费 | 未发放或已撤销 | 进入 `refund_failed`，重试并告警 |

### 8.2 补偿策略

已扣费但业务失败时，不能直接丢弃异常。建议处理顺序：

1. 标记订单为 `compensate_required`。
2. 写入补偿 outbox。
3. Worker 按指数退避重试履约。
4. 如果是库存临时不足、锁冲突、数据库死锁等可恢复错误，优先重试履约。
5. 如果是商品下架、权益配置不存在、业务规则永久失败，则发起原路退款。
6. 退款失败时进入 `refund_failed`，持续重试并通知运营或财务。

### 8.3 为什么不直接在支付回调里发货

不建议在回调请求线程中直接完成复杂业务履约，原因：

- 微信和支付宝要求回调尽快返回。
- 业务发货可能涉及多个表、库存锁、积分账本、会员权益。
- 回调重试会造成重复履约风险。
- 回调线程失败时难以统一补偿。

推荐做法：

- 回调只完成验签、交易落库、订单推进、写 outbox。
- 履约由 worker 异步执行。
- 履约结果通过订单查询接口返回给前端。

## 9. 安全设计

### 9.1 回调验签

微信支付：

- 使用微信支付平台证书验签。
- 校验证书序列号。
- 校验时间戳和 nonce，降低重放风险。
- 保存验签使用的证书序列号和回调哈希。

支付宝：

- 使用支付宝公钥进行 RSA2 验签。
- 校验 `app_id`、`seller_id`、`trade_status`、`out_trade_no`、`total_amount`。

验签失败必须：

- 不更新订单。
- 不发放权益。
- 记录安全日志。
- 达到阈值后告警。

### 9.2 金额校验

必须校验：

- 回调金额等于 `payment_orders.amount_cents`。
- 回调币种等于订单币种。
- 回调订单号等于平台订单号。
- 回调商户号或应用号属于当前环境。

金额不一致属于高危事件，必须告警。

### 9.3 密钥管理

要求：

- 微信 API v3 key、商户私钥、支付宝应用私钥不得提交到代码仓库。
- 生产环境通过 KMS、密文环境变量或专用密钥管理服务注入。
- 日志不得输出私钥、签名串、完整证件号、完整手机号、完整 openid。
- 沙箱和生产商户号必须隔离。

### 9.4 接口防护

创建订单：

- 登录态校验。
- 用户状态校验，例如冻结用户不能支付。
- 商品状态、库存、限购校验。
- IP 和用户维度限流。

查询订单：

- 只能查询自己的订单。
- 管理后台查询需要权限。

退款接口：

- 只允许后台或补偿 worker 调用。
- 人工退款必须记录操作人、原因和审批信息。

### 9.5 审计日志

需要记录：

- 订单创建。
- 渠道下单。
- 支付回调验签结果。
- 订单状态变更。
- 履约成功或失败。
- 退款申请和退款结果。
- 人工补发和人工退款。
- 对账差异处理。

## 10. 对账设计

### 10.1 对账来源

- 微信支付账单。
- 支付宝账单。
- 本地 `payment_orders`。
- 本地 `payment_transactions`。
- 本地 `payment_refunds`。
- 本地 `business_fulfillments`。

### 10.2 对账规则

每日执行：

- 渠道成功，本地无交易：主动查单，补写交易并触发履约。
- 渠道成功，本地未履约：触发履约补偿。
- 渠道退款成功，本地未更新：补写退款状态。
- 本地成功，渠道无成功记录：标记异常，禁止继续履约，人工确认。
- 金额不一致：高优先级告警。

### 10.3 对账异常等级

| 等级 | 场景 | 处理 |
| --- | --- | --- |
| P0 | 金额不一致、疑似未支付发货 | 立即告警并冻结相关订单处理 |
| P1 | 渠道已扣款但本地未履约 | 自动补偿，失败后人工介入 |
| P1 | 退款失败 | 自动重试并通知财务 |
| P2 | 回调延迟导致状态短暂不一致 | 查单修复 |

## 11. API 设计建议

### 11.1 创建支付单

```http
POST /api/v1/payments/orders
```

请求：

```json
{
  "client_request_id": "req_202605260001",
  "channel": "wechat",
  "scene": "h5",
  "business_type": "points_pack",
  "business_id": "points_pack_100",
  "quantity": 1
}
```

响应：

```json
{
  "order_no": "pay_202605260001",
  "status": "pending",
  "amount_cents": 1000,
  "currency": "CNY",
  "channel": "wechat",
  "payment_params": {
    "prepay_id": "wx_prepay_id",
    "pay_sign": "..."
  },
  "expire_at": "2026-05-26T17:30:00+08:00"
}
```

### 11.2 查询支付单

```http
GET /api/v1/payments/orders/:order_no
```

响应：

```json
{
  "order_no": "pay_202605260001",
  "status": "fulfilled",
  "amount_cents": 1000,
  "paid_at": "2026-05-26T17:01:00+08:00",
  "fulfilled_at": "2026-05-26T17:01:02+08:00",
  "business_type": "points_pack",
  "fulfillment_result": {
    "points_added": 1000
  }
}
```

### 11.3 微信回调

```http
POST /api/v1/payments/wechat/notify
```

要求：

- 使用原始 body 验签。
- 验签通过后返回微信要求的成功格式。
- 失败时返回标准错误，但不得泄露内部细节。

### 11.4 支付宝回调

```http
POST /api/v1/payments/alipay/notify
```

要求：

- 使用支付宝公钥验签。
- 校验 `trade_status` 为成功态。
- 返回 `success` 文本。

### 11.5 后台退款

```http
POST /api/v1/admin/payments/orders/:order_no/refund
```

请求：

```json
{
  "amount_cents": 1000,
  "reason": "业务履约失败，原路退款"
}
```

要求：

- 需要后台权限。
- 记录操作人。
- 校验可退款金额。
- 调用渠道退款接口。
- 写入 `payment_refunds`。

## 12. 与现有系统的衔接

当前项目已有：

- `users`
- `user_members`
- `user_points_logs`
- `shop_items`
- `first_recharge_packs`
- `draw_records`
- `user_inventories`
- `prize_fulfillment_tasks`

支付模块接入时建议：

1. 不再让付费业务直接修改内存状态。
2. 商品价格以 MySQL 配置为准，例如 `shop_items.price_cash`、`first_recharge_packs.cash_price`。
3. 购买积分、首充礼包、会员等权益时，先创建 `payment_orders`。
4. 支付成功后，由履约事务更新 `user_members`、`user_points_logs` 或相关权益表。
5. 抽奖、库存、积分等资产变更必须落库，并带幂等键。

## 13. 推荐落地步骤

### 阶段一：基础账本和订单

- 新增支付相关数据表。
- 实现 `PaymentOrderRepository`。
- 实现订单创建和订单查询接口。
- 实现支付订单状态机。
- 增加唯一键和幂等处理。

### 阶段二：渠道接入

- 实现微信支付 adapter。
- 实现支付宝 adapter。
- 完成渠道下单、回调验签、主动查单。
- 增加渠道沙箱配置。

### 阶段三：业务履约

- 实现 `business_fulfillments`。
- 实现积分包、首充礼包、会员等第一批履约类型。
- 所有履约写入改为 MySQL 事务。
- 写入 `account_ledger`。

### 阶段四：补偿和退款

- 实现 `payment_outbox` worker。
- 实现履约重试。
- 实现自动退款。
- 实现退款查询和退款回调。
- 接入告警。

### 阶段五：对账和后台

- 下载微信、支付宝账单。
- 实现本地订单对账。
- 后台支持异常订单查询、人工补发、人工退款。
- 所有人工操作写审计日志。

## 14. 验收标准

功能验收：

- 微信支付成功后能正确发放权益。
- 支付宝支付成功后能正确发放权益。
- 未支付订单不会发放权益。
- 重复回调不会重复发放权益。
- 重复创建订单不会生成多笔待支付订单。
- 履约失败后能自动重试或退款。

安全验收：

- 回调验签失败不会更新订单。
- 金额不一致不会更新订单。
- 客户端篡改金额无效。
- 私钥和 API key 不进入代码仓库。
- 日志中无敏感信息明文。

一致性验收：

- 已扣费但履约失败的订单会进入补偿状态。
- 补偿任务可观测、可重试、可人工处理。
- 业务成功的订单一定存在支付成功交易流水。
- 支付成功的订单最终处于 `fulfilled` 或 `refunded`。

对账验收：

- 渠道成功但本地缺失的订单可自动修复。
- 本地成功但渠道未成功的订单会被识别并告警。
- 退款状态可以通过渠道结果修复。

## 15. 总结

支付模块不能只做“调用微信/支付宝下单接口”，它本质上是资金账本、业务权益和外部支付渠道之间的一致性系统。

本设计通过以下机制保证安全和一致性：

- 支付和履约解耦。
- 支付成功以三方验签回调或查单为准。
- 业务成功以本地 MySQL 事务提交为准。
- 所有关键写入具备唯一约束和幂等键。
- 已扣费但业务失败时进入补偿链路。
- 无法履约时原路退款。
- 通过每日对账修复回调丢失和状态漂移。

最终目标是让用户只会进入两种合理结果：

- 已扣费，并且获得对应业务权益。
- 未获得业务权益，并且未扣费或已退款。
