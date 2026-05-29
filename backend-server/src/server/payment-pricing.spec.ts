import { describe, expect, it } from '@jest/globals';
import { MEMBERSHIP_CASH_CENTS } from './payment-fulfillment';
import { resolveCheckoutAmountCents } from './payment-pricing';

const mockService = {
  shopItems: () => [{ id: 'shop-1', name: '道具', price_cash: 990 }],
  firstRechargePacks: () => [{ id: 'pack-1', name: '首充', cash_price: 600 }],
  getDeliveryShippingFeeCentsByUserId: () => 800,
} as never;

describe('resolveCheckoutAmountCents', () => {
  it('resolves shop item price from catalog', async () => {
    const amount = await resolveCheckoutAmountCents(mockService, 'user-1', 'shop_item', 'shop-1');
    expect(amount).toBe(990);
  });

  it('resolves membership price server-side', async () => {
    const amount = await resolveCheckoutAmountCents(mockService, 'user-1', 'membership', 'monthly');
    expect(amount).toBe(MEMBERSHIP_CASH_CENTS.monthly);
  });
});
