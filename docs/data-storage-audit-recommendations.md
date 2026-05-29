# 数据存储审计与修改意见

审计日期：2026-05-26

## 1. 结论摘要

当前项目已经具备 MySQL、Redis 的连接配置和健康检查，但业务数据并没有真正完成生产化持久化。现状更接近“内存业务状态 + 少量管理配置写 MySQL”的混合模型：

- MySQL 当前主要承载盲盒配置、奖品配置、商店商品、首充礼包等管理配置。
- 用户、会话、抽奖记录、每日额度、库存入账、积分流水、发奖任务、验证码、微信身份、保底状态、活动进度等核心业务状态仍保存在 `MemoryStore` 进程内存。
- Redis 当前只用于健康检查，没有参与会话、验证码、频控、幂等、分布式锁或缓存。
- 奖品图片写入本地文件目录，缺少对象存储、元数据表和备份策略。

如果当前实现用于真实运营，会存在重启丢数据、PM2 reload 状态回退、资产数据不可审计、并发抽奖无法跨进程保证一致性、MySQL 与内存双写不一致等高风险问题。建议把 MySQL 作为长期业务状态的唯一权威数据源，Redis 只承载短生命周期状态和并发辅助能力，图片等二进制资源迁移到对象存储或受管持久卷。

## 2. 审计范围

本次重点查看以下数据存储相关文件：

- `backend-server/src/server/memory-store.ts`
- `backend-server/src/server/lottery-service.ts`
- `backend-server/src/server/admin-config-repository.ts`
- `backend-server/src/server/database.ts`
- `backend-server/src/server/redis.ts`
- `backend-server/src/server/prize-image-storage.ts`
- `backend-server/src/server/singleton.ts`
- `backend-server/app/api/v1/[...path]/route.ts`
- `sql/schema.mysql.sql`
- `docs/database-design.md`
- `docs/configuration.md`
- `deploy/pm2/ecosystem.config.cjs`
- `deploy/release_82.sh`

未发现前端项目中使用 `localStorage`、`sessionStorage`、Cookie 或 IndexedDB 存储业务状态。

## 3. 当前存储链路

### 3.1 进程内存

`MemoryStore` 是当前绝大多数业务接口的事实数据源。它使用 `Map` 和数组保存用户、资料、会话、后台会话、活动配置、奖品、抽奖记录、每日额度、用户库存、交换挂单、会员积分、钱包、登录日志、状态日志、短信验证码、发奖任务、签到、分享、道具、微信身份、战令、助力、组队、赠礼、拼图、闪购、活动参与等状态。

`singleton.ts` 将 `MemoryStore` 缓存在 `globalThis` 上，同一 Node.js 进程内可以复用，但进程重启、PM2 reload、扩容到多进程或多机器后，内存状态不会共享，也不会自动恢复。

### 3.2 MySQL

`database.ts` 只负责创建连接池和 ping。`admin-config-repository.ts` 只持久化部分管理配置：

- `campaigns`
- `prizes`
- `shop_items`
- `first_recharge_packs`

服务启动时，`createMemoryStore()` 会先创建内存种子数据，然后通过 `syncAdminConfigWithMysql()` 从 MySQL 回灌这几类配置。运行时管理端创建、更新、删除配置时，路由层先改 `MemoryStore`，再调用 `upsert*` 或 `delete*Config` 写 MySQL。

这意味着 MySQL 当前不是完整业务数据源，也不是所有写入的事务边界。

### 3.3 Redis

`redis.ts` 只提供连接创建、ping 和 key 前缀函数。除健康检查外，项目内没有其他 Redis 读写。部署模板里 `REDIS_ENABLED` 仍为 `false`。

### 3.4 文件存储

`prize-image-storage.ts` 将上传图片写入 `PRIZE_UPLOAD_DIR`，未配置时写入项目根目录上级的 `.runtime/prize-images`。图片 URL 只保存为 `/api/v1/uploads/prizes/:filename`，没有独立媒体表、哈希校验、引用关系、清理策略和跨机器共享能力。

## 4. 主要问题与修改意见

### P0：核心业务状态仍是内存存储

问题：

- 用户、会话、抽奖、积分、库存、发奖、验证码、保底、活动进度等核心数据重启即丢。
- 当前 `docs/configuration.md` 已明确“业务数据仍由 `MemoryStore` 保存在服务进程内存中”，但部署模板启用了 MySQL，容易让人误以为已经完成生产化落库。
- 后台查询到的抽奖记录、用户资产、积分流水不是数据库记录，无法支撑客服追溯、财务审计、概率审计和用户申诉。

修改意见：

