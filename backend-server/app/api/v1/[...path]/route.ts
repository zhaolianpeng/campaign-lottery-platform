import { z } from 'zod';
import { bearerToken, corsHeaders, fail, ok } from '@/server/api-response';
import { getAppConfig } from '@/server/config';
import { notFound } from '@/server/errors';
import { getService } from '@/server/singleton';
import { getOauthUrl, getJssdkConfig } from '@/server/wechat-service';

export const dynamic = 'force-dynamic';

export function OPTIONS(): Response {
  return new Response(null, {
    status: 204,
    headers: corsHeaders(),
  });
}

const guestLoginSchema = z.object({
  nickname: z.string().optional().default(''),
});

const drawSchema = z.object({
  campaign_id: z.string().min(1),
  draw_count: z.number().int().positive().max(10).optional(),
});

const adminLoginSchema = z.object({
  username: z.string().min(1),
  password: z.string().min(1),
});

const campaignMutationSchema = z.object({
  name: z.string().min(1),
  slug: z.string().min(1),
  status: z.enum(['draft', 'online', 'offline', 'soldout']).default('online'),
  starts_at: z.string().min(1),
  ends_at: z.string().min(1),
  daily_draw_limit: z.number().int().nonnegative(),
  miss_weight: z.number().int().nonnegative(),
  banner_image_url: z.string().optional(),
  campaign_summary: z.string().optional(),
  pity_config: z
    .object({
      enabled: z.boolean(),
      soft_pity_n: z.number().int().nonnegative(),
      pity_factor: z.number().nonnegative(),
      hard_pity_n: z.number().int().nonnegative(),
      target_prize: z.string().default(''),
      up_pool_enabled: z.boolean().optional(),
      up_prize_id: z.string().optional(),
      up_multiplier: z.number().optional(),
      up_level: z.enum(['common', 'rare', 'secret', 'limited', 'S', 'A', 'B']).optional(),
      up_start_at: z.string().optional(),
      up_end_at: z.string().optional(),
    })
    .optional(),
});

const prizeMutationSchema = z.object({
  name: z.string().min(1),
  level: z.enum(['common', 'rare', 'secret', 'limited', 'S', 'A', 'B']),
  stock: z.number().int().nonnegative(),
  probability_weight: z.number().int().nonnegative(),
  status: z.enum(['active', 'inactive']).default('active'),
});

const exchangeOfferSchema = z.object({
  have_prize_id: z.string().min(1),
  want_prize_id: z.string().min(1),
});

const redeemSchema = z.object({
  prize_id: z.string().min(1),
});

const fulfillmentMutationSchema = z.object({
  status: z.string().min(1),
  operator_note: z.string().default(''),
});

const deliveryApproveSchema = z.object({
  task_ids: z.array(z.number().int().positive()),
});

const cardSchema = z.object({
  card_type: z.enum(['weekly', 'monthly', 'season']).default('monthly'),
});

const shopBuySchema = z.object({
  shop_item_id: z.string().min(1),
  quantity: z.number().int().positive().optional(),
});

const useItemSchema = z.object({
  item_type: z.enum(['hint_card', 'see_through', 'pity_inherit', 'specify_voucher', 'ten_draw_ticket', 'free_draw']),
  campaign_id: z.string().optional(),
  prize_id: z.string().optional(),
});

const firstRechargeClaimSchema = z.object({
  pack_id: z.string().min(1),
});

const blendSchema = z.object({
  source_prize_id: z.string().min(1),
  campaign_id: z.string().min(1),
});

const shareCardSchema = z.object({
  card_type: z.string().min(1),
  prize_name: z.string().optional(),
  prize_level: z.string().optional(),
});

const assistSchema = z.object({
  assist_type: z.enum(['free_draw', 'pity_reduce', 'craft_boost']),
  helper_id: z.string().optional().default(''),
});

const assistClaimSchema = z.object({
  assist_type: z.enum(['free_draw', 'pity_reduce', 'craft_boost']),
});

const createTeamSchema = z.object({
  name: z.string().optional().default('我的队伍'),
  max_members: z.number().int().min(2).max(5).default(3),
  goal_draws: z.number().int().positive().default(20),
});

const teamJoinSchema = z.object({
  team_id: z.string().min(1),
});

