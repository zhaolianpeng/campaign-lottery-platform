import { getMysqlPool } from './database';
import { tryFulfillPaidOrder } from './payment-fulfillment';

export async function logPaymentNotify(
  channel: string,
  orderNo: string,
  notifyId: string,
  rawBody: string,
  verified: boolean,
): Promise<boolean> {
  const pool = getMysqlPool();
  if (!pool || !notifyId) {
    return true;
  }
  try {
    await pool.query(
      'INSERT INTO payment_notify_logs (order_no, channel, notify_id, raw_body, verified, created_at) VALUES (?, ?, ?, ?, ?, ?)',
      [orderNo, channel, notifyId, rawBody.slice(0, 65535), verified ? 1 : 0, new Date()],
    );
    return true;
  } catch (error) {
    if (error && typeof error === 'object' && 'code' in error && error.code === 'ER_DUP_ENTRY') {
      return false;
    }
    throw error;
  }
}

export async function fulfillOrderAfterNotify(orderNo: string): Promise<void> {
  await tryFulfillPaidOrder(orderNo);
}
