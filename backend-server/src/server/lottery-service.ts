import { randomInt } from 'node:crypto';
import { campaignInactive, noDrawChances, wechatPhoneRequired } from './errors';
import { MemoryStore } from './memory-store';
import { PityTracker, ProbabilityEngine, type PityEngineConfig } from './probability';
import { exchangeCode, getUserInfo, decryptPhoneNumber } from './wechat-service';
import type {
  AdminOverview,
  ActivityListInfo,
  ActivityParticipation,
  ActivityReward,
  AdminUserDetail,
  AdminUserListResult,
  AssistClaimResult,
  AssistProgress,
  AssistType,
  BattlePass,
  BattlePassInfo,
  BlendRequest,
  BlendResult,
  BuyCardRequest,
  BuyCardResult,
  BuyShopItemRequest,
  BuyShopItemResult,
  BlindBoxDrawResult,
  CardType,
  Campaign,
  CampaignListItem,
  CampaignMutation,
  CheckInResult,
  ClaimActivityRewardRequest,
  ClaimFirstRechargeRequest,
  ClaimFirstRechargeResult,
  ComposePuzzleResult,
  CreateTeamRequest,
  DrawConfig,
  DrawRecord,
  DrawStatistics,
  ExchangeOffer,
  ExchangeOfferMutation,
  FirstRechargePack,
  FlashListInfo,
  FlashPurchaseResult,
  FlashSubscription,
  FulfillmentTask,
  FulfillmentTaskMutation,
  GiftRecord,
  HintMessage,
  InviteRecord,
  InviteStats,
  LeaderboardEntry,
  MonthCardPurchaseResult,
  MonthCardStatus,
  PityConfig,
  PityStatus,
  Prize,
  PrizeMutation,
  PuzzleInfo,
  PuzzleTeam,
  RedeemRequest,
  RedeemResult,
  SeriesProgress,
  SendGiftRequest,
  ShareCard,
  ShareRewardResult,
  SingleDrawResult,
  ShopItem,
  TeamInfo,
  UseItemRequest,
  UserInventory,
  UserItem,
  UserMember,
  UserPointsLog,
  User,
  UserAccount,
  WechatLoginResult,
  PhoneCodeResult,
  UserLoginLog,
  UserProfileMutation,
  UserStatus,
  UserStatusLog,
} from './types';

export class LotteryService {
  private readonly pityTracker = new PityTracker();

  public constructor(private readonly store: MemoryStore) {}

  public guestLogin(nickname: string): ReturnType<MemoryStore['createGuestSession']> {
    return this.store.createGuestSession(nickname);
  }

  public async wechatLogin(code: string): Promise<WechatLoginResult> {
    const tokenResult = await exchangeCode(code);
    const openid = tokenResult.openid;

    const existing = this.store.findWechatUserByOpenid(openid);
    if (existing) {
      const session = this.store.createSession(existing.user_id);
      if (tokenResult.session_key) {
        this.store.storeWechatSessionKey(openid, tokenResult.session_key);
      }
      return {
        openid,
        user_id: existing.user_id,
        token: session.token,
        nickname: existing.nickname ?? '',
        need_phone: !existing.phone,
        is_new: false,
      };
    }

    const userInfo = await getUserInfo(tokenResult.access_token, openid);
    const created = this.store.createWechatUser(openid, '', userInfo.nickname, userInfo.headimgurl);

    if (tokenResult.session_key) {
      this.store.storeWechatSessionKey(openid, tokenResult.session_key);
    }

    return {
      openid,
      user_id: created.user.id,
      token: created.session.token,
      nickname: created.user.nickname,
      need_phone: created.user.status === 'pending_phone',
      is_new: true,
    };
  }

  public wechatBindPhone(openid: string, encryptedData: string, iv: string): { readonly phone: string; readonly user: User } {
    const sessionKey = this.store.getWechatSessionKey(openid);
    if (!sessionKey) {
      throw wechatPhoneRequired;
    }
    const phone = decryptPhoneNumber(sessionKey, encryptedData, iv);
    const user = this.store.bindWechatPhone(openid, phone);
    return { phone, user };
  }

  public phoneLogin(phone: string): ReturnType<MemoryStore['createPhoneUser']> {
    const existing = this.store.findUserByPhone(phone);
    if (existing) {
      const session = this.store.createSession(existing.id);
      return { user: existing, session };
    }
    return this.store.createPhoneUser(phone);
  }

