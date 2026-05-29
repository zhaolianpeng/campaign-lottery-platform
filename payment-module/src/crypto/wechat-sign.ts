import { createSign, createVerify, randomBytes, createDecipheriv } from 'node:crypto';

export function buildWechatAuthorization(
  method: string,
  urlPathWithQuery: string,
  body: string,
  mchId: string,
  serialNo: string,
  privateKeyPem: string,
): { authorization: string; nonce: string; timestamp: string } {
  const timestamp = Math.floor(Date.now() / 1000).toString();
  const nonce = randomBytes(16).toString('hex');
  const message = `${method}\n${urlPathWithQuery}\n${timestamp}\n${nonce}\n${body}\n`;
  const sign = createSign('RSA-SHA256');
  sign.update(message);
  sign.end();
  const signature = sign.sign(privateKeyPem, 'base64');
  const authorization =
    `WECHATPAY2-SHA256-RSA2048 mchid="${mchId}",nonce_str="${nonce}",signature="${signature}",timestamp="${timestamp}",serial_no="${serialNo}"`;
  return { authorization, nonce, timestamp };
}

export function verifyWechatNotifySignature(
  timestamp: string,
  nonce: string,
  body: string,
  signature: string,
  platformPublicKeyPem: string,
): boolean {
  const message = `${timestamp}\n${nonce}\n${body}\n`;
  const verifier = createVerify('RSA-SHA256');
  verifier.update(message);
  verifier.end();
  return verifier.verify(platformPublicKeyPem, signature, 'base64');
}

/** 解密微信支付 V3 回调 resource（ciphertext 为 base64，末 16 字节为 auth tag） */
export function decryptWechatResource(
  apiV3Key: string,
  associatedData: string,
  nonce: string,
  ciphertextBase64: string,
): string {
  const key = Buffer.from(apiV3Key, 'utf8');
  const buf = Buffer.from(ciphertextBase64, 'base64');
  const authTag = buf.subarray(buf.length - 16);
  const data = buf.subarray(0, buf.length - 16);

  const decipher = createDecipheriv('aes-256-gcm', key, Buffer.from(nonce, 'utf8'));
  decipher.setAuthTag(authTag);
  decipher.setAAD(Buffer.from(associatedData, 'utf8'));

  const decrypted = Buffer.concat([decipher.update(data), decipher.final()]);
  return decrypted.toString('utf8');
}

export function buildJsapiPaySign(
  appId: string,
  timeStamp: string,
  nonceStr: string,
  packageValue: string,
  privateKeyPem: string,
): string {
  const message = `${appId}\n${timeStamp}\n${nonceStr}\n${packageValue}\n`;
  const sign = createSign('RSA-SHA256');
  sign.update(message);
  sign.end();
  return sign.sign(privateKeyPem, 'base64');
}
