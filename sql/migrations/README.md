# SQL Migrations

新的数据库变更请按顺序添加到本目录，命名建议使用 `NNN_description.sql`。

- `sql/schema.mysql.sql` 作为基线迁移，由 `backend-server/scripts/migrate.cjs` 首次执行。
- 后续增量 DDL 和数据修复请只追加到本目录，不要再把运行时建表逻辑塞回应用代码。
- 发布时统一通过 `cd backend-server && npm run migrate` 执行。