- 将 MySQL 定义为长期业务状态的唯一权威数据源。
- 新增 repository 层，按领域拆分为 `UserRepository`、`AuthRepository`、`CampaignRepository`、`DrawRepository`、`InventoryRepository`、`LedgerRepository`、`OperationConfigRepository`。
- `MemoryStore` 只保留单元测试或本地 demo 用途，通过接口抽象与 MySQL repository 解耦，生产环境禁止使用内存实现承载资产类写入。
- 健康检查增加“当前业务存储模式”字段。生产环境如果仍使用内存业务存储，应返回 degraded 或显式告警。

### P0：抽奖和资产变更缺少数据库事务

问题：

- `blindBoxDraw()` 中扣积分、扣每日额度、抽样、扣库存、写抽奖记录、写用户库存、创建发奖任务分散在内存对象上完成。
- 这些动作没有 MySQL 事务，没有幂等请求号约束，也没有跨进程锁或条件更新。
- `draw_records` 表有 `request_id` 字段，但当前内存 `DrawRecord` 类型和写入流程没有使用它。
- 保底状态由 `PityTracker` 保存在服务内存，重启后会重置，用户抽奖体验和概率承诺无法审计。

推荐设计：

- 为抽奖请求引入 `request_id`，由客户端传入或服务端生成并返回；数据库中建立唯一约束，重复请求直接返回首次结果。
- 抽奖写入必须在一个 MySQL 事务内完成。
- 事务内按固定顺序处理：
  - 锁定用户会员或钱包行，校验并扣减积分。
  - `INSERT ... ON DUPLICATE KEY UPDATE` 建立每日额度行，再 `SELECT ... FOR UPDATE` 锁定。
  - 读取并锁定保底状态行。
  - 抽样后对中奖奖品执行条件库存扣减：`UPDATE prizes SET stock = stock - 1 WHERE id = ? AND stock > 0`。
  - 插入 `draw_records`、`user_inventories`、`prize_fulfillment_tasks`、`user_points_logs`。
  - 更新 `user_members`、`user_campaign_quotas`、`user_pity_states`。
  - 提交事务后再返回结果。
- 十连抽可以在同一事务中逐次扣库存并写多条记录，不能先一次性生成所有结果再脱离库存状态落库。

### P1：管理配置存在内存与 MySQL 双写不一致

问题：

- 管理端创建或更新活动、奖品、商店商品、首充礼包时，路由层先调用 `service.create*()` 修改内存，再调用 `upsert*()` 写 MySQL。
- 如果 MySQL 写入失败，接口会返回失败，但内存已经被修改；直到进程重启前，用户端看到的是未持久化的脏状态。
- 删除活动先删内存，再分别删除 MySQL 中的奖品和活动，没有事务包裹。

修改意见：

- 管理配置写入改为“先数据库事务提交，后刷新缓存”。
- 配置读取可以使用 MySQL + 进程内只读缓存，但缓存必须有明确失效机制。
- 删除活动、删除奖品、更新保底配置等操作统一放入 repository 事务。
- 后台配置操作增加 `admin_operation_logs`，记录操作人、对象、前后快照、失败原因。

### P1：数据库 schema 与当前类型和实现明显漂移

问题：

- `users` 表缺少当前 `User` 类型已使用的 `phone`、`avatar_url`、`register_source`、`mobile_verified_at`、`last_login_at`、`last_login_ip`、`last_device_id` 等字段。
- `user_sessions` 表缺少 `session_type`、`revoked_at`，无法支撑受限会话、注销、冻结强制下线。
- `draw_records` 表使用 `created_at`，代码类型使用 `drawn_at`，字段命名不一致。
- `prizes` 表缺少当前类型和接口使用的 `sort_order`、`display_prob`。
- schema 只包含一部分表；代码中大量业务状态没有对应表，例如用户资料、钱包、登录日志、状态日志、验证码、微信身份、用户卡、用户道具、首充领取、战令、助力、组队、赠礼、拼图、闪购、活动参与等。

修改意见：

- 以当前 TypeScript 类型和 `docs/database-design.md` 为输入，补齐实际生产 schema。
- 为所有已上线接口的读写状态建立对应表，禁止新增“只有内存没有表”的生产功能。
- 统一时间字段命名：抽奖表建议使用 `drawn_at`，审计表使用 `created_at`，状态型表保留 `updated_at`。
- `prizes` 增加 `sort_order` 和 `display_prob`，或从接口类型中移除未持久化字段。

### P1：缺少约束、外键和数据完整性规则

问题：

- 多数状态字段是 `VARCHAR`，没有 `CHECK`、枚举表或应用层集中校验映射。
- `users.mobile` 只有普通索引，而代码层把手机号当作唯一账号。
- 核心业务表缺少外键或显式一致性约束，孤儿记录风险较高。
- 库存、积分、余额等资产字段缺少非负约束。

