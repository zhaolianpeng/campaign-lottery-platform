# 快速启动指南

本项目是一个活动抽奖平台原型，由两个独立的 Next.js 子项目组成：

- `backend-server`：业务 API 服务，默认运行在 `http://localhost:18100`。
- `front-page`：用户端与管理端前端页面，默认运行在 `http://localhost:3000`。

当前项目使用 npm 管理依赖，两个子项目各自维护 `package-lock.json`。

## 环境要求

- Node.js 版本建议与 Next.js 16 兼容。
- npm。
- 本地端口 `3000` 和 `18100` 未被占用。

## 1. 安装依赖

在仓库根目录分别进入前后端目录安装依赖：

```bash
cd backend-server
npm install

cd ../front-page
npm install
```

如果已经存在 `node_modules`，可跳过本步骤。

## 2. 配置环境变量

分别复制环境变量示例文件：

```bash
cd backend-server
cp .env.example .env.local

cd ../front-page
cp .env.example .env.local
```

Windows PowerShell 可使用：

```powershell
Copy-Item backend-server\.env.example backend-server\.env.local
Copy-Item front-page\.env.example front-page\.env.local
```

默认配置如下：

后端 `backend-server/.env.local`：

```env
ADMIN_USER=admin
ADMIN_PASSWORD=change-me
CORS_ALLOW_ORIGIN=http://localhost:3000

MYSQL_ENABLED=false
MYSQL_DSN=
MYSQL_HOST=127.0.0.1
MYSQL_PORT=3306
MYSQL_DATABASE=campaign_lottery_platform
MYSQL_USER=campaign_lottery_app
MYSQL_PASSWORD=
MYSQL_CHARSET=utf8mb4
MYSQL_CONNECTION_LIMIT=10

REDIS_ENABLED=false
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DATABASE=10
REDIS_KEY_PREFIX=campaign:lottery:
```

前端 `front-page/.env.local`：

```env
NEXT_PUBLIC_API_BASE_URL=http://localhost:18100
NEXT_PUBLIC_ADMIN_USER_HINT=admin
```

## 3. 启动后端服务

打开一个终端：

```bash
cd backend-server
npm run dev
```

启动成功后，后端 API 服务运行在：

```text
http://localhost:18100
```

可访问健康检查：

```text
http://localhost:18100/healthz
```

启用 `MYSQL_ENABLED=true`、`REDIS_ENABLED=true` 后，健康检查会同时 ping MySQL 和 Redis，并在依赖异常时返回 `degraded`。

更多后端配置项说明见 `docs/configuration.md`。

## 4. 启动前端页面

再打开一个终端：

```bash
cd front-page
npm run dev
```

启动成功后，访问：

```text
http://localhost:3000
```

管理端页面：

```text
http://localhost:3000/admin
```

默认管理员账号来自后端环境变量：

```text
用户名：admin
密码：change-me
```

首次本地启动建议修改 `backend-server/.env.local` 中的 `ADMIN_PASSWORD`。

## 5. 本地功能验收入口

用户端 `http://localhost:3000` 登录后可在底部 Tab 验收：

- `系列`：活动横幅、系列列表、概率公示、单抽、十连、开盒结果。
- `我的`：用户库存和收藏款式。
- `交换`：发布交换、接受交换。
- `排行`：收集排行榜。
- `会员`：积分、签到、分享奖励、月卡、战令。
- `商店`：商品列表、道具库存、首充礼包。
- `社交`：邀请助力、组队开盒、礼物赠送。
- `拼图`：拼图进度、拼合领奖、限时抢购。

管理端 `http://localhost:3000/admin` 登录后可验收：

- 总览、活动、礼品、概率、发奖、记录、月卡、商店等 Tab。
- 活动和礼品支持快速创建。
- 概率页支持保存默认软/硬保底和 UP 池配置。
- 发奖页支持确认发奖任务。

## 常用命令

后端：

```bash
cd backend-server
npm run dev
npm run lint
npm run typecheck
npm test
npm run build
```

前端：

```bash
cd front-page
npm run dev
npm run lint
npm run typecheck
npm run build
```

## 启动顺序建议

1. 先启动 `backend-server`，确认 `http://localhost:18100/healthz` 可访问。
2. 再启动 `front-page`，访问 `http://localhost:3000`。
3. 如前端请求失败，优先检查 `front-page/.env.local` 中的 `NEXT_PUBLIC_API_BASE_URL` 是否指向后端地址。

## 注意事项

- 管理配置类数据已经支持通过 MySQL 持久化，包含礼品、保底配置、商店商品和首充礼包；仍依赖进程内存的数据需继续按 `docs/database-design.md` 推进。
- 当前项目以本地开发和演示为主，尚未接入容器化和 CI/CD。
- 前后端分别独立构建和启动，修改任一子项目依赖后请在对应目录重新执行 `npm install`。
- 生产部署模板见 `deploy/`，当前 82 环境使用 PM2 管理 `backend-server` 和 `front-page`，由 nginx 暴露 `/campaign-h5/`、`/campaign-admin/`、`/api/v1/` 和 `/api/campaign/`。
- 模块、API、数据库表结构和系统设计文档位于 `docs/`，其中 `docs/database-design.md` 包含 Mermaid ER 图。
