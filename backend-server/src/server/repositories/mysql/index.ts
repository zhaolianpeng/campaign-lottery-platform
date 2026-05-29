import type { RowDataPacket } from 'mysql2/promise';
import { getMysqlPool } from '../../database';
import type { AdminAuditEntry, AdminAuditRepo, DrawRepo, LedgerRepo, PaymentOrderRepo, PaymentOrderRow, PityRepo, SessionRepo, UserRepo } from '../types';
import type { DrawRecord, User } from '../../types';

interface UserRow extends RowDataPacket {
  id: string;
  nickname: string;
  mobile: string;
  status: string;
}

interface DrawRow extends RowDataPacket {
  id: string;
  campaign_id: string;
  user_id: string;
  prize_id: string | null;
  prize_name: string;
  result: string;
  chance_after: number;
  request_id: string;
  created_at: Date | string;
}

function mysqlUserRepo(): UserRepo {
  return {
    async findById(userId) {
      const pool = getMysqlPool();
      if (!pool) return null;
      const [rows] = await pool.query<UserRow[]>('SELECT id, nickname, mobile, status FROM users WHERE id = ? LIMIT 1', [userId]);
      const row = rows[0];
      if (!row) return null;
      return {
        id: row.id,
        nickname: row.nickname,
        mobile: row.mobile,
        status: row.status as User['status'],
        created_at: new Date().toISOString(),
      };
    },
    async findByMobile(mobile) {
      const pool = getMysqlPool();
      if (!pool) return null;
      const [rows] = await pool.query<UserRow[]>('SELECT id, nickname, mobile, status FROM users WHERE mobile = ? LIMIT 1', [mobile]);
      const row = rows[0];
      if (!row) return null;
      return {
        id: row.id,
        nickname: row.nickname,
        mobile: row.mobile,
        status: row.status as User['status'],
        created_at: new Date().toISOString(),
      };
    },
    async create(user) {
      const pool = getMysqlPool();
      if (!pool) throw new Error('MySQL not enabled');
      const now = new Date();
      await pool.query(
        'INSERT INTO users (id, nickname, mobile, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)',
        [user.id, user.nickname, user.mobile, user.status, now, now],
      );
    },
    async updateStatus(userId, status) {
      const pool = getMysqlPool();
      if (!pool) return;
      await pool.query('UPDATE users SET status = ?, updated_at = ? WHERE id = ?', [status, new Date(), userId]);
    },
  };
}

function mysqlSessionRepo(): SessionRepo {
  return {
    async create(token, userId, expiresAt) {
      const pool = getMysqlPool();
      if (!pool) throw new Error('MySQL not enabled');
      await pool.query('INSERT INTO user_sessions (token, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)', [
        token,
        userId,
        expiresAt,
        new Date(),
      ]);
    },
    async findUserIdByToken(token) {
      const pool = getMysqlPool();
      if (!pool) return null;
      const [rows] = await pool.query<RowDataPacket[]>('SELECT user_id FROM user_sessions WHERE token = ? AND expires_at > ? LIMIT 1', [
        token,
        new Date(),
      ]);
      return (rows[0]?.user_id as string | undefined) ?? null;
    },
    async deleteToken(token) {
      const pool = getMysqlPool();
      if (!pool) return;
      await pool.query('DELETE FROM user_sessions WHERE token = ?', [token]);
    },
  };
}

