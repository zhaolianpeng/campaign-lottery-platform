# 第三方认证与验证码接入说明

本文说明用户管理功能中已预留但尚未接入真实服务商的能力。当前实现保持现有架构不变，未配置第三方凭证时使用开发模式，方便本地和演示环境兼容。

## 1. 配置项

后端通过环境变量读取第三方凭证，示例见 `backend-server/.env.example`。

微信：

- `WECHAT_APP_ID`
- `WECHAT_APP_SECRET`
- `WECHAT_TOKEN`
- `WECHAT_REDIRECT_URI`

短信验证码：

- `SMS_PROVIDER`：默认 `mock`，后续可填 `aliyun`、`tencent` 等。
- `SMS_ACCESS_KEY_ID`
- `SMS_ACCESS_KEY_SECRET`
- `SMS_SIGN_NAME`
- `SMS_TEMPLATE_CODE`

运营商一键登录：

- `CARRIER_AUTH_PROVIDER`：例如闪验、阿里云号码认证等。
- `CARRIER_AUTH_APP_ID`
- `CARRIER_AUTH_API_KEY`

## 2. 当前已实现的预留接口

短信验证码：

- `POST /api/v1/auth/phone/code`
- `POST /api/v1/auth/phone/verify`

开发模式行为：

- 当 `SMS_PROVIDER=mock` 或未配置短信密钥时，后端返回 `dev_code=123456`。
- 验证码有效期为 5 分钟。
- 校验通过后，如果手机号已存在则直接登录；如果手机号不存在则创建手机号用户，保持与现有 `phone-login` 的最大兼容。

微信：

- `GET /api/v1/auth/wechat/oauth-url`
- `POST /api/v1/auth/wechat/login`
- `POST /api/v1/auth/wechat/phone`
- `GET /api/v1/auth/wechat/jssdk-config`

微信登录会在未绑定手机号时返回 `need_phone=true`，账号状态为 `pending_phone`，待手机号验证后变为 `active`。

运营商一键登录：

- 当前仅预留配置项。
- 浏览器 H5 无法可靠直接获取本机号码，生产环境需要接入服务商 SDK。
- 后续建议新增 `POST /api/v1/auth/carrier/verify`，由前端提交 SDK 返回的 token，后端调用服务商接口换取手机号。

## 3. 后续接入短信服务商需要完成的内容

1. 在后端新增短信服务适配器，例如 `backend-server/src/server/sms-service.ts`。
2. 根据 `SMS_PROVIDER` 选择服务商实现。
3. 发送验证码时调用服务商 API，而不是仅写入内存验证码。
4. 验证码入库时只保存哈希，不保存明文。
5. 增加手机号、IP、设备、openid 等维度的频控。
6. 生产化时将验证码审计落到 MySQL，将短期验证码状态写入 Redis。

## 4. 后续接入运营商一键登录需要完成的内容

1. 在前端接入服务商 H5/小程序 SDK，获取认证 token。
2. 后端新增运营商认证服务，例如 `carrier-auth-service.ts`。
3. 后端读取 `CARRIER_AUTH_PROVIDER`、`CARRIER_AUTH_APP_ID`、`CARRIER_AUTH_API_KEY`。
4. 后端用 SDK token 调服务商接口换取手机号。
5. 复用现有手机号绑定逻辑，将手机号绑定到用户主账号。
6. 对失败、取消授权、手机号冲突等情况返回明确错误码。

## 5. 生产安全要求

- 第三方密钥只能通过环境变量或密钥管理系统注入，不要提交到 Git。
- 微信 OAuth 必须配置 HTTPS 回调域名。
- 微信 `state` 参数需要校验，防止 CSRF。
- 微信 `session_key` 不应长期明文保存。
- 手机验证码只保存哈希，并设置过期时间和重试次数。
- 资产类接口必须继续依赖后端用户状态校验，不能只靠前端限制。