  public sendPhoneCode(phone: string, scene?: string): PhoneCodeResult {
    return this.store.sendPhoneCode(phone, scene);
  }

  public verifyPhoneCode(phone: string, code: string, scene?: string): ReturnType<MemoryStore['verifyPhoneCode']> {
    return this.store.verifyPhoneCode(phone, code, scene);
  }

  public campaignList(): readonly CampaignListItem[] {
    return this.store.campaigns().map((campaign) => ({
      campaign,
      prizes: this.store.prizeList(campaign.id),
    }));
  }

  public currentUser(token: string): User {
    return this.store.userFromToken(token);
  }

  public updateCurrentUserProfile(token: string, input: UserProfileMutation): UserAccount {
    const user = this.store.userFromToken(token);
    return this.store.updateUserProfile(user.id, input);
  }

  public currentUserAccount(token: string): UserAccount {
    const user = this.store.userFromToken(token);
    return this.store.userAccount(user.id);
  }

  public logout(token: string): void {
    this.store.logout(token);
  }

  public campaignListWithProgress(token: string): readonly CampaignListItem[] {
    const user = token ? this.safeUserFromToken(token) : null;
    return this.store.campaigns().map((campaign) => ({
      campaign,
      prizes: this.store.prizeList(campaign.id),
      progress: user ? this.store.getSeriesProgress(user.id, campaign.id, campaign.name) : undefined,
    }));
  }

  public campaignProbabilities(campaignId: string): Record<string, unknown> {
    const campaign = this.store.getCampaign(campaignId);
    const prizes = this.store.prizeList(campaignId);
    const totalWeight =
      campaign.miss_weight +
      prizes
        .filter((prize) => prize.status === 'active' && prize.probability_weight > 0)
        .reduce((sum, prize) => sum + prize.probability_weight, 0);
    const pityConfig = this.publicPityConfig(campaign);

    return {
      campaign,
      prizes: prizes.map((prize) => ({
        ...prize,
        base_prob: totalWeight > 0 ? `${((prize.probability_weight / totalWeight) * 100).toFixed(2)}%` : '0.00%',
      })),
      pity_config: pityConfig,
      miss_weight: campaign.miss_weight,
    };
  }

  public blindBoxDraw(token: string, config: DrawConfig): BlindBoxDrawResult {
    const user = this.assetUserFromToken(token);
    const campaign = this.store.getCampaign(config.campaign_id);
    const now = Date.now();
    if (
      campaign.status !== 'online' ||
      now < new Date(campaign.starts_at).getTime() ||
      now > new Date(campaign.ends_at).getTime()
    ) {
      throw campaignInactive;
    }

    const remaining = this.store.checkDrawQuota(user.id, campaign.id, campaign.daily_draw_limit);
    const requestedCount = config.draw_count && config.draw_count > 0 ? config.draw_count : 1;
    const drawCount = Math.min(requestedCount, remaining);
    if (drawCount <= 0) {
      throw noDrawChances;
    }

    this.store.spendDrawPoints(user.id, drawCount);
    this.store.deductDrawQuota(user.id, campaign.id, drawCount);

    const prizes = this.store
      .prizeList(campaign.id)
      .filter((prize) => prize.status === 'active' && prize.stock > 0 && prize.probability_weight > 0);
    const pity = this.enginePityConfig(campaign, prizes);
    const engine = new ProbabilityEngine(
      campaign.miss_weight,
      prizes.map((prize) => ({
        id: prize.id,
        weight: this.weightWithUPPool(campaign, prize),
        level: prize.level,
      })),
    );

    const probabilityResults = engine.drawMultiple(drawCount, pity, this.pityTracker, user.id, campaign.id);
    const upPoolActive = this.isUPPoolActive(campaign);
    const draws: SingleDrawResult[] = [];

    for (const probabilityResult of probabilityResults) {
      let prizeId = probabilityResult.prizeId;
      let isUPPoolWin = probabilityResult.isUPPoolWin;

      if (upPoolActive && prizeId && probabilityResult.prizeLevel === campaign.pity_config?.up_level) {
        const hasGuarantee = this.pityTracker.get(user.id, campaign.id).hasUPPoolGuarantee;
        if (hasGuarantee) {
          prizeId = campaign.pity_config?.up_prize_id ?? prizeId;
          isUPPoolWin = true;
          this.pityTracker.setUPPoolGuarantee(user.id, campaign.id, false);
        } else if (randomInt(2) === 0) {
          prizeId = campaign.pity_config?.up_prize_id ?? prizeId;
          isUPPoolWin = true;
        } else {
          this.pityTracker.setUPPoolGuarantee(user.id, campaign.id, true);
        }
      }

      const record = prizeId
        ? this.store.createDrawRecord(user.id, campaign.id, prizeId)
        : this.store.createMissRecord(user.id, campaign.id);
      const prize = prizeId ? this.store.prizeList(campaign.id).find((item) => item.id === prizeId) : undefined;
      draws.push({
        record_id: record.id,
        prize_id: prizeId || undefined,
        prize_name: record.prize_name,
        prize_level: prize?.level ?? probabilityResult.prizeLevel,
        prize_image_url: prize?.image_url,
        is_win: record.result === 'win',
        is_hard_pity: probabilityResult.isHardPity || undefined,
        is_new: prizeId ? this.isFirstPrize(user.id, prizeId) : undefined,
        is_up_pool_win: isUPPoolWin || undefined,
      });
    }

    return {
      draws,
      remaining_chances: this.store.checkDrawQuota(user.id, campaign.id, campaign.daily_draw_limit),
      pity_status: this.pityStatus(token, campaign.id),
      collection_reward: null,
    };
  }

