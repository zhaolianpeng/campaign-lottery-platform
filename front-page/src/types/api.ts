export type PrizeLevel = 'common' | 'rare' | 'secret' | 'limited' | 'S' | 'A' | 'B';
export type CampaignStatus = 'draft' | 'online' | 'offline' | 'soldout';
export type DrawResultType = 'win' | 'miss';
export type MemberLevel = 'normal' | 'silver' | 'gold' | 'diamond';
export type ExchangeOfferStatus = 'pending' | 'matched' | 'completed' | 'cancelled';
export type UserStatus = 'pending_phone' | 'active' | 'frozen' | 'disabled' | 'cancelled';
export type RegisterSource = 'wechat' | 'mobile' | 'guest' | 'admin_import';

export interface User {
  readonly id: string;
  readonly nickname: string;
  readonly mobile?: string;
  readonly phone?: string;
  readonly avatar_url?: string;
  readonly status?: UserStatus;
  readonly register_source?: RegisterSource;
  readonly mobile_verified_at?: string;
  readonly last_login_at?: string;
  readonly created_at: string;
  readonly updated_at?: string;
}

export interface UserProfile {
  readonly user_id: string;
  readonly gender?: 'unknown' | 'male' | 'female' | 'other';
  readonly birthday?: string;
  readonly province?: string;
  readonly city?: string;
  readonly bio?: string;
  readonly created_at: string;
  readonly updated_at: string;
}

export interface UserAccount {
  readonly user: User;
  readonly profile?: UserProfile;
  readonly member: UserMember;
  readonly cash_balance: number;
  readonly frozen_balance: number;
  readonly status: UserStatus;
}

export interface UserLoginLog {
  readonly id: number;
  readonly user_id?: string;
  readonly login_type: string;
  readonly login_account?: string;
  readonly success: boolean;
  readonly fail_reason?: string;
  readonly created_at: string;
}

export interface UserStatusLog {
  readonly id: number;
  readonly user_id: string;
  readonly from_status: UserStatus;
  readonly to_status: UserStatus;
  readonly reason: string;
  readonly operator_id?: string;
  readonly created_at: string;
}

export interface AdminUserListItem {
  readonly id: string;
  readonly nickname: string;
  readonly mobile?: string;
  readonly avatar_url?: string;
  readonly status: UserStatus;
  readonly register_source: RegisterSource;
  readonly member_level: MemberLevel;
  readonly points_balance: number;
  readonly cash_balance: number;
  readonly total_draws: number;
  readonly total_spent: number;
  readonly last_login_at?: string;
  readonly created_at: string;
}

export interface AdminUserListResult {
  readonly items: readonly AdminUserListItem[];
  readonly page: number;
  readonly page_size: number;
  readonly total: number;
}

export interface AdminUserDetail {
  readonly user: User;
  readonly profile?: UserProfile;
  readonly member: UserMember;
  readonly cash_balance: number;
  readonly frozen_balance: number;
  readonly identities: readonly { readonly openid: string; readonly phone?: string; readonly nickname?: string; readonly avatar?: string }[];
  readonly recent_draws: readonly DrawRecord[];
  readonly points_logs: readonly UserPointsLog[];
  readonly login_logs: readonly UserLoginLog[];
  readonly status_logs: readonly UserStatusLog[];
}

export interface PityConfig {
  readonly enabled: boolean;
  readonly soft_pity_n: number;
  readonly pity_factor: number;
  readonly hard_pity_n: number;
  readonly target_prize: string;
  readonly up_pool_enabled?: boolean;
  readonly up_prize_id?: string;
  readonly up_multiplier?: number;
  readonly up_level?: PrizeLevel;
  readonly up_start_at?: string;
  readonly up_end_at?: string;
}

export interface Campaign {
  readonly id: string;
  readonly name: string;
  readonly slug: string;
  readonly status: CampaignStatus;
  readonly starts_at: string;
  readonly ends_at: string;
  readonly daily_draw_limit: number;
  readonly miss_weight: number;
  readonly banner_image_url: string;
  readonly campaign_summary: string;
  readonly pity_config?: PityConfig;
}

