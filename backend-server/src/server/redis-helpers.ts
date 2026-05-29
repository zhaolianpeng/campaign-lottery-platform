import { getRedisClient, redisKey } from './redis';

const OTP_TTL_SECONDS = 300;
const MAX_OTP_ATTEMPTS = 5;

export async function storeOtpCode(mobile: string, code: string): Promise<void> {
  const client = getRedisClient();
  if (!client) {
    return;
  }
  if (!client.isOpen) {
    await client.connect();
  }
  const key = redisKey(`sms:code:${mobile}`);
  await client.setEx(key, OTP_TTL_SECONDS, code);
  await client.setEx(redisKey(`sms:attempts:${mobile}`), OTP_TTL_SECONDS, '0');
}

export async function verifyOtpCode(mobile: string, code: string): Promise<boolean> {
  const client = getRedisClient();
  if (!client) {
    return false;
  }
  if (!client.isOpen) {
    await client.connect();
  }
  const key = redisKey(`sms:code:${mobile}`);
  const attemptsKey = redisKey(`sms:attempts:${mobile}`);
  const attempts = Number((await client.get(attemptsKey)) ?? '0');
  if (attempts >= MAX_OTP_ATTEMPTS) {
    return false;
  }
  await client.incr(attemptsKey);
  const stored = await client.get(key);
  if (!stored || stored !== code) {
    return false;
  }
  await client.del(key);
  await client.del(attemptsKey);
  return true;
}

export async function incrementRateLimit(bucket: string, limit: number, windowSeconds: number): Promise<boolean> {
  const client = getRedisClient();
  if (!client) {
    return true;
  }
  if (!client.isOpen) {
    await client.connect();
  }
  const key = redisKey(`rl:${bucket}`);
  const count = await client.incr(key);
  if (count === 1) {
    await client.expire(key, windowSeconds);
  }
  return count <= limit;
}

export async function claimIdempotent(requestId: string, ttlSeconds = 86400): Promise<boolean> {
  const client = getRedisClient();
  if (!client) {
    return true;
  }
  if (!client.isOpen) {
    await client.connect();
  }
  const key = redisKey(`idempotent:${requestId}`);
  const result = await client.set(key, '1', { NX: true, EX: ttlSeconds });
  return result === 'OK';
}