  public pityStatus(token: string, campaignId: string): PityStatus {
    const user = this.store.userFromToken(token);
    const campaign = this.store.getCampaign(campaignId);
    const state = this.pityTracker.get(user.id, campaignId);
    const pityConfig = this.publicPityConfig(campaign);
    const hardPityN = pityConfig.hard_pity_n;

    return {
      consecutive_misses: state.consecutiveMisses,
      pity_multiplier: state.pityMultiplier,
      soft_pity_n: pityConfig.soft_pity_n,
      hard_pity_n: hardPityN,
      misses_to_hard_pity: hardPityN > 0 ? Math.max(0, hardPityN - state.consecutiveMisses) : 0,
      has_up_pool_guarantee: state.hasUPPoolGuarantee,
    };
  }

  public userInventory(token: string): readonly UserInventory[] {
    const user = this.store.userFromToken(token);
    return this.store.getUserInventory(user.id);
  }

  public seriesProgress(token: string, campaignId: string): SeriesProgress {
    const user = this.store.userFromToken(token);
    const campaign = this.store.getCampaign(campaignId);
    return this.store.getSeriesProgress(user.id, campaignId, campaign.name);
  }

  public exchangeOffers(): readonly ExchangeOffer[] {
    return this.store.exchangeOffersList();
  }

  public createExchangeOffer(token: string, input: ExchangeOfferMutation): ExchangeOffer {
    const user = this.assetUserFromToken(token);
    return this.store.createExchangeOffer(user.id, input);
  }

  public cancelExchangeOffer(token: string, offerId: string): void {
    const user = this.store.userFromToken(token);
    this.store.cancelExchangeOffer(user.id, offerId);
  }

  public acceptExchangeOffer(token: string, offerId: string): ExchangeOffer {
    const user = this.assetUserFromToken(token);
    return this.store.acceptExchangeOffer(user.id, offerId);
  }

  public userMember(token: string): UserMember {
    const user = this.store.userFromToken(token);
    return this.store.getUserMember(user.id);
  }

  public pointsLog(token: string): readonly UserPointsLog[] {
    const user = this.store.userFromToken(token);
    return this.store.getPointsLog(user.id);
  }

  public redeemPrize(token: string, input: RedeemRequest): RedeemResult {
    const user = this.assetUserFromToken(token);
    return this.store.redeemPrize(user.id, input);
  }

  public dailyCheckIn(token: string): CheckInResult {
    const user = this.assetUserFromToken(token);
    return this.store.dailyCheckIn(user.id);
  }

  public shareReward(token: string): ShareRewardResult {
    const user = this.assetUserFromToken(token);
    return this.store.shareReward(user.id);
  }

  public leaderboard(limit: number): readonly LeaderboardEntry[] {
    return this.store.leaderboard(limit);
  }

  public drawRecords(token: string): readonly DrawRecord[] {
    const user = this.store.userFromToken(token);
    return this.store.userDrawRecords(user.id);
  }

  public adminLogin(username: string, password: string): { readonly token: string } {
    return { token: this.store.adminLogin(username, password) };
  }

  public adminOverview(token: string): AdminOverview {
    return this.store.adminOverview(token);
  }

  public adminCampaigns(token: string): readonly Campaign[] {
    return this.store.adminCampaigns(token);
  }

  public createCampaign(token: string, input: CampaignMutation): Campaign {
    return this.store.createCampaign(token, input);
  }