const giftSchema = z.object({
  receiver_id: z.string().min(1),
  prize_id: z.string().min(1),
  campaign_id: z.string().optional(),
});

const receiveGiftSchema = z.object({
  gift_id: z.string().min(1),
});

const puzzleTemplateSchema = z.object({
  template_id: z.string().min(1),
});

const puzzleTeamJoinSchema = z.object({
  team_id: z.string().min(1),
});

const activityClaimSchema = z.object({
  activity_id: z.string().min(1),
  reward_id: z.string().min(1),
});

const wechatCodeSchema = z.object({
  code: z.string().min(1),
});

const wechatPhoneSchema = z.object({
  openid: z.string().min(1),
  encryptedData: z.string().min(1),
  iv: z.string().min(1),
});

const phoneCodeSchema = z.object({
  phone: z.string().regex(/^1[3-9]\d{9}$/),
  scene: z.enum(['register', 'login', 'bind', 'change_mobile']).optional().default('login'),
});

const phoneVerifySchema = phoneCodeSchema.extend({
  code: z.string().min(4).max(8),
});

const profileSchema = z.object({
  nickname: z.string().min(1).max(20).optional(),
  avatar_url: z.string().url().optional(),
  gender: z.enum(['unknown', 'male', 'female', 'other']).optional(),
  birthday: z.string().optional(),
  province: z.string().max(64).optional(),
  city: z.string().max(64).optional(),
  bio: z.string().max(200).optional(),
});

const userStatusSchema = z.object({
  status: z.enum(['pending_phone', 'active', 'frozen', 'disabled', 'cancelled']),
  reason: z.string().min(1).max(255),
});

const pointsAdjustSchema = z.object({
  points: z.number().int(),
  reason: z.string().min(1).max(120),
  remark: z.string().max(255).optional().default(''),
});

type RouteContext = {
  readonly params: { readonly path?: readonly string[] } | Promise<{ readonly path?: readonly string[] }>;
};

async function readJson(request: Request): Promise<unknown> {
  if (!request.body) {
    return {};
  }
  return request.json();
}

async function segments(context: RouteContext): Promise<readonly string[]> {
  const params = await context.params;
  return params.path ?? [];
}

function searchParam(request: Request, name: string): string {
  return new URL(request.url).searchParams.get(name) ?? '';
}

