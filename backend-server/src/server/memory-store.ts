import { randomUUID } from 'node:crypto';
import { syncAdminConfigWithMysql } from './admin-config-repository';
import { getAppConfig } from './config';
import {
  AppError,
  adminUnauthorized,
  badAdminAuth,
  campaignNotFound,
  insufficientPoints,
  noDrawChances,
  notFound,
  phoneAlreadyBound,
  phoneCodeInvalid,
  unauthorized,
  userStatusForbidden,
} from './errors';
import type {
  AdminUserDetail,
  AdminUserListResult,
  AdminOverview,
  Activity,
  ActivityListInfo,
  ActivityParticipation,
  ActivityReward,
  AssistClaimResult,
  AssistProgress,
  AssistType,
  BattlePass,
  BattlePassInfo,
  BattlePassReward,
  BattlePassSeason,
  BattlePassTask,
  BattlePassTaskProgress,
  BlendRequest,
  BlendResult,
  BuyCardRequest,
  BuyCardResult,
  BuyShopItemRequest,
  BuyShopItemResult,
  Campaign,
  CampaignMutation,
  CampaignPublishValidation,
  CardType,
  CEndFeatureToggles,
  CheckInResult,
  ClaimActivityRewardRequest,
  ClaimFirstRechargeRequest,
  ClaimFirstRechargeResult,
  CollectedPrize,
  ComposePuzzleResult,
  CreateTeamRequest,
  DrawRecord,
  DrawStatistics,
  ExchangeOffer,
  ExchangeOfferMutation,
  FirstRechargePack,
  FirstRechargePackMutation,
  FlashListInfo,
  FlashPurchaseResult,
  FlashSale,
  FlashSubscription,
  FulfillmentTask,
  FulfillmentTaskMutation,
  GiftRecord,
  HintMessage,
  InviteRecord,
  InviteStats,
  ItemType,
  LeaderboardEntry,
  MonthCardPurchaseResult,
  MonthCardStatus,
  PendingAnonymousWin,
  PityConfig,
  Prize,
  PrizeMutation,
  PrizeSummary,
  PuzzleInfo,
  PuzzleProgress,
  PuzzleTeam,
  PuzzleTemplate,
  RedeemRequest,
  RedeemResult,
  ReceiveGiftResult,
  SeriesProgress,
  SendGiftRequest,
  ShareCard,
  Session,
  ShareRewardResult,
  ShopItem,
  ShopItemMutation,
  Team,
  TeamInfo,
  TeamMember,
  TeamReward,
  User,
  UserAccount,
  UserCard,
  UserFirstRecharge,
  UserInventory,
  UserItem,
  UserMember,
  UserLoginLog,
  UserPointsLog,
  UserProfile,
  UserProfileMutation,
  UserStatus,
  UserStatusLog,
  UseItemRequest,
  WechatUser,
} from './types';

const POINTS_PER_DRAW = 100;
const PHONE_CODE_TTL_MS = 5 * 60 * 1000;

interface PhoneVerificationCode {
  readonly id: number;
  readonly phone: string;
  readonly scene: string;
  readonly code: string;
  readonly expires_at: string;
  readonly created_at: string;
  readonly verified_at?: string;
  readonly attempt_count: number;
}

interface UserWallet {
  readonly user_id: string;
  readonly cash_balance: number;
  readonly frozen_balance: number;
  readonly currency: string;
  readonly created_at: string;
  readonly updated_at: string;
}

const cardConfigs: Record<CardType, { readonly price: number; readonly durationDays: number; readonly freeDraws: number; readonly discount: number }> = {
  weekly: { price: 990, durationDays: 7, freeDraws: 1, discount: 0.9 },
  monthly: { price: 2800, durationDays: 30, freeDraws: 2, discount: 0.8 },
  season: { price: 6800, durationDays: 90, freeDraws: 2, discount: 0.75 },
};

const defaultFirstRechargePacks: readonly FirstRechargePack[] = [
  {
    id: 'tier_1',
    name: '首充6元',
    price_points: 600,
    cash_price: 600,
    items: [
      { type: 'points', name: '积分', qty: 60 },
      { type: 'hint_card', name: '提示卡', qty: 3 },
    ],
    description: '60积分+提示卡x3',
    sort_order: 1,
  },
  {
    id: 'tier_2',
    name: '首充30元',
    price_points: 3000,
    cash_price: 3000,
    items: [
      { type: 'points', name: '积分', qty: 300 },
      { type: 'ten_draw_ticket', name: '十连券', qty: 1 },
      { type: 'see_through', name: '透卡', qty: 5 },
    ],
    description: '300积分+十连券x1+透卡x5',
    sort_order: 2,
  },
  {
    id: 'tier_3',
    name: '首充98元',
    price_points: 9800,
    cash_price: 9800,
    items: [
      { type: 'points', name: '积分', qty: 980 },
      { type: 'ten_draw_ticket', name: '十连券', qty: 2 },
      { type: 'free_draw', name: '免费抽券', qty: 5 },
    ],
    description: '980积分+十连券x2+免费抽券x5',
    sort_order: 3,
  },
];

function nowISO(): string {
  return new Date().toISOString();
}

function randomId(prefix: string): string {
  return `${prefix}_${randomUUID().replaceAll('-', '').slice(0, 16)}`;
}

function todayKey(): string {
  return new Date().toISOString().slice(0, 10);
}

function addDays(days: number): string {
  return new Date(Date.now() + days * 24 * 60 * 60 * 1000).toISOString();
}

function hoursUntil(value: string): number {
  return Math.max(0, Math.ceil((new Date(value).getTime() - Date.now()) / (60 * 60 * 1000)));
}

function cloneCampaign(campaign: Campaign): Campaign {
  return {
    ...campaign,
    pity_config: campaign.pity_config ? { ...campaign.pity_config } : undefined,
  };
}

function cloneShopItem(item: ShopItem): ShopItem {
  return { ...item };
}

function clonePackItem(item: FirstRechargePack['items'][number]): FirstRechargePack['items'][number] {
  return { ...item };
}

