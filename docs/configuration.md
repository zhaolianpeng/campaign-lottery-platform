# 配置说明

本文档说明后端服务 `backend-server` 的配置文件用法，重点覆盖 MySQL 与 Redis 连接配置。

## 配置文件位置

后端配置从环境变量读取。本地开发建议复制示例文件：

```bash
cp backend-server/.env.example backend-server/.env.local
```

Windows PowerShell 可使用：

```powershell
Copy-Item backend-server\.env.example backend-server\.env.local
```

`backend-server/.env.local` 用于保存本地真实连接信息，已被 `.gitignore` 忽略，不应提交到仓库。

## 基础配置

- `ADMIN_USER`：管理端登录用户名，默认 `admin`。
- `ADMIN_PASSWORD`：管理端登录密码，本地启动前建议修改。
- `CORS_ALLOW_ORIGIN`：允许跨域访问后端 API 的前端地址，例如 `http://localhost:3000`。

## MySQL 配置

- `MYSQL_ENABLED`：是否启用 MySQL 连接检查。设置为 `true` 后，健康检查会 ping MySQL。
- `MYSQL_DSN`：MySQL 连接字符串。当前支持两种格式：
  - Go 风格 DSN：`user:password@tcp(host:port)/database?parseTime=true&charset=utf8mb4&loc=Local`
  - URL 风格 DSN：`mysql://user:password@host:port/database?charset=utf8mb4`
- `MYSQL_HOST`：MySQL 主机地址。未配置 `MYSQL_DSN` 时使用。
- `MYSQL_PORT`：MySQL 端口。未配置 `MYSQL_DSN` 时使用，默认 `3306`。
- `MYSQL_DATABASE`：数据库名。未配置 `MYSQL_DSN` 时使用。
- `MYSQL_USER`：数据库用户名。未配置 `MYSQL_DSN` 时使用。
- `MYSQL_PASSWORD`：数据库密码。未配置 `MYSQL_DSN` 时使用。
- `MYSQL_CHARSET`：字符集，默认 `utf8mb4`。
- `MYSQL_CONNECTION_LIMIT`：连接池最大连接数，默认 `10`。

如果同时配置了 `MYSQL_DSN` 和拆分字段，后端优先使用 `MYSQL_DSN`。

示例：

```env
MYSQL_ENABLED=true
MYSQL_DSN=user:password@tcp(127.0.0.1:3306)/campaign_lottery_platform?parseTime=true&charset=utf8mb4&loc=Local
MYSQL_CONNECTION_LIMIT=10
```

## Redis 配置

- `REDIS_ENABLED`：是否启用 Redis 连接检查。设置为 `true` 后，健康检查会 ping Redis。
- `REDIS_URL`：Redis 连接字符串，例如 `redis://127.0.0.1:6379/10`。配置后优先使用。
- `REDIS_HOST`：Redis 主机地址。未配置 `REDIS_URL` 时使用。
- `REDIS_PORT`：Redis 端口。未配置 `REDIS_URL` 时使用，默认 `6379`。
- `REDIS_PASSWORD`：Redis 密码。无密码时留空。
- `REDIS_DATABASE`：Redis 数据库编号，默认 `0`。
- `REDIS_KEY_PREFIX`：业务 key 前缀，例如 `campaign:lottery:`。

示例：

```env
REDIS_ENABLED=true
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DATABASE=10
REDIS_KEY_PREFIX=campaign:lottery:
```

## 健康检查

启动后端后访问：

```text
http://localhost:18100/healthz
```

响应中的 `dependencies.mysql` 和 `dependencies.redis` 会展示依赖状态：

- `ok`：已启用且连接成功。
- `disabled`：配置未启用，对服务启动不构成异常。
- `error`：已启用但连接失败，接口会返回 `degraded` 状态和 `503` 状态码。

## 当前业务落库状态

当前项目已经接入 MySQL 与 Redis 的配置化连接和健康检查，但业务数据仍由 `MemoryStore` 保存在服务进程内存中。生产化落库可基于 `docs/database-design.md` 的表结构继续推进。