修改意见：

- 对手机号、微信 openid、用户道具等自然唯一关系增加唯一索引。
- 对资产字段增加非负约束，或在 MySQL 8 中使用 `CHECK`。
- 核心关系建议增加外键，至少覆盖 `user_sessions.user_id`、`prizes.campaign_id`、`draw_records.user_id`、`draw_records.campaign_id`、`user_inventories.user_id`、`user_inventories.prize_id`。
- 对高并发资产表，如抽奖记录和流水，可以保留应用层完整性校验，但必须有唯一键和必要索引兜底。

### P1：Redis 未承担短生命周期状态职责

问题：

- 短信验证码当前是固定 `123456`，明文存在内存数组。
- 微信 `session_key` 存在内存 `Map`。
- 没有请求频控、验证码 TTL、IP 或手机号限流、幂等状态缓存。

修改意见：

- Redis 用于验证码哈希、登录态热缓存、频控计数器、幂等短缓存、后台操作锁和热点配置缓存。
- 验证码只保存哈希值，key 设置 TTL，验证失败次数也设置 TTL。
- Redis 不能作为资产最终账本；积分、库存、抽奖记录、发奖任务仍以 MySQL 事务结果为准。

### P1：本地图片存储不适合生产

问题：

- 图片写入本地 `.runtime/prize-images` 或 `PRIZE_UPLOAD_DIR`，没有跨机器共享能力。
- 部署脚本不会备份该目录，也没有对象生命周期管理。
- 数据库只保存 URL 字符串，不保存文件大小、MIME、哈希、上传人、引用对象、创建时间。

修改意见：

- 生产环境使用对象存储，例如 COS、OSS、S3 或等价服务。
- 新增 `media_assets` 表，保存 `id`、`storage_provider`、`bucket`、`object_key`、`public_url`、`content_type`、`size_bytes`、`sha256`、`uploaded_by`、`created_at`。
- 奖品表保存 `image_asset_id` 或稳定 `public_url`，不要依赖单机文件路径。
- 本地文件存储仅作为开发模式，并在健康检查中标记。

### P2：迁移体系仍是 schema 脚本叠加

问题：

- `sql/schema.mysql.sql` 同时包含建库、建表和条件 `ALTER`，缺少版本号和迁移状态表。
- `release_82.sh` 使用 `mysql --force` 执行 schema，可能掩盖 DDL 错误。
- 没有回滚策略，也没有应用启动时校验 schema 版本。

修改意见：

- 引入版本化迁移目录，例如 `sql/migrations/0001_initial.sql`、`0002_user_account.sql`。
- 增加 `schema_migrations` 表记录已应用版本、校验和、执行时间。
- 发布脚本去掉 `--force`，迁移失败必须阻断发布。
- `/healthz` 增加当前 schema 版本、目标 schema 版本和迁移状态。

### P2：时间、金额和流水模型需要统一

问题：

- 代码中大量 ISO 字符串被切片写入 `DATETIME`，数据库字段没有统一时区说明。
- 积分、现金余额、累计消费混用 `INT`、`BIGINT` 和内存数字，钱包没有落库。
- 资产变更既更新余额又写流水，但当前没有数据库级账本模型保证可追溯。

修改意见：

- 数据库统一使用 UTC 时间，连接层固定 `timezone: 'Z'`，写入使用 `UTC_TIMESTAMP()` 或明确的 UTC 时间。
- 金额使用 `BIGINT` 保存最小货币单位，例如分；积分也使用 `BIGINT`。
- 钱包和积分采用“余额表 + 流水表”模型，所有余额变更必须附带一条不可变流水。
- 管理员人工调整必须记录操作人、原因、备注和前后余额。

## 5. 推荐目标架构

### 5.1 数据源职责

- MySQL：用户、账号、会话审计、活动配置、奖品、库存、抽奖记录、用户库存、积分流水、钱包、发奖任务、运营活动、审计日志等长期状态。
- Redis：验证码、频控、短期幂等缓存、热点配置缓存、后台互斥锁、可重建的临时状态。
- 对象存储：奖品图片、活动 banner、用户头像等二进制文件。
- 进程内存：只读缓存、概率引擎临时对象、测试或 demo 存储，不作为生产权威数据源。

### 5.2 推荐核心表补齐

第一批必须补齐：

- `user_profiles`
- `user_wallets`
- `wallet_transactions`
- `user_login_logs`
- `user_status_logs`
- `phone_verification_codes`
- `wechat_identities`
- `user_pity_states`
- `admin_operation_logs`
- `media_assets`

第二批随功能生产化补齐：

