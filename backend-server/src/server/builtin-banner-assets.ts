type BuiltinBannerAsset = {
  readonly contentType: 'image/svg+xml';
  readonly body: string;
};

function bannerSvg(options: {
  readonly eyebrow: string;
  readonly title: string;
  readonly subtitle: string;
  readonly accent: string;
  readonly primary: string;
  readonly secondary: string;
  readonly glow: string;
}): string {
  const { eyebrow, title, subtitle, accent, primary, secondary, glow } = options;
  return `
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1200 640" role="img" aria-label="${title}">
  <defs>
    <linearGradient id="bg" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0%" stop-color="${primary}" />
      <stop offset="100%" stop-color="${secondary}" />
    </linearGradient>
    <radialGradient id="halo" cx="0.78" cy="0.28" r="0.55">
      <stop offset="0%" stop-color="${glow}" stop-opacity="0.95" />
      <stop offset="100%" stop-color="${glow}" stop-opacity="0" />
    </radialGradient>
  </defs>
  <rect width="1200" height="640" rx="36" fill="url(#bg)" />
  <rect width="1200" height="640" rx="36" fill="url(#halo)" />
  <circle cx="1020" cy="170" r="132" fill="rgba(255,255,255,0.12)" />
  <circle cx="925" cy="235" r="74" fill="rgba(255,255,255,0.08)" />
  <path d="M768 104C904 94 1022 134 1115 226C1181 291 1234 382 1262 504L982 640L676 640C662 566 665 489 690 417C718 334 748 231 768 104Z" fill="rgba(255,255,255,0.08)" />
  <path d="M852 118L1068 118L1116 164L1116 390L852 390Z" fill="rgba(255,255,255,0.1)" />
  <path d="M886 168L1040 168L1076 204L1076 334L886 334Z" fill="rgba(255,255,255,0.18)" />
  <rect x="74" y="76" width="154" height="42" rx="21" fill="rgba(255,255,255,0.18)" />
  <text x="151" y="104" fill="#ffffff" font-size="24" font-family="Arial, PingFang SC, Microsoft YaHei, sans-serif" text-anchor="middle" letter-spacing="2">${eyebrow}</text>
  <text x="72" y="214" fill="#ffffff" font-size="78" font-weight="700" font-family="Arial, PingFang SC, Microsoft YaHei, sans-serif">${title}</text>
  <text x="72" y="278" fill="rgba(255,255,255,0.88)" font-size="34" font-family="Arial, PingFang SC, Microsoft YaHei, sans-serif">${subtitle}</text>
  <rect x="72" y="470" width="224" height="68" rx="34" fill="rgba(255,255,255,0.16)" stroke="rgba(255,255,255,0.22)" />
  <text x="184" y="513" fill="#ffffff" font-size="28" font-weight="700" font-family="Arial, PingFang SC, Microsoft YaHei, sans-serif" text-anchor="middle">立即参与</text>
  <text x="1118" y="578" fill="rgba(255,255,255,0.65)" font-size="30" font-weight="700" font-family="Arial, PingFang SC, Microsoft YaHei, sans-serif" text-anchor="end" letter-spacing="3">${accent}</text>
</svg>`.trim();
}

const SUMMER_CAMPAIGN_BANNER_FILENAME = 'builtin-summer-launch-campaign.svg';
const SUMMER_ACTIVITY_BANNER_FILENAME = 'builtin-summer-launch-activity.svg';
const STARRY_CAMPAIGN_BANNER_FILENAME = 'builtin-starry-series-campaign.svg';
const STARRY_ACTIVITY_BANNER_FILENAME = 'builtin-starry-series-activity.svg';

export const BUILTIN_SUMMER_CAMPAIGN_BANNER_URL = `/api/v1/uploads/prizes/${SUMMER_CAMPAIGN_BANNER_FILENAME}`;
export const BUILTIN_SUMMER_ACTIVITY_BANNER_URL = `/api/v1/uploads/prizes/${SUMMER_ACTIVITY_BANNER_FILENAME}`;
export const BUILTIN_STARRY_CAMPAIGN_BANNER_URL = `/api/v1/uploads/prizes/${STARRY_CAMPAIGN_BANNER_FILENAME}`;
export const BUILTIN_STARRY_ACTIVITY_BANNER_URL = `/api/v1/uploads/prizes/${STARRY_ACTIVITY_BANNER_FILENAME}`;

const BUILTIN_BANNER_ASSETS: Record<string, BuiltinBannerAsset> = {
  [SUMMER_CAMPAIGN_BANNER_FILENAME]: {
    contentType: 'image/svg+xml',
    body: bannerSvg({
      eyebrow: 'SUMMER EVENT',
      title: '夏季开门红',
      subtitle: '登录即得参与资格，抽奖、积分、发奖链路一次打通',
      accent: 'HOT DROP',
      primary: '#7c3aed',
      secondary: '#ec4899',
      glow: '#fde68a',
    }),
  },
  [SUMMER_ACTIVITY_BANNER_FILENAME]: {
    contentType: 'image/svg+xml',
    body: bannerSvg({
      eyebrow: 'LIMITED CAMPAIGN',
      title: '夏季开门红抽奖活动',
      subtitle: '新用户登录即可参与，支持库存与概率后台配置',
      accent: 'FESTIVAL',
      primary: '#6d28d9',
      secondary: '#db2777',
      glow: '#f9a8d4',
    }),
  },
  [STARRY_CAMPAIGN_BANNER_FILENAME]: {
    contentType: 'image/svg+xml',
    body: bannerSvg({
      eyebrow: 'STAR COLLECTION',
      title: '梦幻星辰系列盲盒',
      subtitle: '集齐普通款与隐藏款，解锁限定星海奖励',
      accent: 'STAR BOX',
      primary: '#1d4ed8',
      secondary: '#6d28d9',
      glow: '#93c5fd',
    }),
  },
  [STARRY_ACTIVITY_BANNER_FILENAME]: {
    contentType: 'image/svg+xml',
    body: bannerSvg({
      eyebrow: 'SECRET UP',
      title: '梦幻星辰系列盲盒',
      subtitle: '宇宙之心限时概率提升，活动期内冲刺隐藏款',
      accent: 'UP RATE',
      primary: '#312e81',
      secondary: '#2563eb',
      glow: '#c4b5fd',
    }),
  },
};

export function getBuiltInBannerAsset(filename: string): { readonly buffer: Buffer; readonly contentType: string } | null {
  const asset = BUILTIN_BANNER_ASSETS[filename];
  if (!asset) {
    return null;
  }
  return {
    buffer: Buffer.from(asset.body, 'utf8'),
    contentType: asset.contentType,
  };
}