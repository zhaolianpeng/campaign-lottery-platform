import { randomBytes } from 'node:crypto';

export function generateOrderNo(prefix = 'pay'): string {
  const ts = Date.now().toString(36);
  const rand = randomBytes(4).toString('hex');
  return `${prefix}_${ts}_${rand}`;
}

export function generateRefundNo(prefix = 'ref'): string {
  return generateOrderNo(prefix);
}