  public updateCampaign(token: string, campaignId: string, input: CampaignMutation): Campaign {
    return this.store.updateCampaign(token, campaignId, input);
  }

  public updatePityConfig(token: string, campaignId: string, input: PityConfig): Campaign {
    return this.store.updatePityConfig(token, campaignId, input);
  }

  public deleteCampaign(token: string, campaignId: string): void {
    this.store.deleteCampaign(token, campaignId);
  }

  public adminPrizes(token: string, campaignId: string): readonly Prize[] {
    return this.store.adminPrizes(token, campaignId);
  }

  public createPrize(token: string, campaignId: string, input: PrizeMutation): Prize {
    return this.store.createPrize(token, campaignId, input);
  }

  public updatePrize(token: string, prizeId: string, input: PrizeMutation): Prize {
    return this.store.updatePrize(token, prizeId, input);
  }

  public deletePrize(token: string, prizeId: string): void {
    this.store.deletePrize(token, prizeId);
  }

  public fulfillmentTasks(token: string): readonly FulfillmentTask[] {
    return this.store.fulfillmentTasks(token);
  }

  public updateFulfillmentTask(
    token: string,
    taskId: number,
    input: FulfillmentTaskMutation,
  ): FulfillmentTask {
    return this.store.updateFulfillmentTask(token, taskId, input);
  }

  public adminDrawRecords(token: string): readonly DrawRecord[] {
    return this.store.adminDrawRecords(token);
  }

  public drawStatistics(token: string, campaignId: string): DrawStatistics {
    return this.store.drawStatistics(token, campaignId);
  }

  public adminUsers(token: string, query: { readonly page?: number; readonly page_size?: number; readonly keyword?: string; readonly status?: string; readonly register_source?: string }): AdminUserListResult {
    return this.store.adminUsers(token, query);
  }

  public adminUserDetail(token: string, userId: string): AdminUserDetail {
    return this.store.adminUserDetail(token, userId);
  }

  public updateUserStatus(token: string, userId: string, status: UserStatus, reason: string): User {
    return this.store.updateUserStatus(token, userId, status, reason);
  }

  public adjustUserPoints(token: string, userId: string, points: number, reason: string, remark = ''): UserMember {
    return this.store.adjustUserPoints(token, userId, points, reason, remark);
  }

  public adminUserPointsLog(token: string, userId: string): readonly UserPointsLog[] {
    return this.store.adminUserPointsLog(token, userId);
  }

  public adminUserLoginLogs(token: string, userId: string): readonly UserLoginLog[] {
    return this.store.adminUserLoginLogs(token, userId);
  }

  public adminUserStatusLogs(token: string, userId: string): readonly UserStatusLog[] {
    return this.store.adminUserStatusLogs(token, userId);
  }

  public revokeUserSessions(token: string, userId: string): { readonly revoked: number } {
    return { revoked: this.store.revokeUserSessions(token, userId) };
  }

  public blendPrizes(token: string, input: BlendRequest): BlendResult {
    const user = this.store.userFromToken(token);
    return this.store.blendPrizes(user.id, input);
  }

  public campaignHint(campaignId: string): HintMessage {
    return this.store.campaignHint(campaignId);
  }

  public buyCard(token: string, input: BuyCardRequest): BuyCardResult {
    const user = this.assetUserFromToken(token);
    return this.store.buyCard(user.id, input);
  }

  public userCard(token: string): ReturnType<MemoryStore['getUserCard']> {
    const user = this.store.userFromToken(token);
    return this.store.getUserCard(user.id);
  }

  public monthCardStatus(token: string): MonthCardStatus {
    const user = this.store.userFromToken(token);
    return this.store.monthCardStatus(user.id);
  }

  public buyMonthCard(token: string, cardType: CardType): MonthCardPurchaseResult {
    const user = this.assetUserFromToken(token);
    return this.store.buyMonthCard(user.id, cardType);
  }

  public battlePassInfo(token: string): BattlePassInfo {
    const user = this.store.userFromToken(token);
    return this.store.battlePassInfo(user.id);
  }

  public buyBattlePass(token: string): BattlePass {
    const user = this.assetUserFromToken(token);
    return this.store.buyBattlePass(user.id);
  }

  public claimBattlePassReward(token: string, level: number): boolean {
    const user = this.assetUserFromToken(token);
    return this.store.claimBattlePassReward(user.id, level);
  }

  public shopItems(): readonly ShopItem[] {
    return this.store.shopItems();
  }

