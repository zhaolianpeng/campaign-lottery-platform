import { createHash, randomBytes, createDecipheriv } from 'node:crypto';
import { getAppConfig } from './config';
import { wechatAuthFailed } from './errors';
import type { JssdkConfig } from './types';

interface WechatAccessTokenResult {
  readonly access_token: string;
  readonly expires_in: number;
  readonly openid: string;
  readonly session_key?: string;
  readonly refresh_token?: string;
  readonly scope?: string;
}

interface WechatUserInfoResult {
  readonly openid: string;
  readonly nickname: string;
  readonly sex: number;
  readonly province: string;
  readonly city: string;
  readonly country: string;
  readonly headimgurl: string;
  readonly privilege?: readonly string[];
  readonly unionid?: string;
}

/**
 * 获取微信 OAuth2 授权页 URL（snsapi_userinfo 静默授权或弹出授权）
 */
export function getOauthUrl(scope: 'snsapi_base' | 'snsapi_userinfo' = 'snsapi_userinfo', state = ''): string {
  const config = getAppConfig().wechat;
  if (!config.appId || !config.redirectUri) {
    throw wechatAuthFailed;
  }

  const redirectEncoded = encodeURIComponent(config.redirectUri);
  return `https://open.weixin.qq.com/connect/oauth2/authorize?appid=${config.appId}&redirect_uri=${redirectEncoded}&response_type=code&scope=${scope}&state=${state}#wechat_redirect`;
}

/**
 * 用 code 换取 access_token 和 openid
 */
export async function exchangeCode(code: string): Promise<WechatAccessTokenResult> {
  const config = getAppConfig().wechat;
  if (!config.appId || !config.appSecret) {
    throw wechatAuthFailed;
  }

  const url = `https://api.weixin.qq.com/sns/oauth2/access_token?appid=${config.appId}&secret=${config.appSecret}&code=${code}&grant_type=authorization_code`;

  const response = await fetch(url);
  const data = await response.json() as Record<string, unknown>;

  if (data.errcode && data.errcode !== 0) {
    throw new Error(`微信登录失败: ${data.errmsg ?? 'unknown error'}`);
  }

  return data as unknown as WechatAccessTokenResult;
}

/**
 * 用 access_token 和 openid 获取用户基本信息
 */
export async function getUserInfo(accessToken: string, openid: string): Promise<WechatUserInfoResult> {
  const url = `https://api.weixin.qq.com/sns/userinfo?access_token=${accessToken}&openid=${openid}&lang=zh_CN`;

  const response = await fetch(url);
  const data = await response.json() as Record<string, unknown>;

  if (data.errcode && data.errcode !== 0) {
    throw new Error(`获取微信用户信息失败: ${data.errmsg ?? 'unknown error'}`);
  }

  return data as unknown as WechatUserInfoResult;
}

/**
 * 获取微信 JS-SDK 的 access_token（用于签名）
 * 注意：这是全局 access_token，不同于 OAuth 的 sns access_token
 */
let cachedJsAccessToken: { token: string; expiresAt: number } | null = null;

async function getJsAccessToken(): Promise<string> {
  const now = Date.now();
  if (cachedJsAccessToken && cachedJsAccessToken.expiresAt > now) {
    return cachedJsAccessToken.token;
  }

  const config = getAppConfig().wechat;
  const url = `https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=${config.appId}&secret=${config.appSecret}`;

  const response = await fetch(url);
  const data = await response.json() as Record<string, unknown>;

  if (data.errcode && data.errcode !== 0) {
    throw new Error(`获取 JS-SDK token 失败: ${data.errmsg ?? 'unknown error'}`);
  }

  const token = data.access_token as string;
  const expiresIn = (data.expires_in as number) - 300; // 提前 5 分钟过期

  cachedJsAccessToken = { token, expiresAt: now + expiresIn * 1000 };
  return token;
}

/**
 * 获取 JS-SDK ticket（用于签名）
 */
let cachedJsTicket: { ticket: string; expiresAt: number } | null = null;

async function getJsTicket(): Promise<string> {
  const now = Date.now();
  if (cachedJsTicket && cachedJsTicket.expiresAt > now) {
    return cachedJsTicket.ticket;
  }

  const accessToken = await getJsAccessToken();
  const url = `https://api.weixin.qq.com/cgi-bin/ticket/getticket?access_token=${accessToken}&type=jsapi`;

  const response = await fetch(url);
  const data = await response.json() as Record<string, unknown>;

  if (data.errcode && data.errcode !== 0) {
    throw new Error(`获取 JS-SDK ticket 失败: ${data.errmsg ?? 'unknown error'}`);
  }

  const ticket = data.ticket as string;
  const expiresIn = (data.expires_in as number) - 300;

  cachedJsTicket = { ticket, expiresAt: now + expiresIn * 1000 };
  return ticket;
}

/**
 * 生成 JS-SDK 配置签名
 */
export async function getJssdkConfig(pageUrl: string): Promise<JssdkConfig> {
  const config = getAppConfig().wechat;
  const ticket = await getJsTicket();
  const nonceStr = randomBytes(8).toString('hex');
  const timestamp = Math.floor(Date.now() / 1000);

  const raw = `jsapi_ticket=${ticket}&noncestr=${nonceStr}&timestamp=${timestamp}&url=${pageUrl}`;
  const signature = createHash('sha1').update(raw).digest('hex');

  return {
    appId: config.appId,
    timestamp,
    nonceStr,
    signature,
  };
}

/**
 * 解密微信手机号加密数据
 * @param sessionKey 从 OAuth 获取的 session_key
 * @param encryptedData 前端 wx.getPhoneNumber 返回的加密数据
 * @param iv 加密算法的初始向量
 * @returns 手机号
 */
export function decryptPhoneNumber(sessionKey: string, encryptedData: string, iv: string): string {
  const key = Buffer.from(sessionKey, 'base64');
  const decipherIv = Buffer.from(iv, 'base64');
  const encrypted = Buffer.from(encryptedData, 'base64');

  const decipher = createDecipheriv('aes-128-cbc', key, decipherIv);
  decipher.setAutoPadding(true);

  let decoded = decipher.update(encrypted, undefined, 'utf8');
  decoded += decipher.final('utf8');

  const result = JSON.parse(decoded) as { phoneNumber?: string; purePhoneNumber?: string };

  return result.purePhoneNumber ?? result.phoneNumber ?? '';
}