export interface Prize {
  readonly id: string;
  readonly campaign_id: string;
  readonly name: string;
  readonly level: PrizeLevel;
  readonly stock: number;
  readonly probability_weight: number;
  readonly status: 'active' | 'inactive';
  readonly image_url?: string;
  readonly sort_order?: number;
  readonly display_prob?: string;
}

export interface CampaignPublishValidation {
  readonly campaign_id: string;
  readonly campaign_name: string;
  readonly prize_count: number;
  readonly active_prize_count: number;
  readonly total_stock: number;
  readonly total_weight: number;
  readonly can_publish: boolean;
  readonly errors: readonly string[];
  readonly warnings: readonly string[];
}

export interface PrizeSummary {
  readonly prize_id: string;
  readonly prize_name: string;
  readonly prize_level: string;
  readonly stock: number;
}

export interface CollectedPrize extends Prize {
  readonly count: number;
}

export interface SeriesProgress {
  readonly campaign_id: string;
  readonly campaign_name: string;
  readonly total_items: number;
  readonly collected_items: number;
  readonly progress_percent: number;
  readonly duplicates: number;
  readonly collected_prizes: readonly CollectedPrize[];
  readonly missing_prizes: readonly PrizeSummary[];
}

export interface CampaignListItem {
  readonly campaign: Campaign;
  readonly prizes: readonly Prize[];
  readonly progress?: SeriesProgress;
}

export interface PityStatus {
  readonly consecutive_misses: number;
  readonly pity_multiplier: number;
  readonly soft_pity_n: number;
  readonly hard_pity_n: number;
  readonly misses_to_hard_pity: number;
  readonly has_up_pool_guarantee?: boolean;
}

export interface SingleDrawResult {
  readonly record_id: string;
  readonly prize_id?: string;
  readonly prize_name: string;
  readonly prize_level: string;
  readonly prize_image_url?: string;
  readonly is_win: boolean;
  readonly is_hard_pity?: boolean;
  readonly is_new?: boolean;
  readonly is_up_pool_win?: boolean;
}

export interface BlindBoxDrawResult {
  readonly draws: readonly SingleDrawResult[];
  readonly remaining_chances: number;
  readonly pity_status?: PityStatus;
}

export interface UserInventory {
  readonly id: string;
  readonly user_id: string;
  readonly prize_id: string;
  readonly prize_name: string;
  readonly prize_level: string;
  readonly campaign_id: string;
  readonly source: 'draw' | 'exchange' | 'redeem' | 'collection_reward';
  readonly created_at: string;
}

export interface ExchangeOffer {
  readonly id: string;
  readonly user_id: string;
  readonly user_nickname?: string;
  readonly have_prize_id: string;
  readonly have_prize_name: string;
  readonly want_prize_id: string;
  readonly want_prize_name: string;
  readonly status: ExchangeOfferStatus;
  readonly created_at: string;
}

export interface UserMember {
  readonly user_id: string;
  readonly level: MemberLevel;
  readonly points: number;
  readonly total_draws: number;
  readonly total_spent: number;
  readonly created_at: string;
  readonly updated_at: string;
}

export interface UserPointsLog {
  readonly id: number;
  readonly user_id: string;
  readonly points: number;
  readonly balance: number;
  readonly reason: string;
  readonly remark: string;
  readonly created_at: string;
}

export interface LeaderboardEntry {
  readonly rank: number;
  readonly user_id: string;
  readonly nickname: string;
  readonly collected_count: number;
  readonly total_count: number;
  readonly progress_percent: number;
  readonly series_completed: number;
}

export interface DrawRecord {
  readonly id: string;
  readonly campaign_id: string;
  readonly user_id: string;
  readonly prize_id?: string;
  readonly prize_name: string;
  readonly result: DrawResultType;
  readonly drawn_at: string;
  readonly chance_after: number;
}

export interface FulfillmentTask {
  readonly id: number;
  readonly draw_record_id: string;
  readonly user_id: string;
  readonly prize_id: string;
  readonly status: string;
  readonly payload_json: string;
  readonly operator_note: string;
  readonly created_at: string;
  readonly updated_at: string;
  readonly fulfilled_at?: string;
}

export interface AdminOverview {
  readonly total_users: number;
  readonly total_draws: number;
  readonly total_wins: number;
  readonly campaigns: readonly Campaign[];
  readonly prize_summaries: readonly PrizeSummary[];
  readonly recent_draws: readonly DrawRecord[];
  readonly user_draw_balance: Record<string, number>;
}

