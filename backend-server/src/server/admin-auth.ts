import { createHash, randomBytes, timingSafeEqual } from 'node:crypto';
import type { RowDataPacket } from 'mysql2/promise';
import { getMysqlPool } from './database';
import { badAdminAuth } from './errors';
import { incrementRateLimit } from './redis-helpers';

const ADMIN_LOCKOUT_ATTEMPTS = 5;
const ADMIN_LOCKOUT_WINDOW_SECONDS = 900;

function hashPassword(password: string, salt: string): string {
  return createHash('sha256').update(`${salt}:${password}`).digest('hex');
}

export function hashAdminPassword(password: string): { readonly hash: string; readonly salt: string } {
  const salt = randomBytes(16).toString('hex');
  return { hash: hashPassword(password, salt), salt: `${salt}$${hashPassword(password, salt)}` };
}

export function verifyAdminPassword(password: string, stored: string): boolean {
  const [salt, expected] = stored.split('$');
  if (!salt || !expected) {
    return password === stored;
  }
  const actual = hashPassword(password, salt);
  try {
    return timingSafeEqual(Buffer.from(actual), Buffer.from(expected));
  } catch {
    return false;
  }
}

interface AdminUserRow extends RowDataPacket {
  id: number;
  username: string;
  password_hash: string;
}

export async function verifyAdminLogin(username: string, password: string, ip: string): Promise<{ readonly adminUserId: number; readonly username: string }> {
  const allowed = await incrementRateLimit(`admin-login:${ip}`, ADMIN_LOCKOUT_ATTEMPTS, ADMIN_LOCKOUT_WINDOW_SECONDS);
  if (!allowed) {
    throw badAdminAuth;
  }

  const configUsername = process.env.ADMIN_USER ?? 'admin';
  const configPassword = process.env.ADMIN_PASSWORD ?? '';

  const pool = getMysqlPool();
  if (pool) {
    const [rows] = await pool.query<AdminUserRow[]>(
      'SELECT id, username, password_hash FROM admin_users WHERE username = ? AND status = ? LIMIT 1',
      [username, 'active'],
    );
    const row = rows[0];
    if (row && verifyAdminPassword(password, row.password_hash)) {
      return { adminUserId: row.id, username: row.username };
    }
  }

  if (username === configUsername && configPassword && password === configPassword) {
    return { adminUserId: 0, username };
  }

  throw badAdminAuth;
}
