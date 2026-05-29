# TS 版部署文件

本目录保存当前线上 Next.js 双进程部署所需的正式配置模板，目标环境与 82.156.54.232 保持一致。

## 目录说明

- `pm2/ecosystem.config.cjs`：后端 `backend-server` 与前端 `front-page` 的 PM2 配置。
- `nginx/gaokao-api.conf`：82 上生效的 nginx 站点模板，负责前端入口、管理端入口和 API 反向代理。
- `release_82.sh`：将当前仓库代码与 deploy 模板同步到 82，并完成 migration、构建、PM2 reload、nginx reload 的一键发布脚本。

## 一键发布

在本机执行（**四个变量均为必填**；缺任一项脚本会在本地立即退出）：

```bash
cd deploy
DEPLOY_PASSWORD='服务器 SSH 密码' \
ADMIN_PASSWORD='后台登录密码' \
MYSQL_PASSWORD='MySQL 业务密码' \
CORS_ALLOW_ORIGIN='https://your-domain.example' \
./release_82.sh
```

| 变量 | 用途 |
| --- | --- |
| `DEPLOY_PASSWORD` | SSH 登录远程服务器 |
| `ADMIN_PASSWORD` | 后台管理登录密码（写入 PM2 `env`） |
| `MYSQL_PASSWORD` | MySQL 业务账号 `campaign_lottery_app` 密码（远程 `npm run migrate` 与 PM2 均依赖此值） |
| `CORS_ALLOW_ORIGIN` | 生产 CORS 白名单，禁止使用 `*` |

可选变量：

- `DEPLOY_HOST`：默认 `82.156.54.232`
- `DEPLOY_USER`：默认 `ubuntu`
- `REMOTE_PROJECT_DIR`：默认 `/home/ubuntu/campaign-lottery-next`（发布脚本会把模板中的路径替换为该值；若与 PM2/nginx 实际目录不一致，会跑错代码或健康检查失败）
- `NGINX_SITE_PATH`：默认 `/etc/nginx/sites-available/gaokao-api`

说明：`release_82.sh` 不会同步 `.env.local` 到服务器；migrate 与 PM2 的数据库密码均来自本机执行脚本时传入的 `MYSQL_PASSWORD`。若需在服务器上手动执行 `npm run migrate`，请在 `backend-server/.env.local` 中配置 `MYSQL_ENABLED=true` 与 `MYSQL_PASSWORD`，或在 shell 中 export 同名变量。

## HTTPS（生产推荐）

nginx 模板默认 `listen 80`。上线 HTTPS 时可用 certbot：

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d your-domain.example
sudo nginx -t && sudo systemctl reload nginx
```

外网验证：`curl -fsS https://your-domain.example/healthz` 应返回 `"status":"ok"`（`STORAGE_MODE=mysql` 且 MySQL/Redis 正常时）。

## 监控与日志

- API healthz：`GET /healthz`（含 `storage_mode`、`schema_version`）
- PM2 日志：`~/.pm2/logs/campaign-lottery-api-*.log`、`campaign-lottery-front-*.log`
- 可选：设置 `SENTRY_DSN` 环境变量接入错误追踪（后续接入）

## PM2 发布步骤

1. 将 `deploy/pm2/ecosystem.config.cjs` 复制到服务器项目根目录，例如 `/home/ubuntu/campaign-lottery-platform/ecosystem.config.cjs`。
2. 按实际环境修改以下变量：
   - `ADMIN_PASSWORD`
   - `MYSQL_PASSWORD`
   - 如有跨域要求，调整 `CORS_ALLOW_ORIGIN`
   - 支付联调：`PAYMENT_ENABLED=true`，并确保 `backend-server/config/payment.config.json` 存在（发布脚本会从 `payment.config.mock.json` 复制）
3. 在 `backend-server` 目录执行 `rm -rf .next && npm install && npm run migrate && npm run build`。
4. 在 `front-page` 目录执行 `rm -rf .next && npm install && npm run build`（必须先删 `.next`，否则可能继续提供旧登录页静态资源）。
5. 使用 PM2 启动或重载：

```bash
pm2 start ecosystem.config.cjs
pm2 save
```

或：

```bash
pm2 reload ecosystem.config.cjs --update-env
pm2 save
```

## nginx 发布步骤

1. 将 `deploy/nginx/gaokao-api.conf` 复制到 `/etc/nginx/sites-available/gaokao-api`。
2. 检查 `front-page/.next/static` 路径与项目部署目录一致。
3. 执行：

```bash
sudo nginx -t
sudo systemctl reload nginx
```

## npm 源

各子项目已配置 `.npmrc` 使用 `https://registry.npmjs.org/`。若线上 `npm install` 报 `E401`，多为 `package-lock.json` 仍指向需登录的私有源（如阿里云制品库），或服务器 `~/.npmrc` 配置了过期 token。发布脚本已显式传入 `--registry=https://registry.npmjs.org/`。

## 关键约束

- `/campaign-h5/` 必须代理到 `http://127.0.0.1:3000/`，不能把前缀原样转发给 Next，否则前台和 MIS 会 404。
- `location = /campaign-h5` 需要显式跳转到 `/campaign-h5/`，避免无斜杠访问异常。
- `/api/campaign/` 通过 rewrite 兼容旧路径，最终落到 `backend-server` 的 `/api/v1/`。
- `backend-server` 的 MySQL 连接需使用真实业务密码；如果写成其他凭据，`/healthz` 会 degraded，且任何触发 `getService()` 初始化的接口都可能失败。