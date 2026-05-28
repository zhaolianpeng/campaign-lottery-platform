const DEFAULT_BASE_URL = process.env.BANNER_SYNC_BASE_URL || 'http://127.0.0.1:18100';
const ADMIN_USER = process.env.BANNER_SYNC_ADMIN_USER || process.env.ADMIN_USER || 'admin';
const ADMIN_PASSWORD = process.env.BANNER_SYNC_ADMIN_PASSWORD || process.env.ADMIN_PASSWORD || '';

const DEFAULT_BANNERS = {
  camp_launch_001: '/api/v1/uploads/prizes/builtin-summer-launch-campaign.svg',
  camp_blindbox_001: '/api/v1/uploads/prizes/builtin-starry-series-campaign.svg',
  series_starry_001: '/api/v1/uploads/prizes/builtin-starry-series-campaign.svg',
};

async function request(path, init = {}) {
  const response = await fetch(`${DEFAULT_BASE_URL}${path}`, init);
  const payload = await response.json();
  if (!response.ok || payload.code !== 'ok') {
    throw new Error(payload.message || `Request failed for ${path}`);
  }
  return payload.data;
}

async function main() {
  if (!ADMIN_PASSWORD) {
    throw new Error('Missing admin password. Set BANNER_SYNC_ADMIN_PASSWORD or ADMIN_PASSWORD.');
  }

  const login = await request('/api/v1/admin/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: ADMIN_USER, password: ADMIN_PASSWORD }),
  });

  const token = login.token;
  const campaigns = await request('/api/v1/admin/campaigns', {
    headers: { Authorization: `Bearer ${token}` },
  });

  const updated = [];
  for (const campaign of campaigns) {
    const bannerImageUrl = DEFAULT_BANNERS[campaign.id];
    if (!bannerImageUrl) {
      continue;
    }

    await request(`/api/v1/admin/campaigns/${campaign.id}`, {
      method: 'PUT',
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        name: campaign.name,
        slug: campaign.slug,
        status: campaign.status,
        starts_at: campaign.starts_at,
        ends_at: campaign.ends_at,
        daily_draw_limit: campaign.daily_draw_limit,
        requires_phone_login: campaign.requires_phone_login,
        miss_weight: campaign.miss_weight,
        banner_image_url: bannerImageUrl,
        campaign_summary: campaign.campaign_summary,
        pity_config: campaign.pity_config ?? undefined,
      }),
    });

    updated.push({ id: campaign.id, banner_image_url: bannerImageUrl });
  }

  console.log(JSON.stringify({ baseUrl: DEFAULT_BASE_URL, updated }, null, 2));
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : String(error));
  process.exitCode = 1;
});