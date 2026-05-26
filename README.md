# campaign-lottery-platform

当前仓库的主实现已经切换为 TypeScript + Next.js 双进程架构：

- `backend-server`：基于 Next.js Route Handler 的业务 API，默认端口 `18100`
- `front-page`：用户端与 MIS 管理端前端，默认端口 `3000`

Go 版 `backend/` 与旧 `frontend/` 目录仍保留在仓库中，主要用于历史实现与数据结构参考，不再是当前 82 环境的生产入口。

## 当前架构

- 前端：Next.js App Router、React 19、TypeScript、TanStack Query、React Hook Form、Zod
- 后端：Next.js Route Handler、TypeScript、Zod、mysql2、redis
- 部署：PM2 管理 `campaign-lottery-api` 与 `campaign-lottery-front`，nginx 暴露 `/campaign-h5/`、`/campaign-admin/`、`/api/v1/` 与 `/api/campaign/`
- 持久化：管理配置类数据已支持落 MySQL，包含礼品、保底配置、商店商品、首充礼包

## 目录说明

```text
campaign-lottery-platform/
	backend-server/              # 当前后端服务（Next.js, port 18100）
	front-page/                  # 当前前端与 MIS（Next.js, port 3000）
	deploy/                      # PM2、nginx 与 82 一键发布脚本
	docs/                        # 技术文档
	sql/                         # MySQL schema
	backend/                     # 历史 Go 版实现（非当前生产入口）
```

## 本地开发

后端：

```bash
cd backend-server
cp .env.example .env.local
npm install
npm run dev
```

前端：

```bash
cd front-page
cp .env.example .env.local
npm install
npm run dev
```

默认访问：

- 用户端：`http://localhost:3000`
- 管理端：`http://localhost:3000/admin`
- 后端健康检查：`http://localhost:18100/healthz`

更完整的本地启动说明见 `QUICKSTART.md`。

## 生产部署

正式部署文件已经整理到 `deploy/`：

- `deploy/pm2/ecosystem.config.cjs`
- `deploy/nginx/gaokao-api.conf`
- `deploy/release_82.sh`

82 环境可直接使用一键脚本发布：

```bash
cd deploy
DEPLOY_PASSWORD='服务器 SSH 密码' \
ADMIN_PASSWORD='后台登录密码' \
MYSQL_PASSWORD='MySQL 业务密码' \
./release_82.sh
```

脚本会完成：

1. 同步 `backend-server`、`front-page`、`sql/` 到服务器
2. 用仓库模板替换远端 `ecosystem.config.cjs` 和 nginx 站点配置
3. 执行 schema
4. 构建前后端
5. reload PM2 与 nginx，并做基础健康检查

## API 概览

用户端主要入口：

- `POST /api/v1/auth/guest-login`
- `POST /api/v1/auth/phone-login`
- `POST /api/v1/auth/wechat/login`
- `GET /api/v1/campaigns`
- `POST /api/v1/blindbox/draw`
- `GET /api/v1/shop/items`
- `GET /api/v1/first-recharge/packs`

管理端主要入口：

- `POST /api/v1/admin/login`
- `GET /api/v1/admin/overview`
- `GET/POST /api/v1/admin/campaigns`
- `GET/POST /api/v1/admin/campaigns/:id/prizes`
- `PUT/DELETE /api/v1/admin/prizes/:id`
- `PUT /api/v1/admin/campaigns/:id/pity-config`
- `GET/POST /api/v1/admin/shop-items`
- `PUT/DELETE /api/v1/admin/shop-items/:id`
- `GET/POST /api/v1/admin/first-recharge/packs`
- `PUT/DELETE /api/v1/admin/first-recharge/packs/:id`

## 现状说明

- 82 上当前生产形态已经是 TS 前后端 + PM2 + nginx，不再是 Go 二进制 + 静态页面。
- 管理配置类数据已经落 MySQL，但用户态、会话态与更多业务数据仍有一部分在进程内存中。
- 如果需要继续推进生产化，优先级仍然是补齐剩余持久化、补自动化发布、补更系统的回归测试。
