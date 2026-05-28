# C 端功能入口开关

## 1. 模块概述

| 项 | 说明 |
|----|------|
| 用户目标 | 控制 C 端 8 个底部 Tab 是否对用户可见 |
| 入口 | `features` Tab（文案「入口」） |
| API | `GET/PUT admin/feature-toggles` |

映射见 [navigation-shell.md](../cross-cutting/navigation-shell.md)。

## 2. 界面清单

| 元素 | 说明 |
|------|------|
| 开关列表 | 8 项：`cEndFeatureLabels` 对应 `TabKey` |
| Toggle | 点击切换 → 乐观 `setQueryData` + PUT |

## 3. 核心用户流程 **[已实现]**

1. 进入 Tab → `featureTogglesQuery`
2. 切换某项 → `updateFeatureTogglesMutation`
3. C 端下次 `config/public` 或已加载配置生效；未登录仍强制仅 `series`

## 4. 交互状态表

| 状态 | UI |
|------|-----|
| loading | Loader |
| toggling | 开关 disabled 或 loading |

## 5. 异常

- PUT 失败：应回滚 queryData（以实现为准，建议 invalidate）

## 6. 关联文档

- [navigation-shell.md](../cross-cutting/navigation-shell.md)
- [README.md](../README.md)