export async function GET(request: Request, context: RouteContext): Promise<Response> {
  try {
    const path = await segments(context);
    const service = getService();
    const token = bearerToken(request);

    if (path.join('/') === 'config/public') {
      const config = getAppConfig();
      return ok('public config', {
        wechat: {
          quick_login_enabled: config.wechat.quickLoginEnabled,
        },
      });
    }
    if (path.join('/') === 'campaigns') {
      return ok('campaign list', service.campaignList());
    }
    if (path.join('/') === 'me') {
      return ok('current user', service.currentUser(token));
    }
    if (path.join('/') === 'me/account') {
      return ok('current user account', service.currentUserAccount(token));
    }
    if (path.join('/') === 'me/draw-records') {
      return ok('user draw records', service.drawRecords(token));
    }
    if (path.join('/') === 'blindbox/campaigns') {
      return ok('campaign list with progress', service.campaignListWithProgress(token));
    }
    if (path[0] === 'blindbox' && path[1] === 'campaigns' && path[3] === 'probabilities' && path[2]) {
      return ok('campaign probabilities', service.campaignProbabilities(path[2]));
    }
    if (path.join('/') === 'blindbox/pity-status') {
      return ok('pity status', service.pityStatus(token, searchParam(request, 'campaign_id')));
    }
    if (path.join('/') === 'blindbox/inventory') {
      return ok('user inventory', service.userInventory(token));
    }
    if (path.join('/') === 'blindbox/series-progress') {
      return ok('series progress', service.seriesProgress(token, searchParam(request, 'campaign_id')));
    }
    if (path.join('/') === 'blindbox/exchange-offers') {
      return ok('exchange offers', service.exchangeOffers());
    }
    if (path.join('/') === 'blindbox/member') {
      return ok('member info', service.userMember(token));
    }
    if (path.join('/') === 'blindbox/points-log') {
      return ok('points log', service.pointsLog(token));
    }
    if (path.join('/') === 'blindbox/leaderboard') {
      const limit = Number(searchParam(request, 'limit') || 20);
      return ok('leaderboard', service.leaderboard(limit));
    }
    if (path.join('/') === 'admin/overview') {
      return ok('admin overview', service.adminOverview(token));
    }
    if (path.join('/') === 'admin/campaigns') {
      return ok('admin campaigns', service.adminCampaigns(token));
    }
    if (path[0] === 'admin' && path[1] === 'campaigns' && path[2] && path.length === 3) {
      return ok('admin campaign', service.adminCampaigns(token).find((campaign) => campaign.id === path[2]) ?? null);
    }
    if (path[0] === 'admin' && path[1] === 'campaigns' && path[2] && path[3] === 'prizes') {
      return ok('admin prizes', service.adminPrizes(token, path[2]));
    }
    if (path[0] === 'admin' && path[1] === 'campaigns' && path[2] && path[3] === 'pity-config') {
      const campaign = service.adminCampaigns(token).find((item) => item.id === path[2]);
      return ok('pity config', campaign?.pity_config ?? null);
    }
    if (path.join('/') === 'admin/fulfillment-tasks' || path.join('/') === 'admin/delivery/pending') {
      return ok('fulfillment tasks', service.fulfillmentTasks(token));
    }
    if (path.join('/') === 'admin/draw-records' || path.join('/') === 'admin/lottery-logs') {
      return ok('draw records', service.adminDrawRecords(token));
    }
    if (path.join('/') === 'admin/statistics') {
      return ok('statistics', service.drawStatistics(token, searchParam(request, 'campaign_id')));
    }
    if (path.join('/') === 'admin/users') {
      return ok('admin users', service.adminUsers(token, {
        page: Number(searchParam(request, 'page') || 1),
        page_size: Number(searchParam(request, 'page_size') || 20),
        keyword: searchParam(request, 'keyword'),
        status: searchParam(request, 'status'),
        register_source: searchParam(request, 'register_source'),
      }));
    }
    if (path[0] === 'admin' && path[1] === 'users' && path[2] && path.length === 3) {
      return ok('admin user detail', service.adminUserDetail(token, path[2]));
    }
    if (path[0] === 'admin' && path[1] === 'users' && path[2] && path[3] === 'points-log') {
      return ok('admin user points log', service.adminUserPointsLog(token, path[2]));
    }
    if (path[0] === 'admin' && path[1] === 'users' && path[2] && path[3] === 'login-logs') {
      return ok('admin user login logs', service.adminUserLoginLogs(token, path[2]));
    }
    if (path[0] === 'admin' && path[1] === 'users' && path[2] && path[3] === 'status-logs') {
      return ok('admin user status logs', service.adminUserStatusLogs(token, path[2]));
    }
    if (path.join('/') === 'battle-pass/info') {
      return ok('battle pass info', service.battlePassInfo(token));
    }
    if (path.join('/') === 'month-card/status') {
      return ok('month card status', service.monthCardStatus(token));
    }
    if (path.join('/') === 'blindbox/my-card') {
      return ok('user card', service.userCard(token));
    }
    if (path[0] === 'blindbox' && path[1] === 'hint' && path[2]) {
      return ok('campaign hint', service.campaignHint(path[2]));
    }
    if (path[0] === 'blindbox' && path[1] === 'up-pool' && path[2]) {
      return ok('up pool info', service.campaignProbabilities(path[2]));
    }
    if (path.join('/') === 'shop/items') {
      return ok('shop items', service.shopItems());
    }
    if (path.join('/') === 'shop/items/inventory') {
      return ok('user items', service.userItems(token));
    }
    if (path.join('/') === 'first-recharge/packs') {
      return ok('first recharge packs', service.firstRechargePacks());
    }
    if (path.join('/') === 'first-recharge/status') {
      return ok('first recharge status', service.firstRechargeStatus(token));
    }
    if (path.join('/') === 'share/cards') {
      return ok('share cards', service.shareCards(token));
    }
    if (path.join('/') === 'share/invitees') {
      return ok('invite records', service.inviteRecords(token));
    }
    if (path.join('/') === 'share/invite-stats') {
      return ok('invite stats', service.inviteStats(token));
    }
    if (path.join('/') === 'share/assist-progress') {
      return ok('assist progress', service.assistProgress(token));
    }
    if (path.join('/') === 'team/my') {
      return ok('my team', service.myTeam(token));
    }
    if (path.join('/') === 'share/gifts/incoming') {
      return ok('incoming gifts', service.incomingGifts(token));
    }
    if (path.join('/') === 'share/gifts/sent') {
      return ok('sent gifts', service.sentGifts(token));
    }
    if (path.join('/') === 'puzzle/templates') {
      return ok('puzzle templates', service.puzzleTemplates(token));
    }
    if (path[0] === 'puzzle' && path[1] === 'progress' && path[2]) {
      return ok('puzzle progress', service.puzzleInfo(token, path[2]));
    }
    if (path.join('/') === 'puzzle/my') {
      return ok('all puzzle progress', service.allPuzzleInfo(token));
    }
    if (path.join('/') === 'puzzle/team/my') {
      return ok('my puzzle teams', service.myPuzzleTeams(token));
    }
    if (path.join('/') === 'flash/list') {
      return ok('flash list', service.flashList(token));
    }
    if (path.join('/') === 'flash/my') {
      return ok('my flash subscriptions', service.myFlashSubscriptions(token));
    }
    if (path.join('/') === 'activities') {
      return ok('activity list', service.activityList(token));
    }
    if (path[0] === 'activities' && path[1]) {
      return ok('activity detail', service.activityInfo(token, path[1]));
    }
    if (path.join('/') === 'auth/wechat/oauth-url') {
      return ok('oauth url', { url: getOauthUrl() });
    }
    if (path.join('/') === 'auth/wechat/jssdk-config') {
      const url = searchParam(request, 'url') || request.headers.get('referer') || '';
      const config = await getJssdkConfig(url);
      return ok('jssdk config', config);
    }

    throw notFound;
  } catch (error) {
    return fail(error);
  }
}