- `user_cards`
- `user_items`
- `user_first_recharge_claims`
- `battle_pass_*`
- `invite_records`
- `assist_progress`
- `assist_actions`
- `teams`
- `team_members`
- `gift_records`
- `puzzle_*`
- `flash_*`
- `activities`
- `activity_rewards`
- `activity_participations`

### 5.3 抽奖事务边界

抽奖服务应形成单一事务边界，至少覆盖：

- 扣积分或消耗免费券。
- 扣每日额度。
- 读取和更新保底状态。
- 扣奖品库存。
- 写抽奖记录。
- 写用户库存。
- 写积分流水。
- 创建发奖任务。
- 写活动或战令等派生进度。

事务提交失败时，用户资产不得发生任何可见变化。

## 6. 分阶段改造计划

### 第 0 阶段：生产风险封口

- 在文档和健康检查中明确当前是否仍使用 `MemoryStore` 承载业务写入。
- 生产环境禁止在 `MYSQL_ENABLED=false` 或 schema 未就绪时开放抽奖、充值、兑换、赠礼、抢购等资产接口。
- 管理端配置写入失败时，不允许保留内存脏状态。

### 第 1 阶段：迁移体系和 schema 对齐

- 引入版本化 migrations 和 `schema_migrations`。
- 补齐 `users`、`user_sessions`、`prizes` 与当前 TypeScript 类型不一致的字段。
- 建立 `user_profiles`、`wechat_identities`、`phone_verification_codes`、`user_login_logs`、`user_status_logs`。
- 为手机号、openid、会话撤销、用户状态查询补齐索引。

### 第 2 阶段：用户、认证和验证码落库

- 用户注册、手机号登录、微信登录、绑定手机号改为 MySQL repository。
- 验证码状态迁移到 Redis + MySQL 审计。
- 会话热状态可放 Redis，撤销和审计落 MySQL。
- 移除固定开发验证码在生产环境中的可用性。

### 第 3 阶段：抽奖、库存和资产事务化

- 建立 `DrawRepository` 和资产流水服务。
- 抽奖接口接入 MySQL 事务和 `request_id` 幂等。
- 库存扣减使用条件更新或行锁，防止超卖。
- 保底状态落 MySQL，必要时用 Redis 缓存但不能只存在内存。
- 增加并发测试，覆盖十连抽、库存为 1、多用户同时抽同一奖品、重复请求等场景。

### 第 4 阶段：管理配置去双写

- 后台配置写入以 MySQL 为准，内存只做缓存。
- 配置变更后刷新缓存或通过版本号懒加载。
- 管理操作写入 `admin_operation_logs`。
- 删除活动和奖品改为事务，并明确软删除策略。

### 第 5 阶段：文件和缓存生产化

- 奖品图片迁移到对象存储。
- 建立 `media_assets` 表并补齐引用关系。
- Redis 接入频控、验证码、短期幂等、热点配置缓存。
- 为 MySQL、Redis、对象存储增加备份、恢复演练和告警。

## 7. 验收标准

建议用以下标准判断改造完成：

- 服务重启、PM2 reload 后，用户、会话、积分、库存、抽奖记录、发奖任务不丢失。
- 并发抽奖测试中不会出现负库存、超每日额度、重复扣积分、重复发奖。
- 任意资产变更都能在流水表中追溯到原因、请求、操作人和前后余额。
- 管理端配置写入 MySQL 失败时，用户端不会看到未提交配置。
- `/healthz` 能反映 MySQL、Redis、schema 版本、业务存储模式和对象存储状态。
- 发布迁移失败会阻断部署，而不是继续 reload 服务。
- 通过备份恢复演练后，核心业务数据能恢复到明确时间点。

## 8. 不建议采用的方案

- 不建议简单把 `MemoryStore` 定时序列化成 JSON 文件。这只能降低重启丢失概率，不能解决并发、事务、审计、扩容和恢复问题。
- 不建议把 Redis 作为积分、库存、抽奖记录的唯一账本。Redis 可参与原子扣减和缓存，但最终账本必须在 MySQL 或等价持久数据库中。
- 不建议继续在路由层做“先改内存、再写数据库”的双写。应收敛到 repository 事务或明确的写模型。
- 不建议用一个不断追加 `ALTER` 的 `schema.mysql.sql` 长期管理生产库。应尽快迁移到版本化 migration。

## 9. 优先级建议

最高优先级是 P0 两项：

- 把核心资产和抽奖链路从内存迁到 MySQL 事务。
- 消除管理配置的内存与 MySQL 双写不一致。

这两项完成前，当前系统更适合演示和小范围内部验收，不适合承载真实用户资产和真实抽奖活动。
