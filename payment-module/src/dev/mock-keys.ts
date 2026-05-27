import { generateKeyPairSync } from 'node:crypto';

let cached: { privateKeyPem: string; publicKeyPem: string } | null = null;

/** mock 模式使用的临时 RSA 密钥对（进程内缓存） */
export function getMockRsaKeyPair(): { privateKeyPem: string; publicKeyPem: string } {
  if (!cached) {
    const { publicKey, privateKey } = generateKeyPairSync('rsa', {
      modulusLength: 2048,
      publicKeyEncoding: { type: 'spki', format: 'pem' },
      privateKeyEncoding: { type: 'pkcs8', format: 'pem' },
    });
    cached = {
      privateKeyPem: privateKey,
      publicKeyPem: publicKey,
    };
  }
  return cached;
}