export async function POST(request: Request, context: RouteContext): Promise<Response> {
  try {
    const path = await segments(context);
    const service = getService();
    const token = bearerToken(request);
    const body = await readJson(request);

    if (path.join('/') === 'auth/guest-login') {
      return ok('guest login succeeded', service.guestLogin(guestLoginSchema.parse(body).nickname));
    }
    if (path.join('/') === 'auth/phone-login') {
      const phoneSchema = z.object({ phone: z.string().regex(/^1[3-9]\d{9}$/) });
      return ok('phone login succeeded', service.phoneLogin(phoneSchema.parse(body).phone));
    }
    if (path.join('/') === 'auth/phone/code') {
      const input = phoneCodeSchema.parse(body);
      return ok('phone code sent', service.sendPhoneCode(input.phone, input.scene));
    }
    if (path.join('/') === 'auth/phone/verify') {
      const input = phoneVerifySchema.parse(body);
      return ok('phone verified', service.verifyPhoneCode(input.phone, input.code, input.scene));
    }
    if (path.join('/') === 'auth/logout') {
      service.logout(token);
      return ok('logged out', null);
    }
    if (path.join('/') === 'lottery/draw' || path.join('/') === 'blindbox/draw') {
      return ok('blind box draw completed', service.blindBoxDraw(token, drawSchema.parse(body)));
    }
    if (path.join('/') === 'blindbox/exchange-offers') {
      return ok('exchange offer created', service.createExchangeOffer(token, exchangeOfferSchema.parse(body)), 201);
    }
    if (path[0] === 'blindbox' && path[1] === 'exchange-offers' && path[2] && path[3] === 'accept') {
      return ok('exchange offer accepted', service.acceptExchangeOffer(token, path[2]));
    }
    if (path.join('/') === 'blindbox/redeem') {
      return ok('redeem completed', service.redeemPrize(token, redeemSchema.parse(body)));
    }
    if (path.join('/') === 'blindbox/checkin') {
      return ok('checkin completed', service.dailyCheckIn(token));
    }
    if (path.join('/') === 'blindbox/share-reward') {
      return ok('share reward completed', service.shareReward(token));
    }
    if (path.join('/') === 'blindbox/blend') {
      return ok('blend completed', service.blendPrizes(token, blendSchema.parse(body)));
    }
    if (path.join('/') === 'blindbox/buy-card') {
      return ok('card purchased', service.buyCard(token, cardSchema.parse(body)));
    }
    if (path.join('/') === 'month-card/buy') {
      return ok('month card purchased', service.buyMonthCard(token, cardSchema.parse(body).card_type));
    }
    if (path.join('/') === 'battle-pass/buy') {
      return ok('battle pass purchased', service.buyBattlePass(token));
    }
    if (path[0] === 'battle-pass' && path[1] === 'claim' && path[2]) {
      return ok('battle pass reward claimed', service.claimBattlePassReward(token, Number(path[2])));
    }
    if (path.join('/') === 'shop/buy') {
      return ok('shop item purchased', service.buyShopItem(token, shopBuySchema.parse(body)));
    }
    if (path.join('/') === 'shop/items/use') {
      return ok('shop item used', service.useItem(token, useItemSchema.parse(body)));
    }
    if (path.join('/') === 'first-recharge/claim') {
      return ok('first recharge claimed', service.claimFirstRecharge(token, firstRechargeClaimSchema.parse(body)));
    }
    if (path.join('/') === 'share/card') {
      const input = shareCardSchema.parse(body);
      return ok('share card', service.createShareCard(token, input.card_type, input.prize_name, input.prize_level));
    }
    if (path.join('/') === 'share/invite') {
      return ok('invite link', service.generateInviteLink(token));
    }
    if (path.join('/') === 'share/assist') {
      const input = assistSchema.parse(body);
      return ok('assist recorded', service.assistAction(token, input.assist_type, input.helper_id));
    }
    if (path.join('/') === 'share/assist-claim') {
      return ok('assist reward claimed', service.claimAssist(token, assistClaimSchema.parse(body).assist_type));
    }
    if (path.join('/') === 'team/create') {
      return ok('team created', service.createTeam(token, createTeamSchema.parse(body)));
    }
    if (path.join('/') === 'team/join') {
      return ok('team joined', service.joinTeam(token, teamJoinSchema.parse(body).team_id));
    }
    if (path.join('/') === 'team/leave') {
      service.leaveTeam(token);
      return ok('team left', null);
    }
    if (path.join('/') === 'share/gift') {
      return ok('gift sent', service.sendGift(token, giftSchema.parse(body)));
    }
    if (path.join('/') === 'share/gift/receive') {
      return ok('gift received', service.receiveGift(token, receiveGiftSchema.parse(body).gift_id));
    }
    if (path.join('/') === 'puzzle/compose') {
      return ok('puzzle composed', service.composePuzzle(token, puzzleTemplateSchema.parse(body).template_id));
    }
    if (path.join('/') === 'puzzle/team/create') {
      return ok('puzzle team created', service.createPuzzleTeam(token, puzzleTemplateSchema.parse(body).template_id));
    }
    if (path.join('/') === 'puzzle/team/join') {
      return ok('joined puzzle team', service.joinPuzzleTeam(token, puzzleTeamJoinSchema.parse(body).team_id));
    }
    if (path[0] === 'flash' && path[1] && path[2] === 'subscribe') {
      service.subscribeFlash(token, path[1]);
      return ok('subscribed', null);
    }
    if (path[0] === 'flash' && path[1] && path[2] === 'unsubscribe') {
      service.unsubscribeFlash(token, path[1]);
      return ok('unsubscribed', null);
    }
    if (path[0] === 'flash' && path[1] && path[2] === 'purchase') {
      return ok('purchased', service.purchaseFlash(token, path[1]));
    }
    if (path[0] === 'activities' && path[1] && path[2] === 'join') {
      return ok('joined', service.joinActivity(token, path[1]));
    }
    if (path.join('/') === 'activities/claim') {
      return ok('reward claimed', service.claimActivityReward(token, activityClaimSchema.parse(body)));
    }
    if (path.join('/') === 'admin/login') {
      const input = adminLoginSchema.parse(body);
      return ok('admin login succeeded', service.adminLogin(input.username, input.password));
    }
    if (path.join('/') === 'admin/campaigns') {
      return ok('campaign created', service.createCampaign(token, campaignMutationSchema.parse(body)), 201);
    }
    if (path[0] === 'admin' && path[1] === 'campaigns' && path[2] && path[3] === 'prizes') {
      return ok('prize created', service.createPrize(token, path[2], prizeMutationSchema.parse(body)), 201);
    }
    if (path.join('/') === 'admin/delivery/approve') {
      const input = deliveryApproveSchema.parse(body);
      const results = input.task_ids.map((taskId) =>
        service.updateFulfillmentTask(token, taskId, {
          status: 'fulfilled',
          operator_note: '批量审核通过',
        }),
      );
      return ok('delivery approved', results);
    }
    if (path[0] === 'admin' && path[1] === 'users' && path[2] && path[3] === 'points-adjust') {
      const input = pointsAdjustSchema.parse(body);
      return ok('points adjusted', service.adjustUserPoints(token, path[2], input.points, input.reason, input.remark));
    }
    if (path[0] === 'admin' && path[1] === 'users' && path[2] && path[3] === 'kick-sessions') {
      return ok('user sessions revoked', service.revokeUserSessions(token, path[2]));
    }
    if (path.join('/') === 'auth/wechat/login') {
      const input = wechatCodeSchema.parse(body);
      const result = await service.wechatLogin(input.code);
      return ok('wechat login succeeded', result);
    }
    if (path.join('/') === 'auth/wechat/phone') {
      const input = wechatPhoneSchema.parse(body);
      const result = service.wechatBindPhone(input.openid, input.encryptedData, input.iv);
      return ok('phone bound', result);
    }

    throw notFound;
  } catch (error) {
    return fail(error);
  }
}

