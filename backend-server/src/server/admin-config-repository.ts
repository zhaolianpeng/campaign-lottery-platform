import type { RowDataPacket } from 'mysql2/promise';
import { getMysqlPool } from './database';
import type { AdminConfigState } from './memory-store';
import type { MemoryStore } from './memory-store';
import type { Campaign, FirstRechargePack, PackItem, PityConfig, Prize, ShopItem } from './types';

interface CampaignRow extends RowDataPacket {
  id: string;
  name: string;
  slug: string;
  status: string;
  starts_at: Date | string;
  ends_at: Date | string;
  daily_draw_limit: number;
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

async function ensureConfigTables(): Promise<void> {
  const pool = getMysqlPool();
  if (!pool) {
    return;
  }

  await pool.query(`
    CREATE TABLE IF NOT EXISTS shop_items (
      id VARCHAR(32) NOT NULL,
      name VARCHAR(128) NOT NULL,
      description VARCHAR(255) NOT NULL DEFAULT '',
      image_url VARCHAR(255) NOT NULL DEFAULT '',
      price_points INT NOT NULL DEFAULT 0,
      price_cash INT NOT NULL DEFAULT 0,
      item_type VARCHAR(32) NOT NULL,
      item_qty INT NOT NULL DEFAULT 1,
      stock INT NOT NULL DEFAULT -1,
      daily_limit INT NOT NULL DEFAULT 0,
      category VARCHAR(32) NOT NULL DEFAULT '',
      is_active TINYINT(1) NOT NULL DEFAULT 1,
      expires_at DATETIME NULL,
      sort_order INT NOT NULL DEFAULT 0,
      created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
      updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
      PRIMARY KEY (id),
      KEY idx_shop_items_sort_order (sort_order)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
  `);

  await pool.query(`
    CREATE TABLE IF NOT EXISTS first_recharge_packs (
      id VARCHAR(32) NOT NULL,
      name VARCHAR(128) NOT NULL,
      price_points INT NOT NULL DEFAULT 0,
      cash_price INT NOT NULL DEFAULT 0,
      description VARCHAR(255) NOT NULL DEFAULT '',
      image_url VARCHAR(255) NOT NULL DEFAULT '',
      sort_order INT NOT NULL DEFAULT 0,
      items_json JSON NOT NULL,
      created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
      updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
      PRIMARY KEY (id),
      KEY idx_first_recharge_packs_sort_order (sort_order)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
  `);

  const [prizeImageColumns] = await pool.query<RowDataPacket[]>("SHOW COLUMNS FROM prizes LIKE 'image_url'");
  if (prizeImageColumns.length === 0) {
    await pool.query("ALTER TABLE prizes ADD COLUMN image_url VARCHAR(255) NOT NULL DEFAULT '' AFTER status");
  }

  const [shopImageColumns] = await pool.query<RowDataPacket[]>("SHOW COLUMNS FROM shop_items LIKE 'image_url'");
  if (shopImageColumns.length === 0) {
    await pool.query("ALTER TABLE shop_items ADD COLUMN image_url VARCHAR(255) NOT NULL DEFAULT '' AFTER description");
  }

  const [packImageColumns] = await pool.query<RowDataPacket[]>("SHOW COLUMNS FROM first_recharge_packs LIKE 'image_url'");
  if (packImageColumns.length === 0) {
    await pool.query("ALTER TABLE first_recharge_packs ADD COLUMN image_url VARCHAR(255) NOT NULL DEFAULT '' AFTER description");
  }
}

async function loadCampaignState(): Promise<Pick<AdminConfigState, 'campaigns' | 'prizesByCampaign'> | null> {
  const pool = getMysqlPool();
  if (!pool) {
    return null;
  }

  const [campaignRows] = await pool.query<CampaignRow[]>(`
    SELECT id, name, slug, status, starts_at, ends_at, daily_draw_limit, miss_weight, banner_image_url, campaign_summary, pity_config
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
}

export async function syncAdminConfigWithMysql(store: MemoryStore): Promise<void> {
  const pool = getMysqlPool();
  if (!pool) {
    return;
  }

  await ensureConfigTables();
  await bootstrapIfEmpty(store);

  const [campaignState, shopItems, firstRechargePacks] = await Promise.all([
    loadCampaignState(),
    loadShopItems(),
    loadFirstRechargePacks(),
  ]);

  store.hydrateAdminConfigState({
    campaigns: campaignState?.campaigns,
    prizesByCampaign: campaignState?.prizesByCampaign,
    shopItems: shopItems ?? undefined,
    firstRechargePacks: firstRechargePacks ?? undefined,
  });
}

export async function upsertCampaign(campaign: Campaign): Promise<void> {
  const pool = getMysqlPool();
  if (!pool) {
    return;
  }
  await ensureConfigTables();
  await pool.query(
    `INSERT INTO campaigns (id, name, slug, status, starts_at, ends_at, daily_draw_limit, miss_weight, banner_image_url, campaign_summary, pity_config, created_at, updated_at)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())
     ON DUPLICATE KEY UPDATE
       name = VALUES(name),
       slug = VALUES(slug),
       status = VALUES(status),
       starts_at = VALUES(starts_at),
       ends_at = VALUES(ends_at),
       daily_draw_limit = VALUES(daily_draw_limit),
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
  await ensureConfigTables();
  await pool.query('DELETE FROM prizes WHERE campaign_id = ?', [campaignId]);
  await pool.query('DELETE FROM campaigns WHERE id = ?', [campaignId]);
}

export async function upsertPrize(prize: Prize): Promise<void> {
  const pool = getMysqlPool();
  if (!pool) {
    return;
  }
  await ensureConfigTables();
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
  await ensureConfigTables();
  await pool.query('DELETE FROM prizes WHERE id = ?', [prizeId]);
}

export async function upsertShopItem(item: ShopItem): Promise<void> {
  const pool = getMysqlPool();
  if (!pool) {
    return;
  }
  await ensureConfigTables();
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
  await ensureConfigTables();
  await pool.query('DELETE FROM shop_items WHERE id = ?', [itemId]);
}

export async function upsertFirstRechargePack(pack: FirstRechargePack): Promise<void> {
  const pool = getMysqlPool();
  if (!pool) {
    return;
  }
  await ensureConfigTables();
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
  await ensureConfigTables();
  await pool.query('DELETE FROM first_recharge_packs WHERE id = ?', [packId]);
}
