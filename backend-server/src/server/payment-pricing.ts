import type { CardType } from './types';
import { AppError } from './errors';
import type { LotteryService } from './lottery-service';
import { BATTLE_PASS_CASH_CENTS, MEMBERSHIP_CASH_CENTS } from './payment-fulfillment';

export async function resolveCheckoutAmountCents(
  service: LotteryService,
  userId: string,
  businessType: string,
  businessId: string,
): Promise<number> {
  switch (businessType) {
    case 'membership': {
      const cardType = businessId as CardType;
      const amount = MEMBERSHIP_CASH_CENTS[cardType];
      if (amount === undefined) {
        throw new AppError('invalid_membership_type', '无效的月卡类型', 400);
      }
      return amount;
    }
    case 'battle_pass':
      return BATTLE_PASS_CASH_CENTS;
    case 'shop_item': {
      const item = service.shopItems().find((candidate) => candidate.id === businessId);
      if (!item) {
        throw new AppError('shop_item_not_found', '商店商品不存在', 404);
      }
      if (item.price_cash <= 0) {
        throw new AppError('shop_item_not_cash', '该商品不支持现金购买', 400);
      }
      return item.price_cash;
    }
    case 'first_recharge_pack': {
      const pack = service.firstRechargePacks().find((candidate) => candidate.id === businessId);
      if (!pack) {
        throw new AppError('first_recharge_pack_not_found', '首充礼包不存在', 404);
      }
      if (pack.cash_price <= 0) {
        throw new AppError('first_recharge_pack_not_cash', '该礼包不支持现金购买', 400);
      }
      return pack.cash_price;
    }
    case 'points_pack': {
      const match = /^recharge_(\d+)$/.exec(businessId);
      if (!match) {
        throw new AppError('invalid_points_pack', '无效的积分充值标识', 400);
      }
      const amount = Number(match[1]);
      if (!Number.isInteger(amount) || amount <= 0) {
        throw new AppError('invalid_points_pack', '无效的积分充值金额', 400);
      }
      return amount;
    }
    case 'inventory_delivery':
      return service.getDeliveryShippingFeeCentsByUserId(userId, businessId);
    default:
      throw new AppError('unsupported_business_type', `不支持的业务类型: ${businessType}`, 400);
  }
}
