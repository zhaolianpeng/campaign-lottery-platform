import { createSign, createVerify, randomBytes } from 'node:crypto';

export function formatPrivateKey(pemOrBase64: string): string {
  const trimmed = pemOrBase64.trim();
  if (trimmed.includes('BEGIN')) {
    return trimmed;
  }
  const body = trimmed.replace(/\s+/g, '');
  const lines = body.match(/.{1,64}/g) ?? [];
  return `-----BEGIN PRIVATE KEY-----\n${lines.join('\n')}\n-----END PRIVATE KEY-----`;
}

export function formatPublicKey(pemOrBase64: string): string {
  const trimmed = pemOrBase64.trim();
  if (trimmed.includes('BEGIN')) {
    return trimmed;
  }
  const body = trimmed.replace(/\s+/g, '');
  const lines = body.match(/.{1,64}/g) ?? [];
  return `-----BEGIN PUBLIC KEY-----\n${lines.join('\n')}\n-----END PUBLIC KEY-----`;
}

export function signAlipayParams(
  params: Record<string, string>,
  privateKeyPem: string,
): string {
  const sorted = Object.keys(params)
    .filter((key) => key !== 'sign' && params[key] !== '' && params[key] !== undefined)
    .sort()
    .map((key) => `${key}=${params[key]}`)
    .join('&');

  const signer = createSign('RSA-SHA256');
  signer.update(sorted, 'utf8');
  signer.end();
  return signer.sign(formatPrivateKey(privateKeyPem), 'base64');
}

export function verifyAlipayParams(
  params: Record<string, string>,
  alipayPublicKeyPem: string,
): boolean {
  const sign = params.sign;
  if (!sign) {
    return false;
  }

  const sorted = Object.keys(params)
    .filter(
      (key) =>
        key !== 'sign' &&
        key !== 'sign_type' &&
        params[key] !== '' &&
        params[key] !== undefined,
    )
    .sort()
    .map((key) => `${key}=${params[key]}`)
    .join('&');

  const verifier = createVerify('RSA-SHA256');
  verifier.update(sorted, 'utf8');
  verifier.end();
  return verifier.verify(formatPublicKey(alipayPublicKeyPem), sign, 'base64');
}

export function buildAlipayGetUrl(gateway: string, params: Record<string, string>): string {
  const query = new URLSearchParams(params).toString();
  return `${gateway}?${query}`;
}

export function randomAlipayNonce(): string {
  return randomBytes(16).toString('hex');
}