export async function PUT(request: Request, context: RouteContext): Promise<Response> {
  try {
    const path = await segments(context);
    const service = getService();
    const token = bearerToken(request);
    const body = await readJson(request);

    if (path[0] === 'admin' && path[1] === 'campaigns' && path[2] && path.length === 3) {
      return ok('campaign updated', service.updateCampaign(token, path[2], campaignMutationSchema.parse(body)));
    }
    if (path[0] === 'admin' && path[1] === 'campaigns' && path[2] && path[3] === 'pity-config') {
      const parsed = campaignMutationSchema.shape.pity_config.unwrap().parse(body);
      return ok('pity config updated', service.updatePityConfig(token, path[2], parsed));
    }
    if (path[0] === 'admin' && path[1] === 'prizes' && path[2]) {
      return ok('prize updated', service.updatePrize(token, path[2], prizeMutationSchema.parse(body)));
    }

    throw notFound;
  } catch (error) {
    return fail(error);
  }
}

export async function PATCH(request: Request, context: RouteContext): Promise<Response> {
  try {
    const path = await segments(context);
    const service = getService();
    const token = bearerToken(request);
    const body = await readJson(request);

    if (path.join('/') === 'me/profile') {
      return ok('profile updated', service.updateCurrentUserProfile(token, profileSchema.parse(body)));
    }
    if (path[0] === 'admin' && path[1] === 'fulfillment-tasks' && path[2]) {
      return ok(
        'fulfillment task updated',
        service.updateFulfillmentTask(token, Number(path[2]), fulfillmentMutationSchema.parse(body)),
      );
    }
    if (path[0] === 'admin' && path[1] === 'users' && path[2] && path[3] === 'status') {
      const input = userStatusSchema.parse(body);
      return ok('user status updated', service.updateUserStatus(token, path[2], input.status, input.reason));
    }

    throw notFound;
  } catch (error) {
    return fail(error);
  }
}

export async function DELETE(request: Request, context: RouteContext): Promise<Response> {
  try {
    const path = await segments(context);
    const service = getService();
    const token = bearerToken(request);

    if (path[0] === 'blindbox' && path[1] === 'exchange-offers' && path[2]) {
      service.cancelExchangeOffer(token, path[2]);
      return ok('exchange offer cancelled', null);
    }
    if (path[0] === 'admin' && path[1] === 'campaigns' && path[2] && path.length === 3) {
      service.deleteCampaign(token, path[2]);
      return ok('campaign deleted', null);
    }
    if (path[0] === 'admin' && path[1] === 'prizes' && path[2]) {
      service.deletePrize(token, path[2]);
      return ok('prize deleted', null);
    }
    if (path[0] === 'admin' && path[1] === 'campaigns' && path[2] && path[3] === 'prizes' && path[4]) {
      service.deletePrize(token, path[4]);
      return ok('prize deleted', null);
    }

    throw notFound;
  } catch (error) {
    return fail(error);
  }
}
