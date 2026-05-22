# 盲盒抽奖平台 (Blind Box Lottery Platform)

一个完整的线上盲盒抽奖平台后端，支持系列展示、抽盒（单抽/十连）、保底机制、库存管理、交换市场、积分会员系统和发奖管理。

## 技术栈

- **语言**: Go 1.24+
- **HTTP**: net/http (无框架, Go 1.22+ 路由模式)
- **数据库**: MySQL + Redis (生产), 内存存储 (开发/演示)
- **前端**: 原生 H5 + Admin 页面

## 快速开始（本地开发，无需 MySQL）

```bash
cd backend
go run ./cmd/server/main.go
```

服务启动在 `:18100`，自动使用内存存储，无需 MySQL 或 Redis。

### 种子数据

启动后自动加载 3 个活动/系列：

| 系列 | ID | 说明 |
|------|-----|------|
| 🎁 夏季开门红抽奖活动 | camp_launch_001 | 原始营销活动（红包/优惠券） |
| 🌙 星空系列 | series_starry_001 | 10款：6普通+2稀有+1隐藏+1限定 |
| 🐱 猫咪系列 | series_cat_001 | 7款：5普通+1稀有+1隐藏 |

## 项目结构

```
campaign-lottery-platform/
├── backend/
│   ├── cmd/server/main.go          # 入口
│   ├── go.mod / go.sum
│   ├── .env.example
│   ├── internal/
│   │   ├── config/config.go        # 配置（环境变量）
│   │   ├── model/models.go         # 数据模型
│   │   ├── store/
│   │   │   ├── store.go            # Store 接口定义
│   │   │   ├── memory.go           # 内存存储实现
│   │   │   └── mysql.go            # MySQL 存储实现
│   │   ├── service/service.go      # 业务逻辑层
│   │   ├── router/router.go        # HTTP 路由
│   │   ├── probability/            # 抽奖概率引擎
│   │   │   ├── engine.go           # 核心抽奖引擎
│   │   │   ├── engine_test.go      # 引擎测试
│   │   │   ├── alias.go            # Alias Method O(1) 采样
│   │   │   └── pity.go             # 软/硬保底系统
│   │   └── response/json.go        # JSON 响应格式
├── frontend/
│   ├── h5/                         # 用户端 H5 页面
│   └── admin/                      # 管理端页面
├── sql/schema.mysql.sql            # MySQL 建表脚本
├── deploy/                         # 部署配置
└── 产品功能设计文档.md              # 完整产品功能文档
```

## API 概览

### 用户端 API (`/api/v1/`)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/auth/guest-login` | 游客登录 |
| GET | `/campaigns` | 活动/系列列表 |
| GET | `/me/draw-records` | 我的抽奖记录 |
| POST | `/lottery/draw` | 原始抽奖（兼容） |
| GET | `/blindbox/campaigns/{id}/probabilities` | 概率公示详情 |
| POST | `/blindbox/draw` | 盲盒抽奖（支持单抽/十连） |
| GET | `/blindbox/pity-status?campaign_id=` | 保底状态 |
| GET | `/blindbox/inventory` | 用户库存 |
| GET | `/blindbox/series-progress?campaign_id=` | 系列收集进度 |
| GET | `/blindbox/exchange-offers` | 交换市场列表 |
| POST | `/blindbox/exchange-offers` | 创建交换挂单 |
| DELETE | `/blindbox/exchange-offers/{id}` | 取消挂单 |
| POST | `/blindbox/exchange-offers/{id}/accept` | 接受交换 |
| GET | `/blindbox/member` | 会员/积分信息 |
| GET | `/blindbox/points-log` | 积分变动记录 |
| POST | `/blindbox/redeem` | 积分兑换奖品 |

### 管理端 API (`/api/v1/admin/`)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/admin/login` | 管理员登录 |
| GET | `/admin/overview` | 总览数据 |
| GET/POST | `/admin/campaigns` | 活动列表/创建 |
| PUT/DELETE | `/admin/campaigns/{id}` | 编辑/删除活动 |
| GET/POST | `/admin/campaigns/{id}/prizes` | 奖品列表/创建 |
| PUT/DELETE | `/admin/prizes/{id}` | 编辑/删除奖品 |
| GET | `/admin/fulfillment-tasks` | 发奖任务列表 |
| PATCH | `/admin/fulfillment-tasks/{id}` | 更新发奖状态 |
| GET | `/admin/statistics` | 数据统计 |

## 概率引擎

使用 **Alias Method** 实现 O(1) 加权随机抽取，支持：
- **软保底 (Soft Pity)**：连续 N 次未中奖后概率递增
- **硬保底 (Hard Pity)**：最多 M 次必出稀有/以上
- **Monte Carlo 验证**：通过大量模拟验证概率分布

详见 [`backend/internal/probability/`](backend/internal/probability/)

## 部署

```bash
# 1. 创建数据库
mysql -u root -p < sql/schema.mysql.sql

# 2. 配置环境变量
cp backend/.env.example backend/.env
# 编辑 .env 填入数据库和 Redis 配置

# 3. 启动
cd backend && go run ./cmd/server/main.go
```

或使用 systemd + nginx（见 `deploy/` 目录）。

## 开发状态

- [x] 产品功能设计文档
- [x] 概率引擎（Alias Method + 保底机制）
- [x] 用户认证（游客登录）
- [x] 盲盒抽奖（单抽/十连）
- [x] 保底状态查询
- [x] 库存管理和收集进度
- [x] 交换市场（发布/匹配/接受）
- [x] 积分/会员体系
- [x] 积分兑换奖品
- [x] 管理端 CRUD
- [x] 发奖管理
- [x] 数据统计
- [x] MySQL 持久化实现
- [x] 内存存储（本地开发）
- [ ] 前端盲盒开盒动画
- [ ] 社交分享功能
- [ ] 集齐系列解锁奖励
