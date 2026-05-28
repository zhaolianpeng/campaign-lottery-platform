# 交互设计文档编写规范

> 适用范围：`docs/interaction-design/` 下全部 IXD 文档  
> 版本：v1.0 | 更新：2026-05-28

## 1. 文档目的

交互设计（Interaction Design, IXD）描述**用户在界面上的操作顺序、系统反馈、界面状态与异常处理**。本文档体系不重复 API 字段与表结构，业务规则见 [`api-design.md`](../api-design.md)、[`database-design.md`](../database-design.md) 及各模块设计稿。

## 2. 事实来源与标注

| 标注 | 含义 |
|------|------|
| **[已实现]** | `front-page` / `backend-server` 当前可复现的交互 |
| **[部分实现]** | 主流程可用，缺 UI、缺风控或仅为演示数据 |
| **[规划中]** | [`产品功能设计文档.md`](../../产品功能设计文档.md) 有描述，代码未落地 |

编写时以代码为交互事实，以产品文档为能力对照；差异写入各篇「与产品文档差异表」。

## 3. 单篇文档模板

每篇模块文档应包含以下章节（可按模块删减无关节）：

1. **模块概述** — 用户目标、路由入口（`/` 或 `/admin` + Tab/子视图）
2. **信息架构** — 壳层 → Tab → 详情/弹窗，附 mermaid 图
3. **界面清单** — 区域/组件、触发条件
4. **核心用户流程** — 主路径 + 分支，逐步编号
5. **交互状态表** — idle / loading / empty / error / disabled
6. **表单与校验** — zod 规则、按钮禁用、错误文案
7. **弹窗与抽屉** — 打开/关闭、栈叠规则
8. **权限与门槛** — token、viewerMode、功能开关、账号状态
9. **与产品文档差异表**
10. **异常与边界**
11. **关联文档**

## 4. 全局技术约定

| 项 | 约定 |
|----|------|
| C 端路由 | 仅 `/`（[`front-page/app/page.tsx`](../../front-page/app/page.tsx)），Tab 无 URL |
| 管理端路由 | 仅 `/admin` |
| 数据请求 | TanStack React Query；`staleTime` 约 20s |
| 表单 | react-hook-form + zod |
| API 信封 | `{ code, message, data }`；错误展示 `ApiRequestError.message` |
| 用户鉴权 | `Authorization: Bearer utk_*` |
| 管理鉴权 | `Authorization: Bearer atk_*` |
| 匿名抽盒 | `X-Anonymous-Draw-Token` + `localStorage` 键 `campaign-lottery-anonymous-draw-token` |

## 5. 状态图例（交互状态表）

| 状态 | UI 典型表现 | 用户操作 |
|------|-------------|----------|
| `idle` | 内容已渲染 | 全部可用操作 |
| `loading` | `Loader2` 旋转、骨架屏 `SkeletonGrid` | 主操作按钮 `disabled` |
| `empty` | `EmptyState` 文案 | 引导至系列/登录等 |
| `error` | 红色文案或 `window.alert` | 重试、返回 |
| `disabled` | 灰显按钮、Tab 隐藏 | 需登录/开关关闭/积分不足 |

## 6. 代码锚点索引

| 端 | 主文件 |
|----|--------|
| C 端 | [`front-page/src/features/lottery/lottery-app.tsx`](../../front-page/src/features/lottery/lottery-app.tsx) |
| 管理端 | [`front-page/src/features/admin/admin-app.tsx`](../../front-page/src/features/admin/admin-app.tsx) |
| 支付 | [`front-page/src/client/payment-checkout.ts`](../../front-page/src/client/payment-checkout.ts) |
| 类型 | [`front-page/src/types/api.ts`](../../front-page/src/types/api.ts) |

## 7. Tab 与模块映射

### C 端 `TabKey`

`series` | `inventory` | `exchange` | `rank` | `member` | `shop` | `social` | `puzzle`

### 管理端 `AdminTab`

`overview` | `users` | `campaigns` | `prizes` | `pity` | `features` | `delivery` | `records` | `monthcard` | `shop`
