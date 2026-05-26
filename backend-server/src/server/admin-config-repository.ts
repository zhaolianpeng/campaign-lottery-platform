import type { RowDataPacket } from 'mysql2/promise';
import { getMysqlPool } from './database';
import type { AdminConfigState } from './memory-store';
import type { MemoryStore } from './memory-store';
import type { Campaign, CEndFeatureToggles, FirstRechargePack, PackItem, PendingAnonymousWin, PityConfig, Prize, ShopItem } from './types';

const C_END_FEATURE_TOGGLES_KEY = 'c_end_feature_toggles';

interface CampaignRow extends RowDataPacket {
  id: string;
  name: string;
  slug: string;
  status: string;
  starts_at: Date | string;
  ends_at: Date | string;
  daily_draw_limit: number;
  requires_phone_login: number;
  miss_weight: number;
  banner_image_url: string;
  campaign_summary: string;
  pity_config: string | null;
}

interface PrizeRow extends RowDataPacket {
  id: string;
  campaign_id: string;
  name: string;
  level: string;
  stock: number;
  probability_weight: number;
  status: string;
  image_url: string;
}

interface ShopItemRow extends RowDataPacket {
  id: string;
  name: string;
  description: string;
  image_url: string;
  price_points: number;
  price_cash: number;
  item_type: string;
  item_qty: number;
  stock: number;
  daily_limit: number;
  category: string;
  is_active: number;
  expires_at: Date | string | null;
  sort_order: number;
}

interface FirstRechargePackRow extends RowDataPacket {
  id: string;
  name: string;
  price_points: number;
  cash_price: number;
  description: string;
  image_url: string;
  sort_order: number;
  items_json: string | readonly PackItem[];
}

interface AppSettingRow extends RowDataPacket {
  setting_key: string;
  setting_value: string;
}

interface PendingAnonymousWinRow extends RowDataPacket {
  id: string;
  anonymous_token: string;
  campaign_id: string;
  prize_id: string;
  prize_name: string;
  prize_level: string;
  prize_image_url: string | null;
  drawn_at: Date | string;
}

function toIsoString(value: Date | string | null | undefined): string | undefined {
  if (!value) {
    return undefined;
  }
  return value instanceof Date ? value.toISOString() : new Date(value).toISOString();
}