  public buyShopItem(token: string, input: BuyShopItemRequest): BuyShopItemResult {
    const user = this.assetUserFromToken(token);
    return this.store.buyShopItem(user.id, input);
  }

  public userItems(token: string): readonly UserItem[] {
    const user = this.store.userFromToken(token);
    return this.store.userItemsList(user.id);
  }

  public useItem(token: string, input: UseItemRequest): { readonly item_type: string; readonly remaining: number; readonly message: string } {
    const user = this.assetUserFromToken(token);
    return this.store.useItem(user.id, input);
  }

  public firstRechargePacks(): readonly FirstRechargePack[] {
    return this.store.firstRechargePacks();
  }

  public firstRechargeStatus(token: string): ReturnType<MemoryStore['firstRechargeStatus']> {
    const user = this.store.userFromToken(token);
    return this.store.firstRechargeStatus(user.id);
  }

  public claimFirstRecharge(token: string, input: ClaimFirstRechargeRequest): ClaimFirstRechargeResult {
    const user = this.assetUserFromToken(token);
    return this.store.claimFirstRecharge(user.id, input);
  }

  public createShareCard(token: string, cardType: string, prizeName = '', prizeLevel = ''): ShareCard {
    const user = this.store.userFromToken(token);
    return this.store.createShareCard(user.id, cardType, prizeName, prizeLevel);
  }

  public shareCards(token: string): readonly ShareCard[] {
    const user = this.store.userFromToken(token);
    return this.store.getShareCards(user.id);
  }

  public generateInviteLink(token: string): ShareCard {
    const user = this.store.userFromToken(token);
    return this.store.generateInviteLink(user.id);
  }

  public inviteRecords(token: string): readonly InviteRecord[] {
    const user = this.store.userFromToken(token);
    return this.store.inviteRecordsFor(user.id);
  }

  public inviteStats(token: string): InviteStats {
    const user = this.store.userFromToken(token);
    return this.store.inviteStats(user.id);
  }

  public assistProgress(token: string): Record<AssistType, AssistProgress> {
    const user = this.store.userFromToken(token);
    return this.store.allAssistProgress(user.id);
  }

  public assistAction(token: string, assistType: AssistType, helperId: string): AssistProgress {
    const user = this.store.userFromToken(token);
    return this.store.recordAssist(user.id, assistType, helperId || user.id);
  }

  public claimAssist(token: string, assistType: AssistType): AssistClaimResult {
    const user = this.store.userFromToken(token);
    return this.store.claimAssist(user.id, assistType);
  }

  public createTeam(token: string, input: CreateTeamRequest): TeamInfo {
    const user = this.store.userFromToken(token);
    return this.store.createTeam(user.id, input);
  }

  public joinTeam(token: string, teamId: string): TeamInfo {
    const user = this.store.userFromToken(token);
    return this.store.joinTeam(user.id, teamId);
  }

  public myTeam(token: string): TeamInfo {
    const user = this.store.userFromToken(token);
    return this.store.myTeam(user.id);
  }

  public leaveTeam(token: string): void {
    const user = this.store.userFromToken(token);
    this.store.leaveTeam(user.id);
  }

  public sendGift(token: string, input: SendGiftRequest): GiftRecord {
    const user = this.assetUserFromToken(token);
    return this.store.sendGift(user.id, input);
  }

  public receiveGift(token: string, giftId: string): ReturnType<MemoryStore['receiveGift']> {
    const user = this.assetUserFromToken(token);
    return this.store.receiveGift(user.id, giftId);
  }

  public incomingGifts(token: string): readonly GiftRecord[] {
    const user = this.store.userFromToken(token);
    return this.store.incomingGifts(user.id);
  }

  public sentGifts(token: string): readonly GiftRecord[] {
    const user = this.store.userFromToken(token);
    return this.store.sentGifts(user.id);
  }

  public puzzleTemplates(token: string): readonly ReturnType<MemoryStore['puzzleTemplateList']>[number][] {
    this.store.userFromToken(token);
    return this.store.puzzleTemplateList();
  }

  public puzzleInfo(token: string, templateId: string): PuzzleInfo {
    const user = this.store.userFromToken(token);
    return this.store.puzzleInfo(user.id, templateId);
  }

  public allPuzzleInfo(token: string): readonly PuzzleInfo[] {
    const user = this.store.userFromToken(token);
    return this.store.allPuzzleInfo(user.id);
  }