function cloneFirstRechargePack(pack: FirstRechargePack): FirstRechargePack {
  return {
    ...pack,
    items: pack.items.map((item) => clonePackItem(item)),
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

function cloneCEndFeatureToggles(toggles: CEndFeatureToggles): CEndFeatureToggles {
  return { ...toggles };
}

export interface AdminConfigState {
  readonly campaigns: readonly Campaign[];
  readonly prizesByCampaign: Readonly<Record<string, readonly Prize[]>>;
  readonly shopItems: readonly ShopItem[];
  readonly firstRechargePacks: readonly FirstRechargePack[];
  readonly cEndFeatureToggles: CEndFeatureToggles;
}

function memberLevel(totalSpent: number): UserMember['level'] {
  if (totalSpent >= 50000) {
    return 'diamond';
  }
  if (totalSpent >= 20000) {
    return 'gold';
  }
  if (totalSpent >= 5000) {
    return 'silver';
  }
  return 'normal';
}

export class MemoryStore {
  private readonly users = new Map<string, User>();
  private readonly userProfiles = new Map<string, UserProfile>();
  private readonly sessions = new Map<string, Session>();
  private readonly adminSessions = new Map<string, string>();
  private readonly campaignsById = new Map<string, Campaign>();
  private readonly prizesByCampaign = new Map<string, Prize[]>();
  private readonly drawRecords: DrawRecord[] = [];
  private readonly quotaUsed = new Map<string, number>();
  private readonly inventory: UserInventory[] = [];
  private readonly exchangeOffers: ExchangeOffer[] = [];
  private readonly members = new Map<string, UserMember>();
  private readonly pointsLogs: UserPointsLog[] = [];
  private readonly wallets = new Map<string, UserWallet>();
  private readonly loginLogs: UserLoginLog[] = [];
  private readonly statusLogs: UserStatusLog[] = [];
  private readonly phoneCodes: PhoneVerificationCode[] = [];
  private readonly fulfillmentTaskItems: FulfillmentTask[] = [];
  private readonly checkInDates = new Map<string, string>();
  private readonly checkInStreaks = new Map<string, number>();
  private readonly shareCounts = new Map<string, number>();
  private readonly userCards = new Map<string, UserCard>();
  private readonly userItems = new Map<string, UserItem>();
  private readonly wechatUsers = new Map<string, WechatUser>();
  private readonly wechatByOpenid = new Map<string, WechatUser>();
  private readonly wechatSessionKeys = new Map<string, string>();
  private readonly phoneByNumber = new Map<string, string>();
  private readonly firstRechargeClaims = new Map<string, UserFirstRecharge>();
  private readonly battlePasses = new Map<string, BattlePass>();
  private readonly taskProgress = new Map<string, BattlePassTaskProgress>();
  private readonly shareCards: ShareCard[] = [];
  private readonly inviteRecords: InviteRecord[] = [];
  private readonly assistProgress = new Map<string, AssistProgress>();
  private readonly teams = new Map<string, Team>();
  private readonly teamMembers: TeamMember[] = [];
  private readonly gifts: GiftRecord[] = [];
  private readonly puzzleProgress = new Map<string, PuzzleProgress>();
  private readonly puzzleTeams = new Map<string, PuzzleTeam>();
  private readonly flashSubscriptions: FlashSubscription[] = [];
  private readonly activityParticipations: ActivityParticipation[] = [];
  private readonly pendingAnonymousWins = new Map<string, PendingAnonymousWin[]>();
  private shopItemList: ShopItem[] = [];
  private firstRechargePackList: FirstRechargePack[] = [];
  private battlePassSeason: BattlePassSeason | null = null;
  private battlePassTasks: BattlePassTask[] = [];
  private battlePassRewards: BattlePassReward[] = [];
  private puzzleTemplates: PuzzleTemplate[] = [];
  private flashSales: FlashSale[] = [];
  private activities: Activity[] = [];
  private activityRewards: ActivityReward[] = [];
  private cEndFeatureToggleState: CEndFeatureToggles = defaultCEndFeatureToggles();

  public constructor(
    private readonly adminUser: string,
    private readonly adminPassword: string,
  ) {
    this.seed();
  }

  public createGuestSession(nickname: string): { readonly user: User; readonly session: Session } {
    const createdAt = nowISO();
    const user: User = {
      id: randomId('usr'),
      nickname: nickname.trim() || `Guest${Math.floor(Math.random() * 10000)}`,
      status: 'active',
      register_source: 'guest',
      created_at: createdAt,
      updated_at: createdAt,
      last_login_at: createdAt,
    };
    const session: Session = {
      token: randomId('utk'),
      user_id: user.id,
      expires_at: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
      session_type: 'normal',
    };

    this.users.set(user.id, user);
    this.sessions.set(session.token, session);
    this.userProfiles.set(user.id, {
      user_id: user.id,
      gender: 'unknown',
      created_at: createdAt,
      updated_at: createdAt,
    });
    this.members.set(user.id, {
      user_id: user.id,
      level: 'normal',
      points: 1000,
      total_draws: 0,
      total_spent: 0,
      created_at: createdAt,
      updated_at: createdAt,
    });
    this.wallets.set(user.id, {
      user_id: user.id,
      cash_balance: 0,
      frozen_balance: 0,
      currency: 'CNY',
      created_at: createdAt,
      updated_at: createdAt,
    });
    this.logPoints(user.id, 1000, 1000, 'welcome', '新用户注册赠送');
    this.logLogin(user.id, 'guest', user.nickname, true);

    return { user, session };
  }

  public createWechatUser(openid: string, phone: string, nickname: string, avatar: string): { readonly user: User; readonly session: Session; readonly wechatUser: WechatUser } {
    const createdAt = nowISO();
    const mobile = phone || undefined;
    const user: User = {
      id: randomId('usr'),
      nickname: nickname.trim() || `WeChat${Math.floor(Math.random() * 10000)}`,
      mobile,
      phone: mobile,
      avatar_url: avatar || undefined,
      status: mobile ? 'active' : 'pending_phone',
      register_source: 'wechat',
      mobile_verified_at: mobile ? createdAt : undefined,
      last_login_at: createdAt,
      created_at: createdAt,
      updated_at: createdAt,
    };
    const session: Session = {
      token: randomId('utk'),
      user_id: user.id,
      expires_at: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
      session_type: mobile ? 'normal' : 'limited',
    };
    const wechatUser: WechatUser = {
      openid,
      phone: mobile,
      nickname: nickname || undefined,
      avatar: avatar || undefined,
      user_id: user.id,
      created_at: createdAt,
    };

    this.users.set(user.id, user);
    this.sessions.set(session.token, session);
    this.wechatUsers.set(user.id, wechatUser);
    this.wechatByOpenid.set(openid, wechatUser);
    if (mobile) {
      this.phoneByNumber.set(mobile, user.id);
    }
    this.userProfiles.set(user.id, {
      user_id: user.id,
      gender: 'unknown',
      created_at: createdAt,
      updated_at: createdAt,
    });
    this.members.set(user.id, {
      user_id: user.id,
      level: 'normal',
      points: 1000,
      total_draws: 0,
      total_spent: 0,
      created_at: createdAt,
      updated_at: createdAt,
    });
    this.wallets.set(user.id, {
      user_id: user.id,
      cash_balance: 0,
      frozen_balance: 0,
      currency: 'CNY',
      created_at: createdAt,
      updated_at: createdAt,
    });
    this.logPoints(user.id, 1000, 1000, 'welcome', '新用户注册赠送');
    this.logLogin(user.id, 'wechat', openid.slice(-6), true);

    return { user, session, wechatUser };
  }

  public findWechatUserByOpenid(openid: string): WechatUser | undefined {
    return this.wechatByOpenid.get(openid);
  }

  public createSession(userId: string): Session {
    const user = this.users.get(userId);
    const session: Session = {
      token: randomId('utk'),
      user_id: userId,
      expires_at: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
      session_type: user?.status === 'pending_phone' ? 'limited' : 'normal',
    };
    this.sessions.set(session.token, session);
    if (user) {
      this.users.set(userId, { ...user, last_login_at: nowISO(), updated_at: nowISO() });
    }
    return session;
  }

  public storeWechatSessionKey(openid: string, sessionKey: string): void {
    this.wechatSessionKeys.set(openid, sessionKey);
  }

  public getWechatSessionKey(openid: string): string | undefined {
    return this.wechatSessionKeys.get(openid);
  }

  public bindWechatPhone(openid: string, phone: string): User {
    const wechatUser = this.wechatByOpenid.get(openid);
    if (!wechatUser) {
      throw unauthorized;
    }
    const existingUserId = this.phoneByNumber.get(phone);
    if (existingUserId && existingUserId !== wechatUser.user_id) {
      throw phoneAlreadyBound;
    }
    const user = this.users.get(wechatUser.user_id);
    if (!user) {
      throw unauthorized;
    }
    const now = nowISO();
    const fromStatus = (user.status ?? 'active') as UserStatus;
    const updatedUser: User = {
      ...user,
      mobile: phone,
      phone,
      status: 'active',
      mobile_verified_at: user.mobile_verified_at ?? now,
      updated_at: now,
    };
    const updatedWechat: WechatUser = { ...wechatUser, phone };
    this.users.set(user.id, updatedUser);
    this.phoneByNumber.set(phone, user.id);
    this.wechatByOpenid.set(openid, updatedWechat);
    this.wechatUsers.set(wechatUser.user_id, updatedWechat);
    if (fromStatus !== 'active') {
      this.logStatus(user.id, fromStatus, 'active', '手机号验证完成', 'system');
    }
    return updatedUser;
  }

  public findUserByPhone(phone: string): User | undefined {
    const userId = this.phoneByNumber.get(phone);
    if (!userId) return undefined;
    return this.users.get(userId);
  }

  public createPhoneUser(phone: string, nickname?: string): { readonly user: User; readonly session: Session } {
    const createdAt = nowISO();
    const user: User = {
      id: randomId('usr'),
      nickname: nickname || phone.slice(-4),
      mobile: phone,
      phone,
      status: 'active',
      register_source: 'mobile',
      mobile_verified_at: createdAt,
      last_login_at: createdAt,
      created_at: createdAt,
      updated_at: createdAt,
    };
    const session: Session = {
      token: randomId('utk'),
      user_id: user.id,
      expires_at: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
      session_type: 'normal',
    };
    this.users.set(user.id, user);
    this.phoneByNumber.set(phone, user.id);
    this.sessions.set(session.token, session);
    this.userProfiles.set(user.id, {
      user_id: user.id,
      gender: 'unknown',
      created_at: createdAt,
      updated_at: createdAt,
    });
    this.members.set(user.id, {
      user_id: user.id,
      level: 'normal',
      points: 1000,
      total_draws: 0,
      total_spent: 0,
      created_at: createdAt,
      updated_at: createdAt,
    });
    this.wallets.set(user.id, {
      user_id: user.id,
      cash_balance: 0,
      frozen_balance: 0,
      currency: 'CNY',
      created_at: createdAt,
      updated_at: createdAt,
    });
    this.logPoints(user.id, 1000, 1000, 'welcome', '新用户注册赠送');
    this.logLogin(user.id, 'phone_login', phone.replace(/^(\d{3})\d{4}(\d{4})$/, '$1****$2'), true);
    return { user, session };
  }

  public ensureAnonymousDrawToken(token?: string): string {
    return token?.trim() || randomId('anon');
  }

  public anonymousDrawActorId(token: string): string {
    return `anon:${token}`;
  }

  public userFromToken(token: string): User {
    const session = this.sessions.get(token);
    if (!session || session.revoked_at || new Date(session.expires_at).getTime() < Date.now()) {
      throw unauthorized;
    }
    const user = this.users.get(session.user_id);
    if (!user || user.status === 'disabled' || user.status === 'cancelled') {
      throw unauthorized;
    }
    return user;
  }

  public logout(token: string): void {
    const session = this.sessions.get(token);
    if (session) {
      this.sessions.set(token, { ...session, revoked_at: nowISO() });
    }
  }

  public assertAssetAllowed(userId: string, options?: { readonly allowPendingPhone?: boolean }): void {
    const user = this.users.get(userId);
    const allowPendingPhone = options?.allowPendingPhone ?? false;
    if (
      !user ||
      ((!allowPendingPhone && user.status === 'pending_phone') || user.status === 'frozen' || user.status === 'disabled' || user.status === 'cancelled')
    ) {
      throw userStatusForbidden;
    }
  }

  public updateUserProfile(userId: string, input: UserProfileMutation): UserAccount {
    const user = this.users.get(userId);
    if (!user) {
      throw unauthorized;
    }
    const now = nowISO();
    const updatedUser: User = {
      ...user,
      nickname: input.nickname?.trim() || user.nickname,
      avatar_url: input.avatar_url ?? user.avatar_url,
      updated_at: now,
    };
    const currentProfile = this.userProfiles.get(userId) ?? {
      user_id: userId,
      gender: 'unknown' as const,
      created_at: now,
      updated_at: now,
    };
    const updatedProfile: UserProfile = {
      ...currentProfile,
      gender: input.gender ?? currentProfile.gender,
      birthday: input.birthday ?? currentProfile.birthday,
      province: input.province ?? currentProfile.province,
      city: input.city ?? currentProfile.city,
      bio: input.bio ?? currentProfile.bio,
      updated_at: now,
    };
    this.users.set(userId, updatedUser);
    this.userProfiles.set(userId, updatedProfile);
    return this.userAccount(userId);
  }

  public userAccount(userId: string): UserAccount {
    const user = this.users.get(userId);
    if (!user) {
      throw unauthorized;
    }
    const wallet = this.wallets.get(userId) ?? {
      user_id: userId,
      cash_balance: 0,
      frozen_balance: 0,
      currency: 'CNY',
      created_at: user.created_at,
      updated_at: user.updated_at ?? user.created_at,
    };
    this.wallets.set(userId, wallet);
    return {
      user: { ...user },
      profile: this.userProfiles.get(userId) ? { ...this.userProfiles.get(userId)! } : undefined,
      member: this.getUserMember(userId),
      cash_balance: wallet.cash_balance,
      frozen_balance: wallet.frozen_balance,
      status: (user.status ?? 'active') as UserStatus,
    };
  }

  public sendPhoneCode(phone: string, scene = 'login'): { readonly sent: boolean; readonly provider: string; readonly expires_in: number; readonly dev_code?: string; readonly message: string } {
    const config = getAppConfig();
    const code = '123456';
    const createdAt = nowISO();
    this.phoneCodes.unshift({
      id: this.phoneCodes.length + 1,
      phone,
      scene,
      code,
      expires_at: new Date(Date.now() + PHONE_CODE_TTL_MS).toISOString(),
      created_at: createdAt,
      attempt_count: 0,
    });
    const isMock = !config.sms.accessKeyId || config.sms.provider === 'mock';
    return {
      sent: true,
      provider: config.sms.provider || 'mock',
      expires_in: PHONE_CODE_TTL_MS / 1000,
      dev_code: isMock ? code : undefined,
      message: isMock ? '短信服务未配置，已使用开发验证码。' : '验证码发送请求已记录，等待接入短信服务商。',
    };
  }

  public verifyPhoneCode(phone: string, code: string, scene = 'login'): { readonly user: User; readonly session: Session } {
    const matched = this.phoneCodes.find((item) => item.phone === phone && item.scene === scene && !item.verified_at);
    if (!matched || matched.code !== code || new Date(matched.expires_at).getTime() < Date.now()) {
      throw phoneCodeInvalid;
    }
    const index = this.phoneCodes.findIndex((item) => item.id === matched.id);
    this.phoneCodes[index] = { ...matched, verified_at: nowISO() };
    const existing = this.findUserByPhone(phone);
    if (existing) {
      const session = this.createSession(existing.id);
      this.logLogin(existing.id, 'mobile_code', phone.replace(/^(\d{3})\d{4}(\d{4})$/, '$1****$2'), true);
      return { user: existing, session };
    }
    return this.createPhoneUser(phone);
  }

  public campaigns(): readonly Campaign[] {
    return [...this.campaignsById.values()].map(cloneCampaign);
  }

  public getCampaign(campaignId: string): Campaign {
    const campaign = this.campaignsById.get(campaignId);
    if (!campaign) {
      throw campaignNotFound;
    }
    return cloneCampaign(campaign);
  }

  public prizeList(campaignId: string): readonly Prize[] {
    return [...(this.prizesByCampaign.get(campaignId) ?? [])].map((prize) => ({ ...prize }));
  }

  public exportAdminConfigState(): AdminConfigState {
    const prizesByCampaign: Record<string, readonly Prize[]> = {};
    for (const [campaignId, prizes] of this.prizesByCampaign.entries()) {
      prizesByCampaign[campaignId] = prizes.map((prize) => ({ ...prize }));
    }
    return {
      campaigns: this.campaigns(),
      prizesByCampaign,
      shopItems: this.shopItemList.map((item) => cloneShopItem(item)),
      firstRechargePacks: this.firstRechargePackList.map((pack) => cloneFirstRechargePack(pack)),
      cEndFeatureToggles: cloneCEndFeatureToggles(this.cEndFeatureToggleState),
    };
  }

  public hydrateAdminConfigState(state: Partial<AdminConfigState>): void {
    if (state.campaigns && state.campaigns.length > 0) {
      this.campaignsById.clear();
      for (const campaign of state.campaigns) {
        this.campaignsById.set(campaign.id, cloneCampaign(campaign));
      }
    }
    if (state.prizesByCampaign) {
      this.prizesByCampaign.clear();
      for (const [campaignId, prizes] of Object.entries(state.prizesByCampaign)) {
        this.prizesByCampaign.set(campaignId, prizes.map((prize) => ({ ...prize })));
      }
    }
    if (state.shopItems && state.shopItems.length > 0) {
      this.shopItemList = state.shopItems.map((item) => cloneShopItem(item));
      this.shopItemList.sort((left, right) => left.sort_order - right.sort_order);
    }
    if (state.firstRechargePacks && state.firstRechargePacks.length > 0) {
      this.firstRechargePackList = state.firstRechargePacks.map((pack) => cloneFirstRechargePack(pack));
      this.firstRechargePackList.sort((left, right) => left.sort_order - right.sort_order);
    }
    if (state.cEndFeatureToggles) {
      this.cEndFeatureToggleState = cloneCEndFeatureToggles(state.cEndFeatureToggles);
    }
  }

  public checkDrawQuota(userId: string, campaignId: string, dailyLimit: number): number {
    const used = this.quotaUsed.get(this.quotaKey(userId, campaignId)) ?? 0;
    return Math.max(0, dailyLimit - used);
  }

  public deductDrawQuota(userId: string, campaignId: string, count: number): number {
    const campaign = this.getCampaign(campaignId);
    const remaining = this.checkDrawQuota(userId, campaignId, campaign.daily_draw_limit);
    if (remaining < count) {
      throw noDrawChances;
    }
    const key = this.quotaKey(userId, campaignId);
    this.quotaUsed.set(key, (this.quotaUsed.get(key) ?? 0) + count);
    return this.checkDrawQuota(userId, campaignId, campaign.daily_draw_limit);
  }

  public createDrawRecord(userId: string, campaignId: string, prizeId: string): DrawRecord {
    const prizes = this.prizesByCampaign.get(campaignId) ?? [];
    const index = prizes.findIndex((prize) => prize.id === prizeId);
    const prize = prizes[index];
    if (!prize || prize.stock <= 0) {
      return this.createMissRecord(userId, campaignId);
    }

    const updatedPrize = { ...prize, stock: prize.stock - 1 };
    this.prizesByCampaign.set(campaignId, [
      ...prizes.slice(0, index),
      updatedPrize,
      ...prizes.slice(index + 1),
    ]);

    const record: DrawRecord = {
      id: randomId('draw'),
      campaign_id: campaignId,
      user_id: userId,
      prize_id: prizeId,
      prize_name: prize.name,
      result: 'win',
      drawn_at: nowISO(),
      chance_after: this.checkDrawQuota(userId, campaignId, this.getCampaign(campaignId).daily_draw_limit),
    };
    this.drawRecords.unshift(record);
    this.inventory.unshift({
      id: randomId('inv'),
      user_id: userId,
      prize_id: prize.id,
      prize_name: prize.name,
      prize_level: prize.level,
      campaign_id: campaignId,
      source: 'draw',
      created_at: record.drawn_at,
    });
    this.fulfillmentTaskItems.unshift({
      id: this.fulfillmentTaskItems.length + 1,
      draw_record_id: record.id,
      user_id: userId,
      prize_id: prize.id,
      status: 'pending',
      payload_json: JSON.stringify({ prize_name: prize.name }),
      operator_note: '',
      created_at: record.drawn_at,
      updated_at: record.drawn_at,
    });
    return record;
  }

  public createAnonymousPendingWin(anonymousToken: string, campaignId: string, prizeId: string): PendingAnonymousWin | null {
    const prizes = this.prizesByCampaign.get(campaignId) ?? [];
    const index = prizes.findIndex((prize) => prize.id === prizeId);
    const prize = prizes[index];
    if (!prize || prize.stock <= 0) {
      return null;
    }

    const updatedPrize = { ...prize, stock: prize.stock - 1 };
    this.prizesByCampaign.set(campaignId, [
      ...prizes.slice(0, index),
      updatedPrize,
      ...prizes.slice(index + 1),
    ]);

    const pending: PendingAnonymousWin = {
      id: randomId('draw'),
      campaign_id: campaignId,
      prize_id: prize.id,
      prize_name: prize.name,
      prize_level: prize.level,
      prize_image_url: prize.image_url,
      drawn_at: nowISO(),
    };
    this.pendingAnonymousWins.set(anonymousToken, [...(this.pendingAnonymousWins.get(anonymousToken) ?? []), pending]);
    return pending;
  }

  public anonymousPendingClaimCount(anonymousToken: string): number {
    return this.pendingAnonymousWins.get(anonymousToken)?.length ?? 0;
  }

  public pendingAnonymousWinsForToken(anonymousToken: string): readonly PendingAnonymousWin[] {
    return (this.pendingAnonymousWins.get(anonymousToken) ?? []).map((item) => ({ ...item }));
  }

  public hydratePendingAnonymousWins(entriesByToken: Readonly<Record<string, readonly PendingAnonymousWin[]>>): void {
    this.pendingAnonymousWins.clear();
    for (const [token, entries] of Object.entries(entriesByToken)) {
      this.pendingAnonymousWins.set(token, entries.map((item) => ({ ...item })));
    }
  }

  public claimAnonymousWins(anonymousToken: string, userId: string): number {
    const pendingWins = this.pendingAnonymousWins.get(anonymousToken) ?? [];
    if (pendingWins.length === 0) {
      return 0;
    }
    for (const pendingWin of pendingWins) {
      this.finalizeAnonymousWinClaim(userId, pendingWin);
    }
    this.pendingAnonymousWins.delete(anonymousToken);
    return pendingWins.length;
  }

  public createMissRecord(userId: string, campaignId: string): DrawRecord {
    const record: DrawRecord = {
      id: randomId('draw'),
      campaign_id: campaignId,
      user_id: userId,
      prize_name: '谢谢参与',
      result: 'miss',
      drawn_at: nowISO(),
      chance_after: this.checkDrawQuota(userId, campaignId, this.getCampaign(campaignId).daily_draw_limit),
    };
    this.drawRecords.unshift(record);
    return record;
  }

  public userDrawRecords(userId: string): readonly DrawRecord[] {
    return this.drawRecords
      .filter((record) => record.user_id === userId)
      .map((record) => ({ ...record }));
  }

  public getUserInventory(userId: string): readonly UserInventory[] {
    return this.inventory.filter((item) => item.user_id === userId).map((item) => ({ ...item }));
  }

  private finalizeAnonymousWinClaim(userId: string, pendingWin: PendingAnonymousWin): DrawRecord {
    const record: DrawRecord = {
      id: pendingWin.id,
      campaign_id: pendingWin.campaign_id,
      user_id: userId,
      prize_id: pendingWin.prize_id,
      prize_name: pendingWin.prize_name,
      result: 'win',
      drawn_at: pendingWin.drawn_at,
      chance_after: this.checkDrawQuota(userId, pendingWin.campaign_id, this.getCampaign(pendingWin.campaign_id).daily_draw_limit),
    };
    this.drawRecords.unshift(record);
    this.inventory.unshift({
      id: randomId('inv'),
      user_id: userId,
      prize_id: pendingWin.prize_id,
      prize_name: pendingWin.prize_name,
      prize_level: pendingWin.prize_level,
      campaign_id: pendingWin.campaign_id,
      source: 'draw',
      created_at: pendingWin.drawn_at,
    });
    this.fulfillmentTaskItems.unshift({
      id: this.fulfillmentTaskItems.length + 1,
      draw_record_id: record.id,
      user_id: userId,
      prize_id: pendingWin.prize_id,
      status: 'pending',
      payload_json: JSON.stringify({ prize_name: pendingWin.prize_name }),
      operator_note: 'anonymous draw claimed after login',
      created_at: pendingWin.drawn_at,
      updated_at: pendingWin.drawn_at,
    });
    return record;
  }

  public getSeriesProgress(userId: string, campaignId: string, campaignName: string): SeriesProgress {
    const prizes = this.prizeList(campaignId);
    const inventory = this.getUserInventory(userId).filter((item) => item.campaign_id === campaignId);
    const counts = new Map<string, number>();
    for (const item of inventory) {
      counts.set(item.prize_id, (counts.get(item.prize_id) ?? 0) + 1);
    }

    const collectedPrizes: CollectedPrize[] = [];
    const missingPrizes: PrizeSummary[] = [];

    for (const prize of prizes) {
      const count = counts.get(prize.id) ?? 0;
      if (count > 0) {
        collectedPrizes.push({ ...prize, count });
      } else {
        missingPrizes.push({
          prize_id: prize.id,
          prize_name: prize.name,
          prize_level: prize.level,
          stock: prize.stock,
        });
      }
    }

    const duplicates = [...counts.values()].reduce((sum, count) => sum + Math.max(0, count - 1), 0);
    return {
      campaign_id: campaignId,
      campaign_name: campaignName,
      total_items: prizes.length,
      collected_items: collectedPrizes.length,
      progress_percent: prizes.length > 0 ? (collectedPrizes.length / prizes.length) * 100 : 0,
      duplicates,
      collected_prizes: collectedPrizes,
      missing_prizes: missingPrizes,
    };
  }

  public exchangeOffersList(): readonly ExchangeOffer[] {
    return this.exchangeOffers.filter((offer) => offer.status === 'pending').map((offer) => ({ ...offer }));
  }

  public createExchangeOffer(userId: string, input: ExchangeOfferMutation): ExchangeOffer {
    const have = this.findPrize(input.have_prize_id);
    const want = this.findPrize(input.want_prize_id);
    const user = this.users.get(userId);
    const offer: ExchangeOffer = {
      id: randomId('ex'),
      user_id: userId,
      user_nickname: user?.nickname ?? '',
      have_prize_id: have.id,
      have_prize_name: have.name,
      want_prize_id: want.id,
      want_prize_name: want.name,
      status: 'pending',
      created_at: nowISO(),
    };
    this.exchangeOffers.unshift(offer);
    return offer;
  }

  public cancelExchangeOffer(userId: string, offerId: string): void {
    const index = this.exchangeOffers.findIndex((offer) => offer.id === offerId && offer.user_id === userId);
    if (index < 0) {
      throw notFound;
    }
    this.exchangeOffers[index] = { ...this.exchangeOffers[index], status: 'cancelled' };
  }

  public acceptExchangeOffer(userId: string, offerId: string): ExchangeOffer {
    const index = this.exchangeOffers.findIndex((offer) => offer.id === offerId);
    const offer = this.exchangeOffers[index];
    if (!offer || offer.status !== 'pending' || offer.user_id === userId) {
      throw notFound;
    }
    this.exchangeOffers[index] = { ...offer, status: 'completed' };
    return this.exchangeOffers[index];
  }

  public getUserMember(userId: string): UserMember {
    const member = this.members.get(userId);
    if (!member) {
      throw unauthorized;
    }
    return { ...member };
  }

  public updateUserMember(member: UserMember): void {
    this.members.set(member.user_id, { ...member, updated_at: nowISO() });
  }

  public getPointsLog(userId: string): readonly UserPointsLog[] {
    return this.pointsLogs.filter((log) => log.user_id === userId).map((log) => ({ ...log }));
  }

  public redeemPrize(userId: string, input: RedeemRequest): RedeemResult {
    const prize = this.findPrize(input.prize_id);
    const cost = this.pointsCostForPrize(prize.level);
    const member = this.getUserMember(userId);
    if (member.points < cost) {
      throw insufficientPoints;
    }

    const updated = {
      ...member,
      points: member.points - cost,
      updated_at: nowISO(),
    };
    this.updateUserMember(updated);
    this.logPoints(userId, -cost, updated.points, 'redeem', `兑换 ${prize.name}`);
    this.inventory.unshift({
      id: randomId('inv'),
      user_id: userId,
      prize_id: prize.id,
      prize_name: prize.name,
      prize_level: prize.level,
      campaign_id: prize.campaign_id,
      source: 'redeem',
      created_at: nowISO(),
    });

    return {
      record_id: randomId('redeem'),
      prize_id: prize.id,
      prize_name: prize.name,
      points_cost: cost,
      remaining: updated.points,
    };
  }

  public dailyCheckIn(userId: string): CheckInResult {
    const today = todayKey();
    if (this.checkInDates.get(userId) === today) {
      const member = this.getUserMember(userId);
      return {
        points_awarded: 0,
        streak_days: this.checkInStreaks.get(userId) ?? 1,
        is_bonus: false,
        new_balance: member.points,
      };
    }

    const yesterday = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString().slice(0, 10);
    const streak = this.checkInDates.get(userId) === yesterday ? (this.checkInStreaks.get(userId) ?? 0) + 1 : 1;
    const isBonus = streak % 7 === 0;
    const points = isBonus ? 70 : 20;
    const member = this.getUserMember(userId);
    const updated = { ...member, points: member.points + points, updated_at: nowISO() };

    this.checkInDates.set(userId, today);
    this.checkInStreaks.set(userId, streak);
    this.updateUserMember(updated);
    this.logPoints(userId, points, updated.points, 'daily', '每日签到');

    return {
      points_awarded: points,
      streak_days: streak,
      is_bonus: isBonus,
      new_balance: updated.points,
    };
  }

  public shareReward(userId: string): ShareRewardResult {
    const key = `${userId}:${todayKey()}`;
    const used = this.shareCounts.get(key) ?? 0;
    if (used >= 3) {
      return {
        points_awarded: 0,
        daily_left: 0,
        new_balance: this.getUserMember(userId).points,
      };
    }

    const points = 10;
    const member = this.getUserMember(userId);
    const updated = { ...member, points: member.points + points, updated_at: nowISO() };
    this.shareCounts.set(key, used + 1);
    this.updateUserMember(updated);
    this.logPoints(userId, points, updated.points, 'share', '分享奖励');

    return {
      points_awarded: points,
      daily_left: 3 - used - 1,
      new_balance: updated.points,
    };
  }

  public leaderboard(limit: number): readonly LeaderboardEntry[] {
    const entries = [...this.users.values()].map((user) => {
      const collectedIds = new Set(
        this.inventory.filter((item) => item.user_id === user.id).map((item) => item.prize_id),
      );
      const totalCount = [...this.prizesByCampaign.values()].reduce((sum, prizes) => sum + prizes.length, 0);
      return {
        rank: 0,
        user_id: user.id,
        nickname: user.nickname,
        collected_count: collectedIds.size,
        total_count: totalCount,
        progress_percent: totalCount > 0 ? (collectedIds.size / totalCount) * 100 : 0,
        series_completed: 0,
      };
    });

    return entries
      .sort((left, right) => right.collected_count - left.collected_count)
      .slice(0, limit)
      .map((entry, index) => ({ ...entry, rank: index + 1 }));
  }

  public adminLogin(username: string, password: string): string {
    if (!this.adminPassword || username !== this.adminUser || password !== this.adminPassword) {
      throw badAdminAuth;
    }
    const token = randomId('atk');
    this.adminSessions.set(token, new Date(Date.now() + 12 * 60 * 60 * 1000).toISOString());
    return token;
  }

  public adminOverview(token: string): AdminOverview {
    this.ensureAdmin(token);
    const prizeSummaries = [...this.prizesByCampaign.values()]
      .flat()
      .map((prize) => ({
        prize_id: prize.id,
        prize_name: prize.name,
        prize_level: prize.level,
        stock: prize.stock,
      }));

    return {
      total_users: this.users.size,
      total_draws: this.drawRecords.length,
      total_wins: this.drawRecords.filter((record) => record.result === 'win').length,
      campaigns: this.campaigns(),
      prize_summaries: prizeSummaries,
      recent_draws: this.drawRecords.slice(0, 10),
      user_draw_balance: {},
    };
  }

  public adminCampaigns(token: string): readonly Campaign[] {
    this.ensureAdmin(token);
    return this.campaigns();
  }

  public adminCEndFeatureToggles(token: string): CEndFeatureToggles {
    this.ensureAdmin(token);
    return cloneCEndFeatureToggles(this.cEndFeatureToggleState);
  }

  public updateCEndFeatureToggles(token: string, input: CEndFeatureToggles): CEndFeatureToggles {
    this.ensureAdmin(token);
    this.cEndFeatureToggleState = cloneCEndFeatureToggles(input);
    return cloneCEndFeatureToggles(this.cEndFeatureToggleState);
  }

  public createCampaign(token: string, input: CampaignMutation): Campaign {
    this.ensureAdmin(token);
    const campaign: Campaign = {
      id: randomId('camp'),
      name: input.name,
      slug: input.slug,
      status: input.status,
      starts_at: input.starts_at,
      ends_at: input.ends_at,
      daily_draw_limit: input.daily_draw_limit,
      requires_phone_login: input.requires_phone_login,
      miss_weight: input.miss_weight,
      banner_image_url: input.banner_image_url ?? '',
      campaign_summary: input.campaign_summary ?? '',
      pity_config: input.pity_config,
    };
    this.campaignsById.set(campaign.id, campaign);
    this.prizesByCampaign.set(campaign.id, []);
    return cloneCampaign(campaign);
  }

  public updateCampaign(token: string, campaignId: string, input: CampaignMutation): Campaign {
    this.ensureAdmin(token);
    this.getCampaign(campaignId);
    const campaign: Campaign = {
      id: campaignId,
      name: input.name,
      slug: input.slug,
      status: input.status,
      starts_at: input.starts_at,
      ends_at: input.ends_at,
      daily_draw_limit: input.daily_draw_limit,
      requires_phone_login: input.requires_phone_login,
      miss_weight: input.miss_weight,
      banner_image_url: input.banner_image_url ?? '',
      campaign_summary: input.campaign_summary ?? '',
      pity_config: input.pity_config,
    };
    this.campaignsById.set(campaignId, campaign);
    return cloneCampaign(campaign);
  }

  public updatePityConfig(token: string, campaignId: string, pityConfig: PityConfig): Campaign {
    this.ensureAdmin(token);
    const campaign = this.getCampaign(campaignId);
    this.validatePityConfig(campaignId, pityConfig);
    const updated = { ...campaign, pity_config: pityConfig };
    this.campaignsById.set(campaignId, updated);
    return cloneCampaign(updated);
  }

  public deleteCampaign(token: string, campaignId: string): void {
    this.ensureAdmin(token);
    this.campaignsById.delete(campaignId);
    this.prizesByCampaign.delete(campaignId);
  }

  public adminPrizes(token: string, campaignId: string): readonly Prize[] {
    this.ensureAdmin(token);
    this.getCampaign(campaignId);
    return this.prizeList(campaignId);
  }

  public createPrize(token: string, campaignId: string, input: PrizeMutation): Prize {
    this.ensureAdmin(token);
    this.getCampaign(campaignId);
    const prize: Prize = {
      id: randomId('prize'),
      campaign_id: campaignId,
      name: input.name,
      level: input.level,
      stock: input.stock,
      probability_weight: input.probability_weight,
      status: input.status,
    image_url: input.image_url?.trim() || undefined,
      sort_order: input.sort_order,
      display_prob: input.display_prob,
    };
    this.prizesByCampaign.set(campaignId, [...(this.prizesByCampaign.get(campaignId) ?? []), prize]);
    return { ...prize };
  }

  public updatePrize(token: string, prizeId: string, input: PrizeMutation): Prize {
    this.ensureAdmin(token);
    for (const [campaignId, prizes] of this.prizesByCampaign.entries()) {
      const index = prizes.findIndex((prize) => prize.id === prizeId);
      if (index >= 0) {
        const updated = {
          ...prizes[index],
          ...input,
          image_url: input.image_url === undefined ? prizes[index].image_url : input.image_url.trim() || undefined,
        };
        this.prizesByCampaign.set(campaignId, [
          ...prizes.slice(0, index),
          updated,
          ...prizes.slice(index + 1),
        ]);
        return updated;
      }
    }
    throw notFound;
  }

  public deletePrize(token: string, prizeId: string): void {
    this.ensureAdmin(token);
    for (const [campaignId, prizes] of this.prizesByCampaign.entries()) {
      const nextPrizes = prizes.filter((prize) => prize.id !== prizeId);
      if (nextPrizes.length !== prizes.length) {
        this.prizesByCampaign.set(campaignId, nextPrizes);
        return;
      }
    }
    throw notFound;
  }

  public validateCampaignPublish(token: string, campaignId: string): CampaignPublishValidation {
    this.ensureAdmin(token);
    const campaign = this.getCampaign(campaignId);
    const prizes = this.prizeList(campaignId);
    const activePrizes = prizes.filter((prize) => prize.status === 'active');
    const drawablePrizes = activePrizes.filter((prize) => prize.stock > 0 && prize.probability_weight > 0);
    const totalStock = activePrizes.reduce((sum, prize) => sum + prize.stock, 0);
    const totalWeight = activePrizes.reduce((sum, prize) => sum + prize.probability_weight, 0);
    const errors: string[] = [];
    const warnings: string[] = [];

    if (!campaign.name.trim()) {
      errors.push('盲盒名称不能为空');
    }
    if (!campaign.slug.trim()) {
      errors.push('盲盒标识不能为空');
    }
    if (new Date(campaign.ends_at).getTime() <= new Date(campaign.starts_at).getTime()) {
      errors.push('结束时间必须晚于开始时间');
    }
    if (prizes.length === 0) {
      errors.push('盲盒至少需要配置一个奖品');
    }
    if (activePrizes.length === 0) {
      errors.push('盲盒至少需要一个已上架奖品');
    }
    if (drawablePrizes.length === 0) {
      errors.push('盲盒至少需要一个有库存且权重大于 0 的可抽奖品');
    }
    if (totalWeight <= 0) {
      errors.push('可抽奖品权重合计必须大于 0');
    }
    if (!campaign.banner_image_url) {
      warnings.push('建议补充盲盒 Banner 图，提升用户端展示效果');
    }
    if (!campaign.campaign_summary) {
      warnings.push('建议补充盲盒简介，便于用户理解活动规则');
    }
    if (activePrizes.some((prize) => !prize.image_url)) {
      warnings.push('存在未配置图片的上架奖品');
    }

    if (campaign.pity_config) {
      this.collectPityConfigIssues(campaignId, campaign.pity_config, errors);
    }

    return {
      campaign_id: campaign.id,
      campaign_name: campaign.name,
      prize_count: prizes.length,
      active_prize_count: activePrizes.length,
      total_stock: totalStock,
      total_weight: totalWeight,
      can_publish: errors.length === 0,
      errors,
      warnings,
    };
  }

  public adminShopItems(token: string): readonly ShopItem[] {
    this.ensureAdmin(token);
    return this.shopItemList
      .slice()
      .sort((left, right) => left.sort_order - right.sort_order)
      .map((item) => cloneShopItem(item));
  }

  public createShopItem(token: string, input: ShopItemMutation): ShopItem {
    this.ensureAdmin(token);
    const item: ShopItem = {
      id: randomId('shop'),
      ...input,
      image_url: input.image_url?.trim() || undefined,
    };
    this.shopItemList.push(item);
    this.shopItemList.sort((left, right) => left.sort_order - right.sort_order);
    return cloneShopItem(item);
  }

  public updateShopItem(token: string, itemId: string, input: ShopItemMutation): ShopItem {
    this.ensureAdmin(token);
    const index = this.shopItemList.findIndex((item) => item.id === itemId);
    if (index < 0) {
      throw notFound;
    }
    const updated: ShopItem = {
      id: itemId,
      ...input,
      image_url: input.image_url?.trim() || undefined,
    };
    this.shopItemList.splice(index, 1, updated);
    this.shopItemList.sort((left, right) => left.sort_order - right.sort_order);
    return cloneShopItem(updated);
  }

  public deleteShopItem(token: string, itemId: string): void {
    this.ensureAdmin(token);
    const next = this.shopItemList.filter((item) => item.id !== itemId);
    if (next.length === this.shopItemList.length) {
      throw notFound;
    }
    this.shopItemList = next;
  }

  public adminFirstRechargePacks(token: string): readonly FirstRechargePack[] {
    this.ensureAdmin(token);
    return this.firstRechargePackList
      .slice()
      .sort((left, right) => left.sort_order - right.sort_order)
      .map((pack) => cloneFirstRechargePack(pack));
  }

  public createFirstRechargePack(token: string, input: FirstRechargePackMutation): FirstRechargePack {
    this.ensureAdmin(token);
    const pack: FirstRechargePack = {
      id: randomId('pack'),
      ...input,
      image_url: input.image_url?.trim() || undefined,
      items: input.items.map((item) => clonePackItem(item)),
    };
    this.firstRechargePackList.push(pack);
    this.firstRechargePackList.sort((left, right) => left.sort_order - right.sort_order);
    return cloneFirstRechargePack(pack);
  }

  public updateFirstRechargePack(token: string, packId: string, input: FirstRechargePackMutation): FirstRechargePack {
    this.ensureAdmin(token);
    const index = this.firstRechargePackList.findIndex((pack) => pack.id === packId);
    if (index < 0) {
      throw notFound;
    }
    const updated: FirstRechargePack = {
      id: packId,
      ...input,
      image_url: input.image_url?.trim() || undefined,
      items: input.items.map((item) => clonePackItem(item)),
    };
    this.firstRechargePackList.splice(index, 1, updated);
    this.firstRechargePackList.sort((left, right) => left.sort_order - right.sort_order);
    return cloneFirstRechargePack(updated);
  }

  public deleteFirstRechargePack(token: string, packId: string): void {
    this.ensureAdmin(token);
    const next = this.firstRechargePackList.filter((pack) => pack.id !== packId);
    if (next.length === this.firstRechargePackList.length) {
      throw notFound;
    }
    this.firstRechargePackList = next;
  }

  public adminDrawRecords(token: string): readonly DrawRecord[] {
    this.ensureAdmin(token);
    return this.drawRecords.map((record) => ({ ...record }));
  }

  public adminUsers(token: string, query: { readonly page?: number; readonly page_size?: number; readonly keyword?: string; readonly status?: string; readonly register_source?: string }): AdminUserListResult {
    this.ensureAdmin(token);
    const page = Math.max(1, query.page ?? 1);
    const pageSize = Math.min(100, Math.max(1, query.page_size ?? 20));
    const keyword = (query.keyword ?? '').trim().toLowerCase();
    const filtered = [...this.users.values()].filter((user) => {
      const member = this.members.get(user.id);
      const matchesKeyword =
        !keyword ||
        user.id.toLowerCase().includes(keyword) ||
        user.nickname.toLowerCase().includes(keyword) ||
        (user.mobile ?? user.phone ?? '').includes(keyword);
      const matchesStatus = !query.status || (user.status ?? 'active') === query.status;
      const matchesSource = !query.register_source || (user.register_source ?? 'guest') === query.register_source;
      return Boolean(member) && matchesKeyword && matchesStatus && matchesSource;
    });
    const items = filtered.slice((page - 1) * pageSize, page * pageSize).map((user) => {
      const member = this.getUserMember(user.id);
      const wallet = this.wallets.get(user.id);
      return {
        id: user.id,
        nickname: user.nickname,
        mobile: user.mobile ?? user.phone,
        avatar_url: user.avatar_url,
        status: (user.status ?? 'active') as UserStatus,
        register_source: user.register_source ?? 'guest',
        member_level: member.level,
        points_balance: member.points,
        cash_balance: wallet?.cash_balance ?? 0,
        total_draws: member.total_draws,
        total_spent: member.total_spent,
        last_login_at: user.last_login_at,
        created_at: user.created_at,
      };
    });
    return { items, page, page_size: pageSize, total: filtered.length };
  }

  public adminUserDetail(token: string, userId: string): AdminUserDetail {
    this.ensureAdmin(token);
    const user = this.users.get(userId);
    if (!user) {
      throw notFound;
    }
    const account = this.userAccount(userId);
    return {
      user: account.user,
      profile: account.profile,
      member: account.member,
      cash_balance: account.cash_balance,
      frozen_balance: account.frozen_balance,
      identities: this.wechatUsers.get(userId) ? [{ ...this.wechatUsers.get(userId)! }] : [],
      recent_draws: this.drawRecords.filter((record) => record.user_id === userId).slice(0, 20).map((record) => ({ ...record })),
      points_logs: this.getPointsLog(userId).slice(0, 20),
      login_logs: this.loginLogs.filter((log) => log.user_id === userId).slice(0, 20).map((log) => ({ ...log })),
      status_logs: this.statusLogs.filter((log) => log.user_id === userId).slice(0, 20).map((log) => ({ ...log })),
    };
  }

  public updateUserStatus(token: string, userId: string, status: UserStatus, reason: string): User {
    this.ensureAdmin(token);
    const user = this.users.get(userId);
    if (!user) {
      throw notFound;
    }
    const fromStatus = (user.status ?? 'active') as UserStatus;
    const updated: User = {
      ...user,
      status,
      updated_at: nowISO(),
      ...(status === 'cancelled'
        ? {
            mobile: undefined,
            phone: undefined,
            nickname: `已注销用户${user.id.slice(-4)}`,
            avatar_url: undefined,
          }
        : {}),
    };
    this.users.set(userId, updated);
    if (status === 'disabled' || status === 'cancelled') {
      this.revokeUserSessions(token, userId);
    }
    this.logStatus(userId, fromStatus, status, reason, this.adminUser);
    return { ...updated };
  }

  public adjustUserPoints(token: string, userId: string, points: number, reason: string, remark = ''): UserMember {
    this.ensureAdmin(token);
    const member = this.getUserMember(userId);
    const nextPoints = member.points + points;
    if (nextPoints < 0) {
      throw insufficientPoints;
    }
    const updated: UserMember = {
      ...member,
      points: nextPoints,
      updated_at: nowISO(),
    };
    this.updateUserMember(updated);
    this.logPoints(userId, points, updated.points, 'admin_adjust', `${reason}${remark ? `：${remark}` : ''}`);
    return updated;
  }

  public adminUserPointsLog(token: string, userId: string): readonly UserPointsLog[] {
    this.ensureAdmin(token);
    return this.getPointsLog(userId);
  }

  public adminUserLoginLogs(token: string, userId: string): readonly UserLoginLog[] {
    this.ensureAdmin(token);
    return this.loginLogs.filter((log) => log.user_id === userId).map((log) => ({ ...log }));
  }

  public adminUserStatusLogs(token: string, userId: string): readonly UserStatusLog[] {
    this.ensureAdmin(token);
    return this.statusLogs.filter((log) => log.user_id === userId).map((log) => ({ ...log }));
  }

  public revokeUserSessions(token: string, userId: string): number {
    this.ensureAdmin(token);
    let count = 0;
    for (const [sessionToken, session] of this.sessions.entries()) {
      if (session.user_id === userId && !session.revoked_at) {
        this.sessions.set(sessionToken, { ...session, revoked_at: nowISO() });
        count += 1;
      }
    }
    return count;
  }

  public fulfillmentTasks(token: string): readonly FulfillmentTask[] {
    this.ensureAdmin(token);
    return this.fulfillmentTaskItems.map((task) => ({ ...task }));
  }

  public updateFulfillmentTask(
    token: string,
    taskId: number,
    input: FulfillmentTaskMutation,
  ): FulfillmentTask {
    this.ensureAdmin(token);
    const index = this.fulfillmentTaskItems.findIndex((task) => task.id === taskId);
    const task = this.fulfillmentTaskItems[index];
    if (!task) {
      throw notFound;
    }
    const updated: FulfillmentTask = {
      ...task,
      status: input.status,
      operator_note: input.operator_note,
      updated_at: nowISO(),
      fulfilled_at: input.status === 'fulfilled' ? nowISO() : task.fulfilled_at,
    };
    this.fulfillmentTaskItems[index] = updated;
    return updated;
  }

  public drawStatistics(token: string, campaignId: string): DrawStatistics {
    this.ensureAdmin(token);
    const records = this.drawRecords.filter((record) => !campaignId || record.campaign_id === campaignId);
    const wins = records.filter((record) => record.result === 'win');
    const counts = new Map<string, { readonly name: string; readonly level: string; count: number }>();
    for (const record of wins) {
      if (!record.prize_id) {
        continue;
      }
      const prize = this.findPrize(record.prize_id);
      const current = counts.get(record.prize_id) ?? { name: prize.name, level: prize.level, count: 0 };
      current.count += 1;
      counts.set(record.prize_id, current);
    }

    return {
      total_draws: records.length,
      total_users: new Set(records.map((record) => record.user_id)).size,
      total_wins: wins.length,
      win_rate: records.length > 0 ? (wins.length / records.length) * 100 : 0,
      prize_breakdown: [...counts.entries()].map(([prizeId, item]) => ({
        prize_id: prizeId,
        prize_name: item.name,
        level: item.level,
        count: item.count,
        percent: wins.length > 0 ? (item.count / wins.length) * 100 : 0,
      })),
    };
  }

  public buyCard(userId: string, input: BuyCardRequest): BuyCardResult {
    const config = cardConfigs[input.card_type];
    const member = this.getUserMember(userId);
    if (member.points < config.price) {
      throw insufficientPoints;
    }
    const updated = { ...member, points: member.points - config.price, total_spent: member.total_spent + config.price };
    this.updateUserMember(updated);
    this.logPoints(userId, -config.price, updated.points, 'card', `购买${input.card_type}`);
    const card: UserCard = {
      id: randomId('card'),
      user_id: userId,
      card_type: input.card_type,
      price: config.price,
      started_at: nowISO(),
      expires_at: addDays(config.durationDays),
      daily_free_used: 0,
      free_date: todayKey(),
      created_at: nowISO(),
    };
    this.userCards.set(userId, card);
    return { card_type: input.card_type, expires_at: card.expires_at, price: config.price, points: updated.points };
  }

  public getUserCard(userId: string): UserCard | null {
    const card = this.userCards.get(userId);
    if (!card || new Date(card.expires_at).getTime() < Date.now()) {
      return null;
    }
    return { ...card };
  }

  public monthCardStatus(userId: string): MonthCardStatus {
    const card = this.getUserCard(userId);
    if (!card) {
      return { has_card: false, free_draws: 0, draw_discount: 1, days_left: 0, today_free_used: 0 };
    }
    const config = cardConfigs[card.card_type];
    return {
      has_card: true,
      card_type: card.card_type,
      free_draws: config.freeDraws,
      draw_discount: config.discount,
      expires_at: card.expires_at,
      days_left: Math.ceil((new Date(card.expires_at).getTime() - Date.now()) / (24 * 60 * 60 * 1000)),
      today_free_used: card.free_date === todayKey() ? card.daily_free_used : 0,
    };
  }

  public buyMonthCard(userId: string, cardType: CardType): MonthCardPurchaseResult {
    const result = this.buyCard(userId, { card_type: cardType });
    const card = this.getUserCard(userId);
    if (!card) {
      throw notFound;
    }
    return { card, new_points: result.points };
  }

  public battlePassInfo(userId: string): BattlePassInfo {
    if (!this.battlePassSeason) {
      return { season: null, tasks: [], rewards: [], level_progress: 0 };
    }
    const userPass = this.getOrCreateBattlePass(userId);
    const progress = this.battlePassTasks.map((task) => this.taskProgress.get(`${userId}:${task.id}`)).filter(Boolean) as BattlePassTaskProgress[];
    return {
      season: this.battlePassSeason,
      user_pass: userPass,
      tasks: this.battlePassTasks,
      task_progress: progress,
      rewards: this.battlePassRewards,
      level_progress: userPass.xp,
    };
  }

  public buyBattlePass(userId: string): BattlePass {
    const member = this.getUserMember(userId);
    const cost = 680;
    if (member.points < cost) {
      throw insufficientPoints;
    }
    const updated = { ...member, points: member.points - cost };
    this.updateUserMember(updated);
    this.logPoints(userId, -cost, updated.points, 'battle_pass', '购买付费战令');
    const current = this.getOrCreateBattlePass(userId);
    const next = { ...current, pass_type: 'paid' as const, bought_at: nowISO(), updated_at: nowISO() };
    this.battlePasses.set(userId, next);
    return next;
  }

  public claimBattlePassReward(userId: string, level: number): boolean {
    const pass = this.getOrCreateBattlePass(userId);
    if (pass.level < level || pass.claimed_levels.includes(level)) {
      return false;
    }
    const next = { ...pass, claimed_levels: [...pass.claimed_levels, level], updated_at: nowISO() };
    this.battlePasses.set(userId, next);
    const reward = this.battlePassRewards.find((item) => item.level === level && item.pass_type === pass.pass_type);
    if (reward?.reward_type === 'points') {
      const member = this.getUserMember(userId);
      const updated = { ...member, points: member.points + reward.reward_qty };
      this.updateUserMember(updated);
      this.logPoints(userId, reward.reward_qty, updated.points, 'battle_pass', `领取${reward.reward_name}`);
    }
    return true;
  }

  public shopItems(): readonly ShopItem[] {
    return this.shopItemList.filter((item) => item.is_active).map((item) => cloneShopItem(item));
  }

  public cEndFeatureToggles(): CEndFeatureToggles {
    return cloneCEndFeatureToggles(this.cEndFeatureToggleState);
  }

  public buyShopItem(userId: string, input: BuyShopItemRequest): BuyShopItemResult {
    const item = this.shopItemList.find((candidate) => candidate.id === input.shop_item_id && candidate.is_active);
    if (!item) {
      throw notFound;
    }
    const quantity = Math.max(1, input.quantity ?? 1);
    const pointsCost = item.price_points * quantity;
    const member = this.getUserMember(userId);
    if (member.points < pointsCost) {
      throw insufficientPoints;
    }
    const updated = { ...member, points: member.points - pointsCost };
    this.updateUserMember(updated);
    this.logPoints(userId, -pointsCost, updated.points, 'shop', `购买${item.name}x${quantity}`);
    const newQty = this.addUserItem(userId, item.item_type, item.item_qty * quantity);
    return {
      item_type: item.item_type,
      item_name: item.name,
      quantity,
      points_cost: pointsCost,
      new_points: updated.points,
      new_qty: newQty,
    };
  }

  public userItemsList(userId: string): readonly UserItem[] {
    return [...this.userItems.values()].filter((item) => item.user_id === userId).map((item) => ({ ...item }));
  }

  public useItem(userId: string, input: UseItemRequest): { readonly item_type: ItemType; readonly remaining: number; readonly message: string } {
    const remaining = this.addUserItem(userId, input.item_type, -1);
    return { item_type: input.item_type, remaining, message: '道具已使用' };
  }

  public firstRechargePacks(): readonly FirstRechargePack[] {
    return this.firstRechargePackList.map((pack) => cloneFirstRechargePack(pack));
  }

  public firstRechargeStatus(userId: string): UserFirstRecharge {
    return { user_id: userId, claimed: [...(this.firstRechargeClaims.get(userId)?.claimed ?? [])] };
  }

  public claimFirstRecharge(userId: string, input: ClaimFirstRechargeRequest): ClaimFirstRechargeResult {
    const pack = this.firstRechargePackList.find((item) => item.id === input.pack_id);
    if (!pack) {
      throw notFound;
    }
    const current = this.firstRechargeStatus(userId);
    if (current.claimed.includes(pack.id)) {
      throw notFound;
    }
    let member = this.getUserMember(userId);
    for (const item of pack.items) {
      if (item.type === 'points') {
        member = { ...member, points: member.points + item.qty };
        this.updateUserMember(member);
        this.logPoints(userId, item.qty, member.points, 'first_recharge', pack.name);
      } else if (this.isItemType(item.type)) {
        this.addUserItem(userId, item.type, item.qty);
      }
    }
    this.firstRechargeClaims.set(userId, { user_id: userId, claimed: [...current.claimed, pack.id] });
    return { pack_id: pack.id, pack_name: pack.name, items: pack.items, new_points: this.getUserMember(userId).points };
  }

  public blendPrizes(userId: string, input: BlendRequest): BlendResult {
    const source = this.findPrize(input.source_prize_id);
    const recipe = ({ common: 3, rare: 5, secret: 3 } as Record<string, number>)[source.level] ?? 0;
    const nextLevel = ({ common: 'rare', rare: 'secret', secret: 'limited' } as Record<string, string>)[source.level];
    if (!recipe || !nextLevel) {
      throw notFound;
    }
    const ownedIndexes = this.inventory
      .map((item, index) => ({ item, index }))
      .filter(({ item }) => item.user_id === userId && item.prize_id === input.source_prize_id);
    if (ownedIndexes.length < recipe) {
      throw notFound;
    }
    for (const { index } of ownedIndexes.slice(0, recipe).sort((left, right) => right.index - left.index)) {
      this.inventory.splice(index, 1);
    }
    const result = this.prizeList(input.campaign_id).find((candidate) => candidate.level === nextLevel) ?? source;
    this.inventory.unshift({
      id: randomId('inv'),
      user_id: userId,
      prize_id: result.id,
      prize_name: result.name,
      prize_level: result.level,
      campaign_id: result.campaign_id,
      source: 'collection_reward',
      created_at: nowISO(),
    });
    return {
      source_prize_id: source.id,
      source_prize_name: source.name,
      source_level: source.level,
      result_prize_id: result.id,
      result_prize_name: result.name,
      result_level: result.level,
      remaining_src: ownedIndexes.length - recipe,
    };
  }

  public campaignHint(campaignId: string): HintMessage {
    const campaign = this.getCampaign(campaignId);
    return { type: 'luck', content: `${campaign.name} 今日手感不错，隐藏款概率公示透明，记得关注保底进度。` };
  }

  public createShareCard(userId: string, cardType: string, prizeName = '', prizeLevel = ''): ShareCard {
    const card: ShareCard = {
      id: randomId('share'),
      user_id: userId,
      card_type: cardType,
      title: cardType === 'invite' ? '邀请好友一起开盒' : `我抽到了${prizeName || '惊喜盲盒'}`,
      description: '来 BOX·MAGIC 一起收集限定盲盒。',
      prize_name: prizeName || undefined,
      prize_level: prizeLevel || undefined,
      invite_link: `https://boxmagic.example/invite?from=${userId}`,
      created_at: nowISO(),
    };
    this.shareCards.unshift(card);
    return { ...card };
  }

  public getShareCards(userId: string): readonly ShareCard[] {
    return this.shareCards.filter((card) => card.user_id === userId).map((card) => ({ ...card }));
  }

  public generateInviteLink(userId: string): ShareCard {
    return this.createShareCard(userId, 'invite');
  }

  public inviteRecordsFor(userId: string): readonly InviteRecord[] {
    return this.inviteRecords.filter((record) => record.inviter_id === userId).map((record) => ({ ...record }));
  }

  public inviteStats(userId: string): InviteStats {
    const assists = [...this.assistProgress.values()].filter((progress) => progress.inviter_id === userId);
    return {
      total_invites: this.inviteRecordsFor(userId).length,
      total_assists: assists.reduce((sum, item) => sum + item.current, 0),
      completed_assists: assists.filter((item) => item.claimed).length,
      free_draws_earned: assists.filter((item) => item.assist_type === 'free_draw' && item.claimed).length,
    };
  }

  public allAssistProgress(userId: string): Record<AssistType, AssistProgress> {
    return {
      free_draw: this.getAssistProgress(userId, 'free_draw'),
      pity_reduce: this.getAssistProgress(userId, 'pity_reduce'),
      craft_boost: this.getAssistProgress(userId, 'craft_boost'),
    };
  }

  public recordAssist(userId: string, assistType: AssistType, helperId: string): AssistProgress {
    const progress = this.getAssistProgress(userId, assistType);
    const next = { ...progress, current: Math.min(progress.target_count, progress.current + (helperId ? 1 : 0)) };
    this.assistProgress.set(`${userId}:${assistType}`, next);
    if (helperId) {
      this.inviteRecords.unshift({ id: randomId('invte'), inviter_id: userId, invitee_id: helperId, created_at: nowISO() });
    }
    return next;
  }

  public claimAssist(userId: string, assistType: AssistType): AssistClaimResult {
    const progress = this.getAssistProgress(userId, assistType);
    if (progress.current < progress.target_count || progress.claimed) {
      return { assist_type: assistType, reward_type: 'none', description: '助力未达成或已领取' };
    }
    this.assistProgress.set(`${userId}:${assistType}`, { ...progress, claimed: true });
    if (assistType === 'free_draw') {
      this.addUserItem(userId, 'free_draw', 1);
    }
    return { assist_type: assistType, reward_type: assistType, description: '助力奖励已领取' };
  }

  public createTeam(userId: string, input: CreateTeamRequest): TeamInfo {
    const user = this.users.get(userId);
    const team: Team = {
      id: randomId('team'),
      captain_id: userId,
      name: input.name || '我的队伍',
      max_members: Math.min(5, Math.max(2, input.max_members)),
      goal_draws: Math.max(1, input.goal_draws),
      current_draws: 0,
      starts_at: nowISO(),
      expires_at: addDays(2),
      status: 'recruiting',
      created_at: nowISO(),
    };
    this.teams.set(team.id, team);
    this.teamMembers.push({ team_id: team.id, user_id: userId, nickname: user?.nickname, draws: 0, joined_at: nowISO() });
    return this.teamInfo(team.id);
  }

  public joinTeam(userId: string, teamId: string): TeamInfo {
    const team = this.teams.get(teamId);
    if (!team) {
      throw notFound;
    }
    if (!this.teamMembers.some((member) => member.team_id === teamId && member.user_id === userId)) {
      const user = this.users.get(userId);
      this.teamMembers.push({ team_id: teamId, user_id: userId, nickname: user?.nickname, draws: 0, joined_at: nowISO() });
    }
    return this.teamInfo(teamId);
  }

  public myTeam(userId: string): TeamInfo {
    const membership = this.teamMembers.find((member) => member.user_id === userId);
    return membership ? this.teamInfo(membership.team_id) : { team: null, members: [], captain_name: '', remaining_hours: 0 };
  }

  public leaveTeam(userId: string): void {
    const index = this.teamMembers.findIndex((member) => member.user_id === userId);
    if (index >= 0) {
      this.teamMembers.splice(index, 1);
    }
  }

  public sendGift(userId: string, input: SendGiftRequest): GiftRecord {
    const prize = this.findPrize(input.prize_id);
    const fee = prize.level === 'common' ? 0 : 20;
    const member = this.getUserMember(userId);
    if (member.points < fee) {
      throw insufficientPoints;
    }
    if (fee > 0) {
      const updated = { ...member, points: member.points - fee };
      this.updateUserMember(updated);
      this.logPoints(userId, -fee, updated.points, 'gift', `赠送${prize.name}`);
    }
    const gift: GiftRecord = {
      id: randomId('gift'),
      giver_id: userId,
      receiver_id: input.receiver_id,
      prize_id: prize.id,
      prize_name: prize.name,
      prize_level: prize.level,
      fee_points: fee,
      status: 'sent',
      created_at: nowISO(),
      expires_at: addDays(1),
    };
    this.gifts.unshift(gift);
    return { ...gift };
  }

  public receiveGift(userId: string, giftId: string): ReceiveGiftResult {
    const index = this.gifts.findIndex((gift) => gift.id === giftId && gift.receiver_id === userId && gift.status === 'sent');
    const gift = this.gifts[index];
    if (!gift) {
      throw notFound;
    }
    const newItemId = randomId('inv');
    this.gifts[index] = { ...gift, status: 'received', received_at: nowISO() };
    this.inventory.unshift({
      id: newItemId,
      user_id: userId,
      prize_id: gift.prize_id,
      prize_name: gift.prize_name,
      prize_level: gift.prize_level,
      campaign_id: this.findPrize(gift.prize_id).campaign_id,
      source: 'exchange',
      created_at: nowISO(),
    });
    return { gift_id: giftId, prize_name: gift.prize_name, prize_level: gift.prize_level, new_item_id: newItemId };
  }

  public incomingGifts(userId: string): readonly GiftRecord[] {
    return this.gifts.filter((gift) => gift.receiver_id === userId).map((gift) => ({ ...gift }));
  }

  public sentGifts(userId: string): readonly GiftRecord[] {
    return this.gifts.filter((gift) => gift.giver_id === userId).map((gift) => ({ ...gift }));
  }

  public puzzleTemplateList(): readonly PuzzleTemplate[] {
    return this.puzzleTemplates.filter((template) => template.is_active).map((template) => ({ ...template }));
  }

  public puzzleInfo(userId: string, templateId: string): PuzzleInfo {
    const template = this.puzzleTemplates.find((item) => item.id === templateId);
    if (!template) {
      throw notFound;
    }
    const progress = this.getOrCreatePuzzleProgress(userId, template);
    const collectedNames = progress.collected.map((index) => template.piece_names[index]).filter(Boolean);
    return {
      template,
      progress,
      collected_names: collectedNames,
      missing_names: template.piece_names.filter((_, index) => !progress.collected.includes(index)),
      progress_percent: template.total_pieces > 0 ? (progress.collected.length / template.total_pieces) * 100 : 0,
    };
  }

  public allPuzzleInfo(userId: string): readonly PuzzleInfo[] {
    return this.puzzleTemplates.map((template) => this.puzzleInfo(userId, template.id));
  }

  public composePuzzle(userId: string, templateId: string): ComposePuzzleResult {
    const info = this.puzzleInfo(userId, templateId);
    if (info.progress.collected.length < info.template.total_pieces) {
      throw notFound;
    }
    this.puzzleProgress.set(`${userId}:${templateId}`, { ...info.progress, is_completed: true, completed_at: nowISO() });
    if (info.template.reward_type === 'points') {
      const member = this.getUserMember(userId);
      const updated = { ...member, points: member.points + info.template.reward_qty };
      this.updateUserMember(updated);
      this.logPoints(userId, info.template.reward_qty, updated.points, 'puzzle', info.template.reward_name);
    }
    return {
      template_id: info.template.id,
      template_name: info.template.name,
      reward_type: info.template.reward_type,
      reward_name: info.template.reward_name,
      reward_qty: info.template.reward_qty,
    };
  }

  public createPuzzleTeam(userId: string, templateId: string): PuzzleTeam {
    const template = this.puzzleTemplates.find((item) => item.id === templateId);
    if (!template) {
      throw notFound;
    }
    const team: PuzzleTeam = {
      id: randomId('pteam'),
      template_id: template.id,
      captain_id: userId,
      members: [userId],
      shared: [],
      total_pieces: template.total_pieces,
      is_completed: false,
      created_at: nowISO(),
    };
    this.puzzleTeams.set(team.id, team);
    return team;
  }

  public joinPuzzleTeam(userId: string, teamId: string): PuzzleTeam {
    const team = this.puzzleTeams.get(teamId);
    if (!team) {
      throw notFound;
    }
    const next = { ...team, members: [...new Set([...team.members, userId])] };
    this.puzzleTeams.set(teamId, next);
    return next;
  }

  public myPuzzleTeams(userId: string): readonly PuzzleTeam[] {
    return [...this.puzzleTeams.values()].filter((team) => team.members.includes(userId)).map((team) => ({ ...team }));
  }

  public flashList(userId: string): readonly FlashListInfo[] {
    const member = this.getUserMember(userId);
    return this.flashSales.map((flash) => ({
      flash,
      subscribed: this.flashSubscriptions.some((item) => item.user_id === userId && item.flash_id === flash.id),
      purchasable: member.points >= flash.price_points && flash.remaining_stock > 0,
    }));
  }

  public subscribeFlash(userId: string, flashId: string): void {
    if (!this.flashSales.some((flash) => flash.id === flashId)) {
      throw notFound;
    }
    if (!this.flashSubscriptions.some((item) => item.user_id === userId && item.flash_id === flashId)) {
      this.flashSubscriptions.push({ user_id: userId, flash_id: flashId, created_at: nowISO() });
    }
  }

  public unsubscribeFlash(userId: string, flashId: string): void {
    const index = this.flashSubscriptions.findIndex((item) => item.user_id === userId && item.flash_id === flashId);
    if (index >= 0) {
      this.flashSubscriptions.splice(index, 1);
    }
  }

  public purchaseFlash(userId: string, flashId: string): FlashPurchaseResult {
    const index = this.flashSales.findIndex((flash) => flash.id === flashId);
    const flash = this.flashSales[index];
    if (!flash || flash.remaining_stock <= 0) {
      throw notFound;
    }
    const member = this.getUserMember(userId);
    if (member.points < flash.price_points) {
      throw insufficientPoints;
    }
    const updated = { ...member, points: member.points - flash.price_points };
    this.updateUserMember(updated);
    this.logPoints(userId, -flash.price_points, updated.points, 'flash', flash.name);
    this.flashSales[index] = { ...flash, remaining_stock: flash.remaining_stock - 1 };
    return { flash_id: flash.id, flash_name: flash.name, success: true, message: '抢购成功', item_name: flash.name };
  }

  public myFlashSubscriptions(userId: string): readonly FlashSubscription[] {
    return this.flashSubscriptions.filter((item) => item.user_id === userId).map((item) => ({ ...item }));
  }

  public activityList(userId: string): readonly ActivityListInfo[] {
    return this.activities
      .filter((activity) => activity.status === 'active')
      .sort((left, right) => left.sort_order - right.sort_order)
      .map((activity) => this.activityInfo(userId, activity.id));
  }

  public activityInfo(userId: string, activityId: string): ActivityListInfo {
    const activity = this.activities.find((item) => item.id === activityId);
    if (!activity) {
      throw notFound;
    }
    const participation = this.activityParticipations.find((item) => item.user_id === userId && item.activity_id === activityId);
    return {
      activity,
      joined: Boolean(participation),
      can_claim: Boolean(participation && !participation.reward_claimed),
      rewards: this.activityRewards.filter((reward) => reward.activity_id === activityId),
    };
  }

  public joinActivity(userId: string, activityId: string): ActivityParticipation {
    const existing = this.activityParticipations.find((item) => item.user_id === userId && item.activity_id === activityId);
    if (existing) {
      return { ...existing };
    }
    const participation: ActivityParticipation = {
      id: randomId('actp'),
      user_id: userId,
      activity_id: activityId,
      reward_claimed: false,
      joined_at: nowISO(),
    };
    this.activityParticipations.push(participation);
    return participation;
  }

  public claimActivityReward(userId: string, input: ClaimActivityRewardRequest): ActivityReward {
    const reward = this.activityRewards.find((item) => item.id === input.reward_id && item.activity_id === input.activity_id);
    const index = this.activityParticipations.findIndex((item) => item.user_id === userId && item.activity_id === input.activity_id);
    if (!reward || index < 0) {
      throw notFound;
    }
    this.activityParticipations[index] = { ...this.activityParticipations[index], reward_claimed: true };
    if (reward.reward_type === 'points') {
      const member = this.getUserMember(userId);
      const updated = { ...member, points: member.points + reward.reward_qty };
      this.updateUserMember(updated);
      this.logPoints(userId, reward.reward_qty, updated.points, 'activity', reward.reward_name);
    }
    return reward;
  }

  private addUserItem(userId: string, itemType: ItemType, delta: number): number {
    const key = `${userId}:${itemType}`;
    const current = this.userItems.get(key) ?? { user_id: userId, item_type: itemType, quantity: 0 };
    const quantity = Math.max(0, current.quantity + delta);
    this.userItems.set(key, { ...current, quantity });
    return quantity;
  }

  private isItemType(value: string): value is ItemType {
    return ['hint_card', 'see_through', 'pity_inherit', 'specify_voucher', 'ten_draw_ticket', 'free_draw'].includes(value);
  }

  private getOrCreateBattlePass(userId: string): BattlePass {
    const current = this.battlePasses.get(userId);
    if (current) {
      return { ...current, claimed_levels: [...current.claimed_levels] };
    }
    const created: BattlePass = {
      user_id: userId,
      season_id: this.battlePassSeason?.id ?? 1,
      pass_type: 'free',
      level: 1,
      xp: 0,
      total_xp: 0,
      claimed_levels: [],
      updated_at: nowISO(),
    };
    this.battlePasses.set(userId, created);
    return created;
  }

  private getAssistProgress(userId: string, assistType: AssistType): AssistProgress {
    const key = `${userId}:${assistType}`;
    const current = this.assistProgress.get(key);
    if (current) {
      return { ...current };
    }
    const target = assistType === 'free_draw' ? 3 : assistType === 'pity_reduce' ? 5 : 2;
    const created: AssistProgress = {
      inviter_id: userId,
      assist_type: assistType,
      target_count: target,
      current: 0,
      claimed: false,
      expires_at: addDays(1),
      created_at: nowISO(),
    };
    this.assistProgress.set(key, created);
    return created;
  }

  private teamInfo(teamId: string): TeamInfo {
    const team = this.teams.get(teamId);
    if (!team) {
      throw notFound;
    }
    const members = this.teamMembers.filter((member) => member.team_id === teamId);
    const captain = this.users.get(team.captain_id);
    const reward: TeamReward | undefined =
      team.current_draws >= team.goal_draws
        ? { team_id: team.id, captain_id: team.captain_id, reward_type: 'points', reward_qty: 100, description: '组队目标达成奖励' }
        : undefined;
    return { team, members, captain_name: captain?.nickname ?? team.captain_id, remaining_hours: hoursUntil(team.expires_at), reward };
  }

  private getOrCreatePuzzleProgress(userId: string, template: PuzzleTemplate): PuzzleProgress {
    const key = `${userId}:${template.id}`;
    const current = this.puzzleProgress.get(key);
    if (current) {
      return { ...current, collected: [...current.collected] };
    }
    const collected = template.total_pieces > 1 ? [0, 1] : [0];
    const created: PuzzleProgress = {
      user_id: userId,
      template_id: template.id,
      collected,
      total_pieces: template.total_pieces,
      is_completed: false,
    };
    this.puzzleProgress.set(key, created);
    return created;
  }

  private seed(): void {
    const now = Date.now();
    const startsAt = new Date(now - 24 * 60 * 60 * 1000).toISOString();
    const endsAt = new Date(now + 60 * 24 * 60 * 60 * 1000).toISOString();

    const campaigns: readonly Campaign[] = [
      {
        id: 'camp_launch_001',
        name: '夏季开门红抽奖活动',
        slug: 'summer-launch',
        status: 'online',
        starts_at: startsAt,
        ends_at: new Date(now + 30 * 24 * 60 * 60 * 1000).toISOString(),
        daily_draw_limit: 3,
        requires_phone_login: false,
        miss_weight: 86,
        banner_image_url: '',
        campaign_summary: '新用户登录即可参与，中奖后进入发奖队列，支持后台配置库存和概率。',
      },
      {
        id: 'series_starry_001',
        name: '星空系列',
        slug: 'starry-night',
        status: 'online',
        starts_at: startsAt,
        ends_at: endsAt,
        daily_draw_limit: 10,
        requires_phone_login: false,
        miss_weight: 72,
        banner_image_url: '',
        campaign_summary: '收集星光、月色与银河，集齐普通款和隐藏款可解锁限定奖励。',
        pity_config: {
          enabled: true,
          soft_pity_n: 20,
          pity_factor: 0.08,
          hard_pity_n: 60,
          target_prize: 'star_09',
        },
      },
      {
        id: 'series_cat_001',
        name: '猫咪系列',
        slug: 'cute-cats',
        status: 'online',
        starts_at: startsAt,
        ends_at: new Date(now + 45 * 24 * 60 * 60 * 1000).toISOString(),
        daily_draw_limit: 8,
        requires_phone_login: false,
        miss_weight: 68,
        banner_image_url: '',
        campaign_summary: '超萌猫咪盲盒，集齐全部款式可以解锁隐藏版布偶猫王。',
      },
    ];

    for (const campaign of campaigns) {
      this.campaignsById.set(campaign.id, campaign);
    }

    this.prizesByCampaign.set('camp_launch_001', [
      { id: 'prize_001', campaign_id: 'camp_launch_001', name: '88元红包', level: 'S', stock: 8, probability_weight: 2, status: 'active' },
      { id: 'prize_002', campaign_id: 'camp_launch_001', name: '20元优惠券', level: 'A', stock: 60, probability_weight: 18, status: 'active' },
      { id: 'prize_003', campaign_id: 'camp_launch_001', name: '品牌周边礼盒', level: 'B', stock: 20, probability_weight: 8, status: 'active' },
    ]);
    this.prizesByCampaign.set('series_starry_001', [
      { id: 'star_01', campaign_id: 'series_starry_001', name: '繁星点点', level: 'common', stock: 500, probability_weight: 15, status: 'active' },
      { id: 'star_02', campaign_id: 'series_starry_001', name: '月光如水', level: 'common', stock: 500, probability_weight: 15, status: 'active' },
      { id: 'star_03', campaign_id: 'series_starry_001', name: '银河之泪', level: 'common', stock: 400, probability_weight: 14, status: 'active' },
      { id: 'star_04', campaign_id: 'series_starry_001', name: '流星划过', level: 'common', stock: 400, probability_weight: 12, status: 'active' },
      { id: 'star_05', campaign_id: 'series_starry_001', name: '极光之舞', level: 'common', stock: 300, probability_weight: 10, status: 'active' },
      { id: 'star_06', campaign_id: 'series_starry_001', name: '星云之眼', level: 'common', stock: 300, probability_weight: 6, status: 'active' },
      { id: 'star_07', campaign_id: 'series_starry_001', name: '星月传说', level: 'rare', stock: 100, probability_weight: 15, status: 'active' },
      { id: 'star_08', campaign_id: 'series_starry_001', name: '北极光', level: 'rare', stock: 80, probability_weight: 10, status: 'active' },
      { id: 'star_09', campaign_id: 'series_starry_001', name: '宇宙之心', level: 'secret', stock: 10, probability_weight: 2, status: 'active' },
      { id: 'star_10', campaign_id: 'series_starry_001', name: '星辰大海', level: 'limited', stock: 3, probability_weight: 1, status: 'active' },
    ]);
    this.prizesByCampaign.set('series_cat_001', [
      { id: 'cat_01', campaign_id: 'series_cat_001', name: '英短蓝猫', level: 'common', stock: 600, probability_weight: 16, status: 'active' },
      { id: 'cat_02', campaign_id: 'series_cat_001', name: '橘猫胖胖', level: 'common', stock: 600, probability_weight: 16, status: 'active' },
      { id: 'cat_03', campaign_id: 'series_cat_001', name: '黑猫酷酷', level: 'common', stock: 500, probability_weight: 14, status: 'active' },
      { id: 'cat_04', campaign_id: 'series_cat_001', name: '三花猫', level: 'common', stock: 500, probability_weight: 12, status: 'active' },
      { id: 'cat_05', campaign_id: 'series_cat_001', name: '暹罗猫', level: 'common', stock: 400, probability_weight: 10, status: 'active' },
      { id: 'cat_06', campaign_id: 'series_cat_001', name: '俄罗斯蓝猫', level: 'rare', stock: 120, probability_weight: 18, status: 'active' },
      { id: 'cat_07', campaign_id: 'series_cat_001', name: '布偶猫王', level: 'secret', stock: 8, probability_weight: 2, status: 'active' },
    ]);

    this.shopItemList = [
      {
        id: 'shop_hint_pack',
        name: '提示卡礼包',
        description: '开盒前获得摇盒提示，适合冲刺缺失款式。',
        price_points: 120,
        price_cash: 0,
        item_type: 'hint_card',
        item_qty: 3,
        stock: -1,
        daily_limit: 3,
        category: 'daily',
        is_active: true,
        sort_order: 1,
      },
      {
        id: 'shop_see_through',
        name: '透卡',
        description: '预览下一抽的普通款范围。',
        price_points: 180,
        price_cash: 0,
        item_type: 'see_through',
        item_qty: 1,
        stock: -1,
        daily_limit: 2,
        category: 'item',
        is_active: true,
        sort_order: 2,
      },
      {
        id: 'shop_ten_ticket',
        name: '十连券',
        description: '可用于一次十连开盒。',
        price_points: 900,
        price_cash: 0,
        item_type: 'ten_draw_ticket',
        item_qty: 1,
        stock: 100,
        daily_limit: 1,
        category: 'weekly',
        is_active: true,
        sort_order: 3,
      },
    ];

    this.firstRechargePackList = defaultFirstRechargePacks.map((pack) => cloneFirstRechargePack(pack));

    this.battlePassSeason = {
      id: 1,
      name: '星光收藏季',
      max_level: 20,
      xp_per_level: 100,
      start_at: startsAt,
      end_at: endsAt,
      status: 'active',
    };
    this.battlePassTasks = [
      { id: 1, season_id: 1, type: 'daily', name: '每日开盒', description: '完成 1 次开盒', xp_reward: 20, condition: 'draw_once', target_count: 1 },
      { id: 2, season_id: 1, type: 'weekly', name: '收集达人', description: '新增 3 个收藏', xp_reward: 80, condition: 'collect_three', target_count: 3 },
    ];
    this.battlePassRewards = [
      { level: 1, pass_type: 'free', reward_type: 'points', reward_name: '免费积分', reward_qty: 50 },
      { level: 2, pass_type: 'paid', reward_type: 'draw_ticket', reward_name: '十连券', reward_qty: 1 },
    ];

    this.puzzleTemplates = [
      {
        id: 'puzzle_starry_week',
        name: '星空拼图周挑战',
        campaign_id: 'series_starry_001',
        total_pieces: 6,
        piece_names: ['星芒', '月影', '银河', '流光', '极光', '宇宙'],
        reward_type: 'points',
        reward_qty: 120,
        reward_name: '120积分',
        period_type: 'weekly',
        is_active: true,
        created_at: nowISO(),
      },
      {
        id: 'puzzle_cat_month',
        name: '猫咪拼图月挑战',
        campaign_id: 'series_cat_001',
        total_pieces: 6,
        piece_names: ['耳朵', '胡须', '爪印', '尾巴', '铃铛', '王冠'],
        reward_type: 'points',
        reward_qty: 160,
        reward_name: '160积分',
        period_type: 'monthly',
        is_active: true,
        created_at: nowISO(),
      },
    ];

    this.flashSales = [
      {
        id: 'flash_starry_secret',
        campaign_id: 'series_starry_001',
        name: '宇宙之心限时抢购',
        description: '会员专属隐藏款限量抢购。',
        price_points: 1500,
        total_stock: 10,
        remaining_stock: 10,
        min_vip_level: 'silver',
        min_total_draws: 1,
        start_at: startsAt,
        end_at: endsAt,
        status: 'active',
        created_at: nowISO(),
      },
    ];

    this.activities = [
      {
        id: 'activity_launch_week',
        name: '开服星光周',
        description: '参与活动领取额外积分，抽盒与收集都有奖励。',
        type: 'festival',
        banner_url: '',
        rules: { campaign_id: 'camp_launch_001', checkin_multiplier: 2 },
        sort_order: 1,
        status: 'active',
        start_at: startsAt,
        end_at: endsAt,
        created_at: nowISO(),
        updated_at: nowISO(),
      },
      {
        id: 'activity_up_starry',
        name: '星空隐藏 UP',
        description: '宇宙之心限时概率提升。',
        type: 'up_pool',
        banner_url: '',
        rules: { campaign_id: 'camp_blindbox_001', up_campaign_id: 'camp_blindbox_001', up_prize_id: 'prize_bb_secret', up_multiplier: 3, up_level: 'secret' },
        sort_order: 2,
        status: 'active',
        start_at: startsAt,
        end_at: endsAt,
        created_at: nowISO(),
        updated_at: nowISO(),
      },
    ];
    this.activityRewards = [
      { id: 'reward_launch_points', activity_id: 'activity_launch_week', condition: '参与即可领取', reward_type: 'points', reward_qty: 80, reward_name: '开服积分' },
      { id: 'reward_up_ticket', activity_id: 'activity_up_starry', condition: '参与 UP 活动', reward_type: 'points', reward_qty: 60, reward_name: 'UP 池助力积分' },
    ];
  }

  private ensureAdmin(token: string): void {
    const expiresAt = this.adminSessions.get(token);
    if (!expiresAt || new Date(expiresAt).getTime() < Date.now()) {
      throw adminUnauthorized;
    }
  }

  private quotaKey(userId: string, campaignId: string): string {
    return `${userId}:${campaignId}:${todayKey()}`;
  }

  private logPoints(userId: string, points: number, balance: number, reason: string, remark: string): void {
    this.pointsLogs.unshift({
      id: this.pointsLogs.length + 1,
      user_id: userId,
      points,
      balance,
      reason,
      remark,
      created_at: nowISO(),
    });
  }

  private logLogin(userId: string | undefined, loginType: UserLoginLog['login_type'], loginAccount: string | undefined, success: boolean, failReason?: string): void {
    this.loginLogs.unshift({
      id: this.loginLogs.length + 1,
      user_id: userId,
      login_type: loginType,
      login_account: loginAccount,
      success,
      fail_reason: failReason,
      created_at: nowISO(),
    });
  }

  private logStatus(userId: string, fromStatus: UserStatus, toStatus: UserStatus, reason: string, operatorId?: string): void {
    this.statusLogs.unshift({
      id: this.statusLogs.length + 1,
      user_id: userId,
      from_status: fromStatus,
      to_status: toStatus,
      reason,
      operator_id: operatorId,
      created_at: nowISO(),
    });
  }

  private validatePityConfig(campaignId: string, pityConfig: PityConfig): void {
    const errors: string[] = [];
    this.collectPityConfigIssues(campaignId, pityConfig, errors);
    if (errors.length > 0) {
      throw new AppError('invalid_pity_config', errors.join('；'), 400);
    }
  }

  private collectPityConfigIssues(campaignId: string, pityConfig: PityConfig, errors: string[]): void {
    const prizes = this.prizeList(campaignId);
    const prizeIds = new Set(prizes.map((prize) => prize.id));

    if (pityConfig.hard_pity_n > 0 && pityConfig.hard_pity_n < pityConfig.soft_pity_n) {
      errors.push('硬保底次数必须大于等于软保底起始次数');
    }
    if (pityConfig.target_prize && !prizeIds.has(pityConfig.target_prize)) {
      errors.push('保底目标奖品必须属于当前盲盒');
    }
    if (pityConfig.up_prize_id && !prizeIds.has(pityConfig.up_prize_id)) {
      errors.push('UP 奖品必须属于当前盲盒');
    }
  }

  private findPrize(prizeId: string): Prize {
    for (const prizes of this.prizesByCampaign.values()) {
      const prize = prizes.find((item) => item.id === prizeId);
      if (prize) {
        return prize;
      }
    }
    throw notFound;
  }

  private pointsCostForPrize(level: string): number {
    if (level === 'limited') {
      return 2000;
    }
    if (level === 'secret') {
      return 1200;
    }
    if (level === 'rare') {
      return 500;
    }
    return 200;
  }

  public spendDrawPoints(userId: string, drawCount: number): UserMember {
    const cost = drawCount >= 2 ? drawCount * 95 : drawCount * POINTS_PER_DRAW;
    const member = this.getUserMember(userId);
    if (member.points < cost) {
      throw insufficientPoints;
    }

    const totalSpent = member.total_spent + cost;
    const updated: UserMember = {
      ...member,
      level: memberLevel(totalSpent),
      points: member.points - cost,
      total_draws: member.total_draws + drawCount,
      total_spent: totalSpent,
      updated_at: nowISO(),
    };
    this.updateUserMember(updated);
    this.logPoints(userId, -cost, updated.points, 'draw', `抽盒 ${drawCount} 次`);
    return updated;
  }
}

export async function createMemoryStore(): Promise<MemoryStore> {
  const { admin } = getAppConfig();
  const store = new MemoryStore(admin.user, admin.password);
  await syncAdminConfigWithMysql(store);
  return store;
}