function parseJsonValue<T>(raw: string | T | null | undefined, fallback: T): T {
  if (raw == null) {
    return fallback;
  }
  if (typeof raw !== 'string') {
    return raw;
  }
  try {
    return JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
}

function parsePackItems(raw: string | readonly PackItem[]): readonly PackItem[] {
  try {
    const parsed = parseJsonValue(raw, [] as PackItem[]);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function normalizePityConfig(raw: PityConfig | null | undefined): PityConfig | undefined {
  if (!raw) {
    return undefined;
  }
  return {
    ...raw,
    target_prize: raw.target_prize?.trim() ?? '',
    up_prize_id: raw.up_prize_id?.trim() || undefined,
    up_level: raw.up_level || undefined,
    up_start_at: raw.up_start_at || undefined,
    up_end_at: raw.up_end_at || undefined,
  };
}

function defaultCEndFeatureToggles(): CEndFeatureToggles {
  return {
    series: true,
    inventory: true,
    exchange: true,
    rank: true,
    member: true,
    shop: true,
    social: true,
    puzzle: true,
  };
}

function normalizeCEndFeatureToggles(raw: Partial<CEndFeatureToggles> | null | undefined): CEndFeatureToggles {
  const defaults = defaultCEndFeatureToggles();
  return {
    series: raw?.series ?? defaults.series,
    inventory: raw?.inventory ?? defaults.inventory,
    exchange: raw?.exchange ?? defaults.exchange,
    rank: raw?.rank ?? defaults.rank,
    member: raw?.member ?? defaults.member,
    shop: raw?.shop ?? defaults.shop,
    social: raw?.social ?? defaults.social,
    puzzle: raw?.puzzle ?? defaults.puzzle,
  };
}

async function loadCampaignState(): Promise<Pick<AdminConfigState, 'campaigns' | 'prizesByCampaign'> | null> {
  const pool = getMysqlPool();
  if (!pool) {
    return null;
  }

  const [campaignRows] = await pool.query<CampaignRow[]>(`
    SELECT id, name, slug, status, starts_at, ends_at, daily_draw_limit, requires_phone_login, miss_weight, banner_image_url, campaign_summary, pity_config
    FROM campaigns
    ORDER BY created_at ASC
  `);

  if (campaignRows.length === 0) {
    return null;
  }

  const [prizeRows] = await pool.query<PrizeRow[]>(`
    SELECT id, campaign_id, name, level, stock, probability_weight, status, image_url
    FROM prizes
    ORDER BY campaign_id ASC, created_at ASC
  `);

  const campaigns: Campaign[] = campaignRows.map((row) => ({
    id: row.id,
    name: row.name,
    slug: row.slug,
    status: row.status as Campaign['status'],
    starts_at: new Date(row.starts_at).toISOString(),
    ends_at: new Date(row.ends_at).toISOString(),
    daily_draw_limit: row.daily_draw_limit,
    requires_phone_login: Boolean(row.requires_phone_login),
    miss_weight: row.miss_weight,
    banner_image_url: row.banner_image_url,
    campaign_summary: row.campaign_summary,
    pity_config: normalizePityConfig(row.pity_config ? parseJsonValue(row.pity_config, undefined) : undefined),
  }));

  const prizesByCampaign: Record<string, Prize[]> = {};
  for (const row of prizeRows) {
    if (!prizesByCampaign[row.campaign_id]) {
      prizesByCampaign[row.campaign_id] = [];
    }
    prizesByCampaign[row.campaign_id].push({
      id: row.id,
      campaign_id: row.campaign_id,
      name: row.name,
      level: row.level as Prize['level'],
      stock: row.stock,
      probability_weight: row.probability_weight,
      status: row.status as Prize['status'],
      image_url: row.image_url || undefined,
    });
  }

  return { campaigns, prizesByCampaign };
}

async function loadShopItems(): Promise<readonly ShopItem[] | null> {
  const pool = getMysqlPool();
  if (!pool) {
    return null;
  }

  const [rows] = await pool.query<ShopItemRow[]>(`
    SELECT id, name, description, image_url, price_points, price_cash, item_type, item_qty, stock, daily_limit, category, is_active, expires_at, sort_order
    FROM shop_items
    ORDER BY sort_order ASC, created_at ASC
  `);

  if (rows.length === 0) {
    return null;
  }

  return rows.map((row) => ({
    id: row.id,
    name: row.name,
    description: row.description,
    image_url: row.image_url || undefined,
    price_points: row.price_points,
    price_cash: row.price_cash,
    item_type: row.item_type as ShopItem['item_type'],
    item_qty: row.item_qty,
    stock: row.stock,
    daily_limit: row.daily_limit,
    category: row.category,
    is_active: Boolean(row.is_active),
    expires_at: toIsoString(row.expires_at),
    sort_order: row.sort_order,
  }));
}

async function loadFirstRechargePacks(): Promise<readonly FirstRechargePack[] | null> {
  const pool = getMysqlPool();
  if (!pool) {
    return null;
  }

  const [rows] = await pool.query<FirstRechargePackRow[]>(`
    SELECT id, name, price_points, cash_price, description, image_url, sort_order, items_json
    FROM first_recharge_packs
    ORDER BY sort_order ASC, created_at ASC
  `);

  if (rows.length === 0) {
    return null;
  }

  return rows.map((row) => ({
    id: row.id,
    name: row.name,
    price_points: row.price_points,
    cash_price: row.cash_price,
    description: row.description,
    image_url: row.image_url || undefined,
    sort_order: row.sort_order,
    items: parsePackItems(row.items_json),
  }));
}

async function loadCEndFeatureToggles(): Promise<CEndFeatureToggles | null> {
  const pool = getMysqlPool();
  if (!pool) {
    return null;
  }

  const [rows] = await pool.query<AppSettingRow[]>(
    'SELECT setting_key, setting_value FROM app_settings WHERE setting_key = ? LIMIT 1',
    [C_END_FEATURE_TOGGLES_KEY],
  );

  if (rows.length === 0) {
    return null;
  }

  return normalizeCEndFeatureToggles(parseJsonValue<Partial<CEndFeatureToggles>>(rows[0].setting_value, defaultCEndFeatureToggles()));
}

async function loadPendingAnonymousWins(): Promise<Readonly<Record<string, readonly PendingAnonymousWin[]>>> {
  const pool = getMysqlPool();
  if (!pool) {
    return {};
  }

  const [rows] = await pool.query<PendingAnonymousWinRow[]>(`
    SELECT id, anonymous_token, campaign_id, prize_id, prize_name, prize_level, prize_image_url, drawn_at
    FROM pending_anonymous_wins
    ORDER BY drawn_at ASC, id ASC
  `);

  const entriesByToken: Record<string, PendingAnonymousWin[]> = {};
  for (const row of rows) {
    if (!entriesByToken[row.anonymous_token]) {
      entriesByToken[row.anonymous_token] = [];
    }
    entriesByToken[row.anonymous_token].push({
      id: row.id,
      campaign_id: row.campaign_id,
      prize_id: row.prize_id,
      prize_name: row.prize_name,
      prize_level: row.prize_level as PendingAnonymousWin['prize_level'],
      prize_image_url: row.prize_image_url || undefined,
      drawn_at: new Date(row.drawn_at).toISOString(),
    });
  }

  return entriesByToken;
}

async function bootstrapIfEmpty(store: MemoryStore): Promise<void> {
  const pool = getMysqlPool();
  if (!pool) {
    return;
  }

  const state = store.exportAdminConfigState();
  const [campaignRows] = await pool.query<RowDataPacket[]>('SELECT id FROM campaigns LIMIT 1');
  if (campaignRows.length === 0) {
    for (const campaign of state.campaigns) {
      await upsertCampaign(campaign);
      for (const prize of state.prizesByCampaign[campaign.id] ?? []) {
        await upsertPrize(prize);
      }
    }
  }

  const [shopRows] = await pool.query<RowDataPacket[]>('SELECT id FROM shop_items LIMIT 1');
  if (shopRows.length === 0) {
    for (const item of state.shopItems) {
      await upsertShopItem(item);
    }
  }

  const [packRows] = await pool.query<RowDataPacket[]>('SELECT id FROM first_recharge_packs LIMIT 1');
  if (packRows.length === 0) {
    for (const pack of state.firstRechargePacks) {
      await upsertFirstRechargePack(pack);
    }
  }

  const [settingRows] = await pool.query<RowDataPacket[]>('SELECT setting_key FROM app_settings WHERE setting_key = ? LIMIT 1', [C_END_FEATURE_TOGGLES_KEY]);
  if (settingRows.length === 0) {
    await upsertCEndFeatureToggles(state.cEndFeatureToggles);
  }
}

export async function syncAdminConfigWithMysql(store: MemoryStore): Promise<void> {
  const pool = getMysqlPool();
  if (!pool) {
    return;
  }

  await bootstrapIfEmpty(store);

  const [campaignState, shopItems, firstRechargePacks, cEndFeatureToggles, pendingAnonymousWins] = await Promise.all([
    loadCampaignState(),
    loadShopItems(),
    loadFirstRechargePacks(),
    loadCEndFeatureToggles(),
    loadPendingAnonymousWins(),
  ]);

  store.hydrateAdminConfigState({
    campaigns: campaignState?.campaigns,
    prizesByCampaign: campaignState?.prizesByCampaign,
    shopItems: shopItems ?? undefined,
    firstRechargePacks: firstRechargePacks ?? undefined,
    cEndFeatureToggles: cEndFeatureToggles ?? undefined,
  });
  store.hydratePendingAnonymousWins(pendingAnonymousWins);
}

export async function upsertCEndFeatureToggles(toggles: CEndFeatureToggles): Promise<void> {
  const pool = getMysqlPool();
  if (!pool) {
    return;
  }

  await pool.query(
    `INSERT INTO app_settings (setting_key, setting_value, created_at, updated_at)
     VALUES (?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())
     ON DUPLICATE KEY UPDATE
       setting_value = VALUES(setting_value),
       updated_at = UTC_TIMESTAMP()`,
    [C_END_FEATURE_TOGGLES_KEY, JSON.stringify(normalizeCEndFeatureToggles(toggles))],
  );
}

export async function replacePendingAnonymousWins(anonymousToken: string, wins: readonly PendingAnonymousWin[]): Promise<void> {
  const pool = getMysqlPool();
  if (!pool || !anonymousToken.trim()) {
    return;
  }

  const connection = await pool.getConnection();
  try {
    await connection.beginTransaction();
    await connection.query('DELETE FROM pending_anonymous_wins WHERE anonymous_token = ?', [anonymousToken]);
    for (const win of wins) {
      await connection.query(
        `INSERT INTO pending_anonymous_wins (id, anonymous_token, campaign_id, prize_id, prize_name, prize_level, prize_image_url, drawn_at, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())`,
        [
          win.id,
          anonymousToken,
          win.campaign_id,
          win.prize_id,
          win.prize_name,
          win.prize_level,
          win.prize_image_url ?? null,
          win.drawn_at.slice(0, 19).replace('T', ' '),
        ],
      );
    }
    await connection.commit();
  } catch (error) {
    await connection.rollback();
    throw error;
  } finally {
    connection.release();
  }
}

export async function deletePendingAnonymousWins(anonymousToken: string): Promise<void> {
  const pool = getMysqlPool();
  if (!pool || !anonymousToken.trim()) {
    return;
  }
  await pool.query('DELETE FROM pending_anonymous_wins WHERE anonymous_token = ?', [anonymousToken]);
}

export async function upsertCampaign(campaign: Campaign): Promise<void> {
  const pool = getMysqlPool();
  if (!pool) {
    return;
  }
  await pool.query(
    `INSERT INTO campaigns (id, name, slug, status, starts_at, ends_at, daily_draw_limit, requires_phone_login, miss_weight, banner_image_url, campaign_summary, pity_config, created_at, updated_at)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())
     ON DUPLICATE KEY UPDATE
       name = VALUES(name),
       slug = VALUES(slug),
       status = VALUES(status),
       starts_at = VALUES(starts_at),
       ends_at = VALUES(ends_at),
       daily_draw_limit = VALUES(daily_draw_limit),
       requires_phone_login = VALUES(requires_phone_login),
       miss_weight = VALUES(miss_weight),
       banner_image_url = VALUES(banner_image_url),
       campaign_summary = VALUES(campaign_summary),
       pity_config = VALUES(pity_config),
       updated_at = UTC_TIMESTAMP()`,
    [
      campaign.id,
      campaign.name,
      campaign.slug,
      campaign.status,
      campaign.starts_at.slice(0, 19).replace('T', ' '),
      campaign.ends_at.slice(0, 19).replace('T', ' '),
      campaign.daily_draw_limit,
      campaign.requires_phone_login ? 1 : 0,
      campaign.miss_weight,
      campaign.banner_image_url,
      campaign.campaign_summary,
      campaign.pity_config ? JSON.stringify(normalizePityConfig(campaign.pity_config)) : null,
    ],
  );
}

export async function deleteCampaignConfig(campaignId: string): Promise<void> {
  const pool = getMysqlPool();
  if (!pool) {
    return;
  }
  await pool.query('DELETE FROM prizes WHERE campaign_id = ?', [campaignId]);
  await pool.query('DELETE FROM campaigns WHERE id = ?', [campaignId]);
}

export async function upsertPrize(prize: Prize): Promise<void> {
  const pool = getMysqlPool();
  if (!pool) {
    return;
  }
  await pool.query(
    `INSERT INTO prizes (id, campaign_id, name, level, stock, probability_weight, status, image_url, created_at, updated_at)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())
     ON DUPLICATE KEY UPDATE
       campaign_id = VALUES(campaign_id),
       name = VALUES(name),
       level = VALUES(level),
       stock = VALUES(stock),
       probability_weight = VALUES(probability_weight),
       status = VALUES(status),
       image_url = VALUES(image_url),
       updated_at = UTC_TIMESTAMP()`,
    [
      prize.id,
      prize.campaign_id,
      prize.name,
      prize.level,
      prize.stock,
      prize.probability_weight,
      prize.status,
      prize.image_url ?? '',
    ],
  );
}

export async function deletePrizeConfig(prizeId: string): Promise<void> {
  const pool = getMysqlPool();
  if (!pool) {
    return;
  }
  await pool.query('DELETE FROM prizes WHERE id = ?', [prizeId]);
}

export async function upsertShopItem(item: ShopItem): Promise<void> {
  const pool = getMysqlPool();
  if (!pool) {
    return;
  }
  await pool.query(
    `INSERT INTO shop_items (id, name, description, image_url, price_points, price_cash, item_type, item_qty, stock, daily_limit, category, is_active, expires_at, sort_order)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
     ON DUPLICATE KEY UPDATE
       name = VALUES(name),
       description = VALUES(description),
       image_url = VALUES(image_url),
       price_points = VALUES(price_points),
       price_cash = VALUES(price_cash),
       item_type = VALUES(item_type),
       item_qty = VALUES(item_qty),
       stock = VALUES(stock),
       daily_limit = VALUES(daily_limit),
       category = VALUES(category),
       is_active = VALUES(is_active),
       expires_at = VALUES(expires_at),
       sort_order = VALUES(sort_order),
       updated_at = UTC_TIMESTAMP()`,
    [
      item.id,
      item.name,
      item.description,
      item.image_url ?? '',
      item.price_points,
      item.price_cash,
      item.item_type,
      item.item_qty,
      item.stock,
      item.daily_limit,
      item.category,
      item.is_active ? 1 : 0,
      item.expires_at ? item.expires_at.slice(0, 19).replace('T', ' ') : null,
      item.sort_order,
    ],
  );
}

export async function deleteShopItemConfig(itemId: string): Promise<void> {
  const pool = getMysqlPool();
  if (!pool) {
    return;
  }
  await pool.query('DELETE FROM shop_items WHERE id = ?', [itemId]);
}

export async function upsertFirstRechargePack(pack: FirstRechargePack): Promise<void> {
  const pool = getMysqlPool();
  if (!pool) {
    return;
  }
  await pool.query(
    `INSERT INTO first_recharge_packs (id, name, price_points, cash_price, description, image_url, sort_order, items_json)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?)
     ON DUPLICATE KEY UPDATE
       name = VALUES(name),
       price_points = VALUES(price_points),
       cash_price = VALUES(cash_price),
       description = VALUES(description),
       image_url = VALUES(image_url),
       sort_order = VALUES(sort_order),
       items_json = VALUES(items_json),
       updated_at = UTC_TIMESTAMP()`,
    [pack.id, pack.name, pack.price_points, pack.cash_price, pack.description, pack.image_url ?? '', pack.sort_order, JSON.stringify(pack.items)],
  );
}

export async function deleteFirstRechargePackConfig(packId: string): Promise<void> {
  const pool = getMysqlPool();
  if (!pool) {
    return;
  }
  await pool.query('DELETE FROM first_recharge_packs WHERE id = ?', [packId]);
}