  public composePuzzle(token: string, templateId: string): ComposePuzzleResult {
    const user = this.assetUserFromToken(token);
    return this.store.composePuzzle(user.id, templateId);
  }

  public createPuzzleTeam(token: string, templateId: string): PuzzleTeam {
    const user = this.store.userFromToken(token);
    return this.store.createPuzzleTeam(user.id, templateId);
  }

  public joinPuzzleTeam(token: string, teamId: string): PuzzleTeam {
    const user = this.store.userFromToken(token);
    return this.store.joinPuzzleTeam(user.id, teamId);
  }

  public myPuzzleTeams(token: string): readonly PuzzleTeam[] {
    const user = this.store.userFromToken(token);
    return this.store.myPuzzleTeams(user.id);
  }

  public flashList(token: string): readonly FlashListInfo[] {
    const user = this.store.userFromToken(token);
    return this.store.flashList(user.id);
  }

  public subscribeFlash(token: string, flashId: string): void {
    const user = this.store.userFromToken(token);
    this.store.subscribeFlash(user.id, flashId);
  }

  public unsubscribeFlash(token: string, flashId: string): void {
    const user = this.store.userFromToken(token);
    this.store.unsubscribeFlash(user.id, flashId);
  }

  public purchaseFlash(token: string, flashId: string): FlashPurchaseResult {
    const user = this.assetUserFromToken(token);
    return this.store.purchaseFlash(user.id, flashId);
  }

  public myFlashSubscriptions(token: string): readonly FlashSubscription[] {
    const user = this.store.userFromToken(token);
    return this.store.myFlashSubscriptions(user.id);
  }

  public activityList(token: string): readonly ActivityListInfo[] {
    const user = this.store.userFromToken(token);
    return this.store.activityList(user.id);
  }

  public activityInfo(token: string, activityId: string): ActivityListInfo {
    const user = this.store.userFromToken(token);
    return this.store.activityInfo(user.id, activityId);
  }

  public joinActivity(token: string, activityId: string): ActivityParticipation {
    const user = this.store.userFromToken(token);
    return this.store.joinActivity(user.id, activityId);
  }

  public claimActivityReward(token: string, input: ClaimActivityRewardRequest): ActivityReward {
    const user = this.assetUserFromToken(token);
    return this.store.claimActivityReward(user.id, input);
  }

  private enginePityConfig(campaign: Campaign, prizes: readonly Prize[]): PityEngineConfig {
    const publicConfig = this.publicPityConfig(campaign);
    const targetPrize =
      prizes.find((prize) => prize.id === publicConfig.target_prize) ??
      prizes.find((prize) => prize.level === 'secret') ??
      prizes[0];

    return {
      enabled: publicConfig.enabled,
      softPityN: publicConfig.soft_pity_n,
      pityFactor: publicConfig.pity_factor,
      hardPityN: publicConfig.hard_pity_n,
      targetPrizeId: targetPrize?.id ?? '',
      targetWeight: targetPrize?.probability_weight ?? 0,
    };
  }

  private publicPityConfig(campaign: Campaign): PityConfig {
    return (
      campaign.pity_config ?? {
        enabled: false,
        soft_pity_n: 0,
        pity_factor: 0,
        hard_pity_n: 0,
        target_prize: '',
      }
    );
  }

  private isUPPoolActive(campaign: Campaign): boolean {
    const config = campaign.pity_config;
    if (!config?.up_pool_enabled || !config.up_prize_id) {
      return false;
    }
    const now = Date.now();
    const start = config.up_start_at ? new Date(config.up_start_at).getTime() : 0;
    const end = config.up_end_at ? new Date(config.up_end_at).getTime() : Number.MAX_SAFE_INTEGER;
    return now >= start && now <= end;
  }

  private weightWithUPPool(campaign: Campaign, prize: Prize): number {
    const config = campaign.pity_config;
    if (this.isUPPoolActive(campaign) && config?.up_prize_id === prize.id) {
      return prize.probability_weight * Math.max(1, config.up_multiplier ?? 1);
    }
    return prize.probability_weight;
  }

  private isFirstPrize(userId: string, prizeId: string): boolean {
    return this.store.getUserInventory(userId).filter((item) => item.prize_id === prizeId).length <= 1;
  }

  private safeUserFromToken(token: string): { readonly id: string } | null {
    try {
      return this.store.userFromToken(token);
    } catch {
      return null;
    }
  }

  private assetUserFromToken(token: string): User {
    const user = this.store.userFromToken(token);
    this.store.assertAssetAllowed(user.id);
    return user;
  }
}