function mysqlDrawRepo(): DrawRepo {
  return {
    async findByRequestId(requestId) {
      if (!requestId) return null;
      const pool = getMysqlPool();
      if (!pool) return null;
      const [rows] = await pool.query<DrawRow[]>('SELECT * FROM draw_records WHERE request_id = ? LIMIT 1', [requestId]);
      const row = rows[0];
      if (!row) return null;
      return {
        id: row.id,
        campaign_id: row.campaign_id,
        user_id: row.user_id,
        prize_id: row.prize_id ?? undefined,
        prize_name: row.prize_name,
        result: row.result as DrawRecord['result'],
        drawn_at: row.created_at instanceof Date ? row.created_at.toISOString() : new Date(row.created_at).toISOString(),
        chance_after: row.chance_after,
      };
    },
    async insertDraw(record, requestId) {
      const pool = getMysqlPool();
      if (!pool) throw new Error('MySQL not enabled');
      await pool.query(
        'INSERT INTO draw_records (id, campaign_id, user_id, prize_id, prize_name, result, chance_after, request_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)',
        [
          record.id,
          record.campaign_id,
          record.user_id,
          record.prize_id ?? null,
          record.prize_name,
          record.result,
          record.chance_after,
          requestId,
          new Date(record.drawn_at),
        ],
      );
    },
  };
}

function mysqlLedgerRepo(): LedgerRepo {
  return {
    async getPoints(userId) {
      const pool = getMysqlPool();
      if (!pool) return 0;
      const [rows] = await pool.query<RowDataPacket[]>('SELECT points FROM user_members WHERE user_id = ? LIMIT 1', [userId]);
      return Number(rows[0]?.points ?? 0);
    },
    async addPoints(userId, delta, reason, remark) {
      const pool = getMysqlPool();
      if (!pool) throw new Error('MySQL not enabled');
      const conn = await pool.getConnection();
      try {
        await conn.beginTransaction();
        const [rows] = await conn.query<RowDataPacket[]>('SELECT points FROM user_members WHERE user_id = ? FOR UPDATE', [userId]);
        const current = Number(rows[0]?.points ?? 0);
        const next = current + delta;
        if (rows[0]) {
          await conn.query('UPDATE user_members SET points = ?, updated_at = ? WHERE user_id = ?', [next, new Date(), userId]);
        } else {
          await conn.query('INSERT INTO user_members (user_id, level, points, total_draws, total_spent, created_at, updated_at) VALUES (?, ?, ?, 0, 0, ?, ?)', [
            userId,
            'normal',
            next,
            new Date(),
            new Date(),
          ]);
        }
        await conn.query('INSERT INTO user_points_logs (user_id, points, balance, reason, remark, created_at) VALUES (?, ?, ?, ?, ?, ?)', [
          userId,
          delta,
          next,
          reason,
          remark,
          new Date(),
        ]);
        await conn.commit();
        return next;
      } catch (error) {
        await conn.rollback();
        throw error;
      } finally {
        conn.release();
      }
    },
  };
}

function mysqlPityRepo(): PityRepo {
  return {
    async getState(userId, campaignId) {
      const pool = getMysqlPool();
      if (!pool) return { soft: 0, hard: 0, upGuarantee: false };
      const [rows] = await pool.query<RowDataPacket[]>(
        'SELECT soft_pity_count, hard_pity_count, up_pool_guarantee FROM user_pity_state WHERE user_id = ? AND campaign_id = ? LIMIT 1',
        [userId, campaignId],
      );
      const row = rows[0];
      return {
        soft: Number(row?.soft_pity_count ?? 0),
        hard: Number(row?.hard_pity_count ?? 0),
        upGuarantee: Boolean(row?.up_pool_guarantee),
      };
    },
    async saveState(userId, campaignId, state) {
      const pool = getMysqlPool();
      if (!pool) return;
      await pool.query(
        `INSERT INTO user_pity_state (user_id, campaign_id, soft_pity_count, hard_pity_count, up_pool_guarantee, updated_at)
         VALUES (?, ?, ?, ?, ?, ?)
         ON DUPLICATE KEY UPDATE soft_pity_count = VALUES(soft_pity_count), hard_pity_count = VALUES(hard_pity_count),
           up_pool_guarantee = VALUES(up_pool_guarantee), updated_at = VALUES(updated_at)`,
        [userId, campaignId, state.soft, state.hard, state.upGuarantee ? 1 : 0, new Date()],
      );
    },
  };
}

