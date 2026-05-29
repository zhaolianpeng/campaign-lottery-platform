# TS 版部署文件

本目录保存当前线上 Next.js 双进程部署所需的正式配置模板，目标环境与 82.156.54.232 保持一致。

## 目录说明

- `pm2/ecosystem.config.cjs`：后端 `backend-server` 与前端 `front-page` 的 PM2 配置。
- `nginx/gaokao-api.conf`：82 上生效的 nginx 站点模板，负责前端入口、管理端入口和 API 反向代理。
- `release_82.sh`：将当前仓库代码与 deploy 模板同步到 82，并完成 migration、构建、PM2 reload、nginx reload 的一键发布脚本。

## 一键发布

在本机执行：

```bash
cd deploy
DEPLOY_PASSWORD='服务器 SSH 密码' \
ADMIN_PASSWORD='后台登录密码' \
MYSQL_PASSWORD='MySQL 业务密码' \
./release_82.sh
```

可选变量：

- `DEPLOY_HOST`：默认 `82.156.54.232`
- `DEPLOY_USER`：默认 `ubuntu`
- `REMOTE_PROJECT_DIR`：默认 `/home/ubuntu/campaign-lottery-platform`（必须与 `deploy/pm2/ecosystem.config.cjs`、`deploy/nginx/gaokao-api.conf` 中的路径一致；若写成 `campaign-lottery-next` 等其它目录，线上会继续跑旧代码）
- `NGINX_SITE_PATH`：默认 `/etc/nginx/sites-available/gaokao-api`
- `CORS_ALLOW_ORIGIN`：默认 `*`

## PM2 发布步骤

1. 将 `deploy/pm2/ecosystem.config.cjs` 复制到服务器项目根目录，例如 `/home/ubuntu/campaign-lottery-platform/ecosystem.config.cjs`。
2. 按实际环境修改以下变量：
   - `ADMIN_PASSWORD`
   - `MYSQL_PASSWORD`
   - 如有跨域要求，调整 `CORS_ALLOW_ORIGIN`
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

## 关键约束

- `/campaign-h5/` 必须代理到 `http://127.0.0.1:3000/`，不能把前缀原样转发给 Next，否则前台和 MIS 会 404。
- `location = /campaign-h5` 需要显式跳转到 `/campaign-h5/`，避免无斜杠访问异常。
- `/api/campaign/` 通过 rewrite 兼容旧路径，最终落到 `backend-server` 的 `/api/v1/`。
- `backend-server` 的 MySQL 连接需使用真实业务密码；如果写成其他凭据，`/healthz` 会 degraded，且任何触发 `getService()` 初始化的接口都可能失败。