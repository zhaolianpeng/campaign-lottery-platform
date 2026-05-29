'use client';

/** Admin fulfillment tab — delivery logistics modal fields (MVP). */
export interface DeliveryLogisticsInput {
  readonly carrier: string;
  readonly tracking_no: string;
  readonly note?: string;
}

export function buildDeliveryPayload(input: DeliveryLogisticsInput): Record<string, string> {
  return {
    carrier: input.carrier.trim(),
    tracking_no: input.tracking_no.trim(),
    note: input.note?.trim() ?? '',
  };
}