function mysqlPaymentOrderRepo(): PaymentOrderRepo {
  return {
    async upsert(order) {
      const pool = getMysqlPool();
      if (!pool) throw new Error('MySQL not enabled');
      await pool.query(
        `INSERT INTO payment_orders
          (order_no, user_id, client_request_id, channel, presentation, subject, business_type, business_id, amount_cents, status, expire_at, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'created', DATE_ADD(NOW(), INTERVAL 30 MINUTE), NOW(), NOW())
         ON DUPLICATE KEY UPDATE updated_at = NOW()`,
        [order.orderNo, order.userId, order.clientRequestId, order.channel, order.presentation, order.subject, order.businessType, order.businessId, order.amountCents],
      );
    },
    async findByOrderNo(orderNo) {
      const pool = getMysqlPool();
      if (!pool) return null;
      const [rows] = await pool.query<RowDataPacket[]>('SELECT * FROM payment_orders WHERE order_no = ? LIMIT 1', [orderNo]);
      const row = rows[0];
      if (!row) return null;
      return mapPaymentOrderRow(row);
    },
    async updateStatus(orderNo, status, patch) {
      const pool = getMysqlPool();
      if (!pool) return;
      await pool.query(
        'UPDATE payment_orders SET status = ?, channel_trade_no = COALESCE(?, channel_trade_no), paid_at = COALESCE(?, paid_at), updated_at = NOW() WHERE order_no = ?',
        [status, patch?.channelTradeNo ?? null, patch?.paidAt ?? null, orderNo],
      );
    },
    async listByUser(userId, limit = 50) {
      const pool = getMysqlPool();
      if (!pool) return [];
      const [rows] = await pool.query<RowDataPacket[]>(
        'SELECT * FROM payment_orders WHERE user_id = ? ORDER BY created_at DESC LIMIT ?',
        [userId, limit],
      );
      return rows.map(mapPaymentOrderRow);
    },
  };
}

function mapPaymentOrderRow(row: RowDataPacket): PaymentOrderRow {
  return {
    orderNo: String(row.order_no),
    userId: String(row.user_id),
    clientRequestId: String(row.client_request_id),
    channel: String(row.channel),
    businessType: String(row.business_type),
    businessId: String(row.business_id),
    amountCents: Number(row.amount_cents),
    status: String(row.status),
  };
}

function mysqlAdminAuditRepo(): AdminAuditRepo {
  return {
    async insert(entry) {
      const pool = getMysqlPool();
      if (!pool) return;
      await pool.query(
        'INSERT INTO admin_audit_logs (admin_user_id, admin_username, action, resource_type, resource_id, request_id, payload_json, ip_address, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)',
        [
          entry.adminUserId ?? null,
          entry.adminUsername,
          entry.action,
          entry.resourceType,
          entry.resourceId,
          entry.requestId,
          entry.payload ? JSON.stringify(entry.payload) : null,
          entry.ipAddress,
          new Date(),
        ],
      );
    },
    async listRecent(limit = 100) {
      const pool = getMysqlPool();
      if (!pool) return [];
      const [rows] = await pool.query<RowDataPacket[]>(
        'SELECT admin_username, action, resource_type, resource_id, request_id, ip_address FROM admin_audit_logs ORDER BY created_at DESC LIMIT ?',
        [limit],
      );
      return rows.map((row) => ({
        adminUsername: String(row.admin_username),
        action: String(row.action),
        resourceType: String(row.resource_type),
        resourceId: String(row.resource_id),
        requestId: String(row.request_id),
        ipAddress: String(row.ip_address),
      }));
    },
  };
}

export function createMysqlRepositories(): import('../types').RepositoryBundle {
  return {
    users: mysqlUserRepo(),
    sessions: mysqlSessionRepo(),
    draws: mysqlDrawRepo(),
    ledger: mysqlLedgerRepo(),
    pity: mysqlPityRepo(),
    paymentOrders: mysqlPaymentOrderRepo(),
    adminAudit: mysqlAdminAuditRepo(),
  };
}