export type CardType = 'weekly' | 'monthly' | 'season';
export type ItemType = 'hint_card' | 'see_through' | 'pity_inherit' | 'specify_voucher' | 'ten_draw_ticket' | 'free_draw';
export type AssistType = 'free_draw' | 'pity_reduce' | 'craft_boost';

export interface ShopItem {
  readonly id: string;
  readonly name: string;
  readonly description: string;
  readonly price_points: number;
  readonly price_cash: number;
  readonly item_type: ItemType;
  readonly item_qty: number;
  readonly stock: number;
  readonly daily_limit: number;
  readonly category: string;
  readonly is_active: boolean;
  readonly sort_order: number;
}

export interface UserItem {
  readonly user_id: string;
  readonly item_type: ItemType;
  readonly quantity: number;
}

export interface FirstRechargePack {
  readonly id: string;
  readonly name: string;
  readonly price_points: number;
  readonly cash_price: number;
  readonly description: string;
  readonly sort_order: number;
  readonly items: readonly { readonly type: string; readonly name: string; readonly qty: number }[];
}

export interface UserFirstRecharge {
  readonly user_id: string;
  readonly claimed: readonly string[];
}

export interface MonthCardStatus {
  readonly has_card: boolean;
  readonly card_type?: CardType;
  readonly free_draws: number;
  readonly draw_discount: number;
  readonly expires_at?: string;
  readonly days_left: number;
  readonly today_free_used: number;
}

export interface BattlePassInfo {
  readonly season: { readonly id: number; readonly name: string; readonly max_level: number } | null;
  readonly user_pass?: { readonly level: number; readonly pass_type: string; readonly claimed_levels: readonly number[] };
  readonly tasks: readonly { readonly id: number; readonly name: string; readonly description: string; readonly xp_reward: number }[];
  readonly rewards: readonly { readonly level: number; readonly pass_type: string; readonly reward_name: string; readonly reward_qty: number }[];
  readonly level_progress: number;
}

export interface InviteStats {
  readonly total_invites: number;
  readonly total_assists: number;
  readonly completed_assists: number;
  readonly free_draws_earned: number;
}

export interface AssistProgress {
  readonly inviter_id: string;
  readonly assist_type: AssistType;
  readonly target_count: number;
  readonly current: number;
  readonly claimed: boolean;
  readonly expires_at: string;
}

export interface TeamInfo {
  readonly team: { readonly id: string; readonly name: string; readonly goal_draws: number; readonly current_draws: number; readonly status: string } | null;
  readonly members: readonly { readonly user_id: string; readonly nickname?: string; readonly draws: number }[];
  readonly captain_name: string;
  readonly remaining_hours: number;
}

export interface GiftRecord {
  readonly id: string;
  readonly giver_id: string;
  readonly receiver_id: string;
  readonly prize_id: string;
  readonly prize_name: string;
  readonly prize_level: string;
  readonly status: string;
  readonly created_at: string;
}

export interface PuzzleTemplate {
  readonly id: string;
  readonly name: string;
  readonly campaign_id: string;
  readonly total_pieces: number;
  readonly piece_names: readonly string[];
  readonly reward_name: string;
  readonly reward_qty: number;
}

export interface PuzzleInfo {
  readonly template: PuzzleTemplate;
  readonly progress: { readonly collected: readonly number[]; readonly total_pieces: number; readonly is_completed: boolean };
  readonly collected_names: readonly string[];
  readonly missing_names: readonly string[];
  readonly progress_percent: number;
}

export interface FlashListInfo {
  readonly flash: {
    readonly id: string;
    readonly name: string;
    readonly description: string;
    readonly price_points: number;
    readonly remaining_stock: number;
    readonly status: string;
  };
  readonly subscribed: boolean;
  readonly purchasable: boolean;
}

export interface ActivityListInfo {
  readonly activity: {
    readonly id: string;
    readonly name: string;
    readonly description: string;
    readonly type: string;
    readonly status: string;
  };
  readonly joined: boolean;
  readonly can_claim: boolean;
  readonly rewards?: readonly { readonly id: string; readonly reward_name: string; readonly reward_qty: number; readonly condition: string }[];
}
