import type { CardType } from './types';
import { AppError, paymentForbidden, paymentNotPaid, unauthorized } from './errors';
import { getPaymentModule, isPaymentEnabled } from './payment-gateway';
import { getService } from './singleton';

export const MEMBERSHIP_CASH_CENTS: Record<CardType, number> = {
  weekly: 990,
  monthly: 2800,
  season: 6800,
};

export const BATTLE_PASS_CASH_CENTS = 6800;

export interface PaymentFulfillmentResult {
  readonly order_no: string;
  readonly status: string;
  readonly business_type: string;
  readonly business_id: string;
  readonly already_fulfilled: boolean;
  readonly result: unknown;
}

export async function fulfillPaymentOrder(token: string, orderNo: string): Promise<PaymentFulfillmentResult> {
  if (!token) {
    throw unauthorized;
  }
  if (!isPaymentEnabled()) {
    throw new AppError('payment_disabled', '支付功能未启用', 503);
  }

  const service = await getService();
  const user = service.currentUser(token);
  const payment = getPaymentModule();
  const order = payment.getOrder(orderNo);

  if (order.userId !== user.id) {
    throw paymentForbidden;
  }

  if (order.status === 'fulfilled') {
    return {
      order_no: orderNo,
      status: order.status,
      business_type: order.businessType,
      business_id: order.businessId,
      already_fulfilled: true,
      result: null,
    };
  }

  if (order.status !== 'paid' && order.status !== 'fulfilling') {
    throw paymentNotPaid;
  }

  payment.beginFulfillment(orderNo);

  let result: unknown;
  try {
    result = await runBusinessFulfillment(service, user.id, order.businessType, order.businessId, order.amountCents);
    payment.completeFulfillment(orderNo);
  } catch (error) {
    payment.getOrder(orderNo);
    throw error;
  }

  const fulfilled = payment.getOrder(orderNo);
  return {
    order_no: orderNo,
    status: fulfilled.status,
    business_type: order.businessType,
    business_id: order.businessId,
    already_fulfilled: false,
    result,
  };
}

async function runBusinessFulfillment(
  service: Awaited<ReturnType<typeof getService>>,
  userId: string,
  businessType: string,
  businessId: string,
  amountCents: number,
): Promise<unknown> {
  switch (businessType) {
    case 'first_recharge_pack':
      return service.claimFirstRechargeByUserId(userId, businessId);

    case 'membership': {
      const cardType = businessId as CardType;
      const expected = MEMBERSHIP_CASH_CENTS[cardType];
      if (expected !== undefined && amountCents !== expected) {
        throw new AppError('payment_amount_mismatch', '月卡支付金额不匹配', 400);
      }
      return service.grantMonthCardByUserId(userId, cardType);
    }

    case 'shop_item':
      return service.grantShopItemByUserId(userId, { shop_item_id: businessId, quantity: 1 });

    case 'battle_pass':
      if (amountCents !== BATTLE_PASS_CASH_CENTS) {
        throw new AppError('payment_amount_mismatch', '战令支付金额不匹配', 400);
      }
      return service.grantBattlePassByUserId(userId);

    case 'points_pack':
      return service.grantPointsPackByUserId(userId, businessId, amountCents);

    case 'inventory_delivery':
      return service.fulfillDeliveryRequestByUserId(userId, businessId, amountCents);

    default:
      throw new AppError('unsupported_business_type', `不支持的业务类型: ${businessType}`, 400);
  }
}
