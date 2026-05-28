# 总览与抽奖记录

## 1. 模块概述

| Tab | 用户目标 | API |
|-----|----------|-----|
| `overview` | 查看运营概览数字与最近动态 | `GET admin/overview` |
| `records` | 审计抽奖日志 | `GET admin/draw-records` |

## 2. 总览 `overview` **[已实现]**

### 界面清单

- 统计卡片：用户数、抽奖数、中奖数等（以 `AdminOverview` 为准）
- 在线盲盒列表摘要
- 最近抽奖记录片段

### 流程

1. 进入 Tab → `overviewQuery`
2. 只读，无编辑

### 状态

| 状态 | UI |
|------|-----|
| loading | Loader |
| error | 错误提示 |

## 3. 抽奖记录 `records` **[已实现]**

### 界面清单

- 表格：用户、盲盒、奖品、结果、时间等
- `shortId` 截断展示 ID

### 流程

1. 进入 Tab → `recordsQuery`
2. 滚动浏览，无导出按钮 **[规划中]**

## 4. 与产品文档差异表

| 能力 | 状态 | 备注 |
|------|------|------|
| 概率偏离分析 | **[规划中]** | |
| 收入排名图表 | **[规划中]** | |
| 异常行为标记 | **[规划中]** | |
| 记录高级筛选 | **[部分实现]** | 全量列表 |

## 5. 关联文档

- [04-campaign-prize-pity.md](./04-campaign-prize-pity.md)
- [05-fulfillment.md](./05-fulfillment.md)
