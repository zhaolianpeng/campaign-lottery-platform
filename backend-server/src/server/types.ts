export type PrizeLevel = 'common' | 'rare' | 'secret' | 'limited' | 'S' | 'A' | 'B';
export type CampaignStatus = 'draft' | 'online' | 'offline' | 'soldout';
export type DrawResultType = 'win' | 'miss';
export type MemberLevel = 'normal' | 'silver' | 'gold' | 'diamond';
export type ExchangeOfferStatus = 'pending' | 'matched' | 'completed' | 'cancelled';

export interface User {
  readonly id: string;
  readonly nickname: string;
  readonly created_at: string;
}

export interface Session {
  readonly token: string;
  readonly user_id: string;
  readonly expires_at: string;
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

export interface DrawConfig {
  readonly campaign_id: string;
  readonly draw_count?: number;
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
  readonly collection_reward?: CollectionReward | null;
}

export interface CollectedPrize extends Prize {
  readonly count: number;
}

export interface PrizeSummary {
  readonly prize_id: string;
  readonly prize_name: string;
  readonly prize_level: string;
  readonly stock: number;
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

export interface ExchangeOfferMutation {
  readonly have_prize_id: string;
  readonly want_prize_id: string;
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

export interface RedeemRequest {
  readonly prize_id: string;
}

export interface RedeemResult {
  readonly record_id: string;
  readonly prize_id: string;
  readonly prize_name: string;
  readonly points_cost: number;
  readonly remaining: number;
}

export interface CollectionReward {
  readonly campaign_id: string;
  readonly campaign_name: string;
  readonly reward_type: string;
  readonly reward_name: string;
  readonly reward_prize_id?: string;
  readonly description: string;
}

export interface CheckInResult {
  readonly points_awarded: number;
  readonly streak_days: number;
  readonly is_bonus: boolean;
  readonly new_balance: number;
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

export interface ShareRewardResult {
  readonly points_awarded: number;
  readonly daily_left: number;
  readonly new_balance: number;
}

export interface DrawStatistics {
  readonly total_draws: number;
  readonly total_users: number;
  readonly total_wins: number;
  readonly win_rate: number;
  readonly prize_breakdown: readonly PrizeStatItem[];
}

export interface PrizeStatItem {
  readonly prize_id: string;
  readonly prize_name: string;
  readonly level: string;
  readonly count: number;
  readonly percent: number;
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

export interface CampaignMutation {
  readonly name: string;
  readonly slug: string;
  readonly status: CampaignStatus;
  readonly starts_at: string;
  readonly ends_at: string;
  readonly daily_draw_limit: number;
  readonly miss_weight: number;
  readonly banner_image_url?: string;
  readonly campaign_summary?: string;
  readonly pity_config?: PityConfig;
}

export interface PrizeMutation {
  readonly name: string;
  readonly level: PrizeLevel;
  readonly stock: number;
  readonly probability_weight: number;
  readonly status: 'active' | 'inactive';
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

export interface FulfillmentTaskMutation {
  readonly status: string;
  readonly operator_note: string;
}

export type CardType = 'weekly' | 'monthly' | 'season';
export type ItemType =
  | 'hint_card'
  | 'see_through'
  | 'pity_inherit'
  | 'specify_voucher'
  | 'ten_draw_ticket'
  | 'free_draw';
export type AssistType = 'free_draw' | 'pity_reduce' | 'craft_boost';
export type ActivityType = 'up_pool' | 'discount' | 'festival' | 'checkin_boost' | 'craft_boost' | 'flash_sale';

export interface UserCard {
  readonly id: string;
  readonly user_id: string;
  readonly card_type: CardType;
  readonly price: number;
  readonly started_at: string;
  readonly expires_at: string;
  readonly daily_free_used: number;
  readonly free_date: string;
  readonly created_at: string;
}

export interface BuyCardRequest {
  readonly card_type: CardType;
}

export interface BuyCardResult {
  readonly card_type: CardType;
  readonly expires_at: string;
  readonly price: number;
  readonly points: number;
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

export interface MonthCardPurchaseResult {
  readonly card: UserCard;
  readonly new_points: number;
}

export interface BattlePassSeason {
  readonly id: number;
  readonly name: string;
  readonly max_level: number;
  readonly xp_per_level: number;
  readonly start_at: string;
  readonly end_at: string;
  readonly status: 'upcoming' | 'active' | 'ended';
}

export interface BattlePass {
  readonly user_id: string;
  readonly season_id: number;
  readonly pass_type: 'free' | 'paid';
  readonly level: number;
  readonly xp: number;
  readonly total_xp: number;
  readonly claimed_levels: readonly number[];
  readonly bought_at?: string;
  readonly updated_at: string;
}

export interface BattlePassTask {
  readonly id: number;
  readonly season_id: number;
  readonly type: 'daily' | 'weekly' | 'season';
  readonly name: string;
  readonly description: string;
  readonly xp_reward: number;
  readonly condition: string;
  readonly target_count: number;
}

export interface BattlePassTaskProgress {
  readonly user_id: string;
  readonly task_id: number;
  readonly progress: number;
  readonly completed: boolean;
  readonly completed_at?: string;
}

export interface BattlePassReward {
  readonly level: number;
  readonly pass_type: 'free' | 'paid';
  readonly reward_type: 'points' | 'draw_ticket' | 'prize' | 'title';
  readonly reward_name: string;
  readonly reward_qty: number;
  readonly reward_id?: string;
}

export interface BattlePassInfo {
  readonly season: BattlePassSeason | null;
  readonly user_pass?: BattlePass;
  readonly tasks: readonly BattlePassTask[];
  readonly task_progress?: readonly BattlePassTaskProgress[];
  readonly rewards: readonly BattlePassReward[];
  readonly level_progress: number;
}

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
  readonly expires_at?: string;
  readonly sort_order: number;
}

export interface UserItem {
  readonly user_id: string;
  readonly item_type: ItemType;
  readonly quantity: number;
}

export interface BuyShopItemRequest {
  readonly shop_item_id: string;
  readonly quantity?: number;
}

export interface BuyShopItemResult {
  readonly item_type: ItemType;
  readonly item_name: string;
  readonly quantity: number;
  readonly points_cost: number;
  readonly new_points: number;
  readonly new_qty: number;
}

export interface UseItemRequest {
  readonly item_type: ItemType;
  readonly campaign_id?: string;
  readonly prize_id?: string;
}

export interface PackItem {
  readonly type: string;
  readonly name: string;
  readonly qty: number;
  readonly prize_id?: string;
}

export interface FirstRechargePack {
  readonly id: string;
  readonly name: string;
  readonly price_points: number;
  readonly cash_price: number;
  readonly items: readonly PackItem[];
  readonly description: string;
  readonly sort_order: number;
}

export interface UserFirstRecharge {
  readonly user_id: string;
  readonly claimed: readonly string[];
}

export interface ClaimFirstRechargeRequest {
  readonly pack_id: string;
}

export interface ClaimFirstRechargeResult {
  readonly pack_id: string;
  readonly pack_name: string;
  readonly items: readonly PackItem[];
  readonly new_points: number;
}

export interface BlendRequest {
  readonly source_prize_id: string;
  readonly campaign_id: string;
}

export interface BlendResult {
  readonly source_prize_id: string;
  readonly source_prize_name: string;
  readonly source_level: string;
  readonly result_prize_id: string;
  readonly result_prize_name: string;
  readonly result_level: string;
  readonly remaining_src: number;
}

export interface HintMessage {
  readonly type: string;
  readonly content: string;
}

export interface ShareCard {
  readonly id: string;
  readonly user_id: string;
  readonly card_type: string;
  readonly title: string;
  readonly description: string;
  readonly image_url?: string;
  readonly prize_name?: string;
  readonly prize_level?: string;
  readonly invite_link?: string;
  readonly created_at: string;
}

export interface InviteRecord {
  readonly id: string;
  readonly inviter_id: string;
  readonly invitee_id: string;
  readonly created_at: string;
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
  readonly created_at: string;
}

export interface AssistClaimResult {
  readonly assist_type: AssistType;
  readonly reward_type: string;
  readonly description: string;
}

export interface CreateTeamRequest {
  readonly name: string;
  readonly max_members: number;
  readonly goal_draws: number;
}

export interface Team {
  readonly id: string;
  readonly captain_id: string;
  readonly name: string;
  readonly max_members: number;
  readonly goal_draws: number;
  readonly current_draws: number;
  readonly starts_at: string;
  readonly expires_at: string;
  readonly status: 'recruiting' | 'active' | 'completed' | 'expired';
  readonly created_at: string;
  readonly completed_at?: string;
}

export interface TeamMember {
  readonly team_id: string;
  readonly user_id: string;
  readonly nickname?: string;
  readonly draws: number;
  readonly joined_at: string;
}

export interface TeamReward {
  readonly team_id: string;
  readonly captain_id: string;
  readonly reward_type: string;
  readonly reward_qty: number;
  readonly description: string;
}

export interface TeamInfo {
  readonly team: Team | null;
  readonly members: readonly TeamMember[];
  readonly captain_name: string;
  readonly remaining_hours: number;
  readonly reward?: TeamReward;
}

export interface SendGiftRequest {
  readonly receiver_id: string;
  readonly prize_id: string;
  readonly campaign_id?: string;
}

export interface GiftRecord {
  readonly id: string;
  readonly giver_id: string;
  readonly receiver_id: string;
  readonly prize_id: string;
  readonly prize_name: string;
  readonly prize_level: string;
  readonly fee_points: number;
  readonly status: 'sent' | 'received' | 'expired';
  readonly created_at: string;
  readonly received_at?: string;
  readonly expires_at: string;
}

export interface ReceiveGiftResult {
  readonly gift_id: string;
  readonly prize_name: string;
  readonly prize_level: string;
  readonly new_item_id?: string;
}

export interface PuzzleTemplate {
  readonly id: string;
  readonly name: string;
  readonly campaign_id: string;
  readonly total_pieces: number;
  readonly piece_names: readonly string[];
  readonly reward_type: string;
  readonly reward_id?: string;
  readonly reward_qty: number;
  readonly reward_name: string;
  readonly period_type: string;
  readonly is_active: boolean;
  readonly created_at: string;
}

export interface PuzzleProgress {
  readonly user_id: string;
  readonly template_id: string;
  readonly collected: readonly number[];
  readonly total_pieces: number;
  readonly is_completed: boolean;
  readonly completed_at?: string;
  readonly team_id?: string;
}

export interface PuzzleTeam {
  readonly id: string;
  readonly template_id: string;
  readonly captain_id: string;
  readonly members: readonly string[];
  readonly shared: readonly number[];
  readonly total_pieces: number;
  readonly is_completed: boolean;
  readonly created_at: string;
}

export interface PuzzleInfo {
  readonly template: PuzzleTemplate;
  readonly progress: PuzzleProgress;
  readonly collected_names: readonly string[];
  readonly missing_names: readonly string[];
  readonly progress_percent: number;
}

export interface ComposePuzzleResult {
  readonly template_id: string;
  readonly template_name: string;
  readonly reward_type: string;
  readonly reward_name: string;
  readonly reward_qty: number;
}

export interface FlashSale {
  readonly id: string;
  readonly campaign_id: string;
  readonly name: string;
  readonly description: string;
  readonly price_points: number;
  readonly total_stock: number;
  readonly remaining_stock: number;
  readonly min_vip_level: MemberLevel;
  readonly min_total_draws: number;
  readonly start_at: string;
  readonly end_at: string;
  readonly status: 'upcoming' | 'active' | 'ended';
  readonly created_at: string;
}

export interface FlashSubscription {
  readonly user_id: string;
  readonly flash_id: string;
  readonly created_at: string;
}

export interface FlashPurchaseResult {
  readonly flash_id: string;
  readonly flash_name: string;
  readonly success: boolean;
  readonly message: string;
  readonly item_name?: string;
}

export interface FlashListInfo {
  readonly flash: FlashSale;
  readonly subscribed: boolean;
  readonly purchasable: boolean;
}

export interface ActivityRules {
  readonly up_prize_id?: string;
  readonly up_multiplier?: number;
  readonly up_level?: string;
  readonly up_campaign_id?: string;
  readonly discount_rate?: number;
  readonly discount_target?: string;
  readonly checkin_multiplier?: number;
  readonly craft_boost_rate?: number;
  readonly gift_pack_id?: string;
  readonly flash_id?: string;
}

export interface Activity {
  readonly id: string;
  readonly name: string;
  readonly description: string;
  readonly type: ActivityType;
  readonly banner_url?: string;
  readonly rules: ActivityRules;
  readonly sort_order: number;
  readonly status: 'draft' | 'active' | 'paused' | 'ended';
  readonly start_at: string;
  readonly end_at: string;
  readonly created_at: string;
  readonly updated_at: string;
}

export interface ActivityParticipation {
  readonly id: string;
  readonly user_id: string;
  readonly activity_id: string;
  readonly data?: string;
  readonly reward_claimed: boolean;
  readonly joined_at: string;
}

export interface ActivityReward {
  readonly id: string;
  readonly activity_id: string;
  readonly condition: string;
  readonly reward_type: string;
  readonly reward_qty: number;
  readonly reward_name: string;
  readonly reward_id?: string;
}

export interface ActivityListInfo {
  readonly activity: Activity;
  readonly joined: boolean;
  readonly can_claim: boolean;
  readonly rewards?: readonly ActivityReward[];
}

export interface ClaimActivityRewardRequest {
  readonly activity_id: string;
  readonly reward_id: string;
}
